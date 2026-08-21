package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	grpcapi "github.com/observability-alerting/engine/api/grpc"
	httpapi "github.com/observability-alerting/engine/api/http"
	alertadapter "github.com/observability-alerting/engine/internal/alert/adapter"
	"github.com/observability-alerting/engine/internal/alert/application"
	alertinfra "github.com/observability-alerting/engine/internal/alert/infrastructure"
	auditadapter "github.com/observability-alerting/engine/internal/audit/adapter"
	auditapplication "github.com/observability-alerting/engine/internal/audit/application"
	auditinfra "github.com/observability-alerting/engine/internal/audit/infrastructure"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/config"
	incidentadapter "github.com/observability-alerting/engine/internal/incident/adapter"
	incidentapplication "github.com/observability-alerting/engine/internal/incident/application"
	incidentinfra "github.com/observability-alerting/engine/internal/incident/infrastructure"
	ingestionadapter "github.com/observability-alerting/engine/internal/ingestion/adapter"
	ingestionapplication "github.com/observability-alerting/engine/internal/ingestion/application"
	ingestioninfra "github.com/observability-alerting/engine/internal/ingestion/infrastructure"
	"github.com/observability-alerting/engine/internal/migration"
	notificationadapter "github.com/observability-alerting/engine/internal/notification/adapter"
	notificationapplication "github.com/observability-alerting/engine/internal/notification/application"
	notificationinfra "github.com/observability-alerting/engine/internal/notification/infrastructure"
	queueapplication "github.com/observability-alerting/engine/internal/queue/application"
	queuedomain "github.com/observability-alerting/engine/internal/queue/domain"
	queueinfra "github.com/observability-alerting/engine/internal/queue/infrastructure"
	ruleadapter "github.com/observability-alerting/engine/internal/rule/adapter"
	ruleapplication "github.com/observability-alerting/engine/internal/rule/application"
	ruleinfra "github.com/observability-alerting/engine/internal/rule/infrastructure"
	scheduleadapter "github.com/observability-alerting/engine/internal/schedule/adapter"
	scheduleapplication "github.com/observability-alerting/engine/internal/schedule/application"
	scheduleinfra "github.com/observability-alerting/engine/internal/schedule/infrastructure"
	silenceadapter "github.com/observability-alerting/engine/internal/silence/adapter"
	silenceapplication "github.com/observability-alerting/engine/internal/silence/application"
	silenceinfra "github.com/observability-alerting/engine/internal/silence/infrastructure"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func NewLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	return &App{cfg: cfg, logger: logger}, nil
}

func (a *App) Run(ctx context.Context) error {
	pool, err := a.connectDB(ctx)
	if err != nil {
		return err
	}
	a.pool = pool
	defer pool.Close()

	if a.cfg.Database.AutoMigrate {
		if err := migration.NewRunner(pool, a.cfg.Database.MigrationPath).Run(ctx); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		a.logger.Info("database migrations applied")
	}

	deps, err := a.buildDependencies(pool)
	if err != nil {
		return err
	}

	httpSrv := httpapi.NewServer(a.cfg, deps.HTTP, a.logger)
	grpcSrv, err := grpcapi.NewServer(a.cfg.GRPC.Address, deps.GRPC, a.logger)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpSrv.Start(); err != nil {
			errCh <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := grpcSrv.Start(); err != nil {
			errCh <- err
		}
	}()

	if a.cfg.Evaluator.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runScheduler(ctx, deps.Evaluator)
		}()
	}
	if a.cfg.Notification.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runNotificationWorker(ctx, deps.Notification)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runEvaluationQueueWorker(ctx, deps.Queue, deps.Evaluator)
		}()
	}

	a.logger.Info("service started", "http", a.cfg.HTTP.Address, "grpc", a.cfg.GRPC.Address, "tenant", a.cfg.Tenant)
	select {
	case <-ctx.Done():
		a.logger.Info("shutdown signal received")
	case err := <-errCh:
		a.logger.Error("server stopped unexpectedly", "error", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("http shutdown failed", "error", err)
	}
	grpcSrv.Stop()
	wg.Wait()
	return nil
}

func (a *App) connectDB(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(a.cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse database dsn: %w", err)
	}
	if a.cfg.Database.QueryExecMode == "simple" {
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	}
	cfg.MaxConns = int32(a.cfg.Database.MaxConnections)
	cfg.MinConns = int32(a.cfg.Database.MinConnections)
	cfg.ConnConfig.ConnectTimeout = a.cfg.Database.ConnectTimeout
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, a.cfg.Database.ConnectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

type dependencies struct {
	HTTP         httpapi.Dependencies
	GRPC         grpcapi.Dependencies
	Evaluator    *application.Evaluator
	Notification *notificationapplication.Service
	Queue        *queueapplication.QueueService
}

func (a *App) buildDependencies(pool *pgxpool.Pool) (dependencies, error) {
	queueRepo := queueinfra.NewPGRepository(pool)
	queue := queueapplication.NewQueueService(queueRepo, common.NewID("worker"), a.cfg.Evaluator.LockTTL)

	ingestionRepo := ingestioninfra.NewPGRepository(pool)
	ingestionSvc := ingestionapplication.NewService(ingestionRepo, queue, a.logger, 0, 5*time.Minute)

	ruleRepo := ruleinfra.NewPGRepository(pool)
	ruleSvc := ruleapplication.NewService(ruleRepo, a.logger)

	alertRepo := alertinfra.NewPGRepository(pool)
	alertSvc := application.NewService(alertRepo, a.logger)

	notifyRepo := notificationinfra.NewPGRepository(pool)
	sender := notificationadapter.NewSender(nil, a.logger)
	notifySvc := notificationapplication.NewService(notifyRepo, sender, a.logger, common.NewID("notify"), a.cfg.Evaluator.LockTTL)

	scheduleRepo := scheduleinfra.NewPGRepository(pool)
	scheduleSvc := scheduleapplication.NewService(scheduleRepo, a.logger)

	silenceRepo := silenceinfra.NewPGRepository(pool)
	silenceSvc := silenceapplication.NewService(silenceRepo, a.logger)

	incidentRepo := incidentinfra.NewPGRepository(pool)
	incidentSvc := incidentapplication.NewService(incidentRepo, a.logger)

	auditRepo := auditinfra.NewPGRepository(pool)
	auditSvc := auditapplication.NewService(auditRepo, a.logger)

	evaluator := application.NewEvaluator(ruleSvc, ingestionRepo, alertSvc, notifySvc, silenceSvc, auditSvc, a.logger, common.NewID("evaluator"))

	deps := dependencies{
		Evaluator:    evaluator,
		Notification: notifySvc,
		Queue:        queue,
		HTTP: httpapi.Dependencies{
			Ingestion:    ingestionadapter.NewHandler(ingestionSvc),
			Rule:         ruleadapter.NewHandler(ruleSvc),
			Alert:        alertadapter.NewHandler(alertSvc),
			Notification: notificationadapter.NewHandler(notifySvc),
			Schedule:     scheduleadapter.NewHandler(scheduleSvc),
			Silence:      silenceadapter.NewHandler(silenceSvc),
			Incident:     incidentadapter.NewHandler(incidentSvc),
			Audit:        auditadapter.NewHandler(auditSvc),
			Evaluator:    evaluatorHTTPHandler{evaluator: evaluator, logger: a.logger},
		},
		GRPC: grpcapi.Dependencies{
			Ingestion: ingestionSvc,
			Rule:      ruleSvc,
			Alert:     alertSvc,
			Incident:  incidentSvc,
			Evaluator: evaluator,
		},
	}
	return deps, nil
}

func (a *App) runScheduler(ctx context.Context, evaluator *application.Evaluator) {
	ticker := time.NewTicker(a.cfg.Evaluator.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			result, err := evaluator.EvaluateAll(ctx, now, a.cfg.Evaluator.BatchSize)
			if err != nil {
				a.logger.Error("scheduled evaluation failed", "error", err)
				continue
			}
			a.logger.Info("scheduled evaluation complete", "evaluated", result.Evaluated, "matches", result.Matches, "failed", result.Failed)
		}
	}
}

func (a *App) runNotificationWorker(ctx context.Context, service *notificationapplication.Service) {
	ticker := time.NewTicker(a.cfg.Notification.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed, err := service.ProcessBatch(ctx, 20)
			if err != nil {
				a.logger.Error("notification worker failed", "error", err)
				continue
			}
			if processed > 0 {
				a.logger.Info("notification worker processed tasks", "processed", processed)
			}
		}
	}
}

func (a *App) runEvaluationQueueWorker(ctx context.Context, queue *queueapplication.QueueService, evaluator *application.Evaluator) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := queue.Handle(ctx, "evaluation", 10, func(ctx context.Context, task queuedomain.Task) error {
				_ = task
				result, err := evaluator.EvaluateAll(ctx, time.Now(), a.cfg.Evaluator.BatchSize)
				if err != nil {
					return err
				}
				_ = result
				return nil
			})
			if err != nil {
				a.logger.Error("evaluation queue worker failed", "error", err)
			}
		}
	}
}

type evaluatorHTTPHandler struct {
	evaluator *application.Evaluator
	logger    *slog.Logger
}

func (h evaluatorHTTPHandler) Run(w http.ResponseWriter, r *http.Request) {
	result, err := h.evaluator.EvaluateAll(r.Context(), time.Now(), 100)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	httpapi.WriteOK(w, result)
}
