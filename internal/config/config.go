package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment  string             `yaml:"environment"`
	Tenant       string             `yaml:"tenant"`
	LogLevel     string             `yaml:"log_level"`
	HTTP         HTTPConfig         `yaml:"http"`
	GRPC         GRPCConfig         `yaml:"grpc"`
	Database     DatabaseConfig     `yaml:"database"`
	Evaluator    EvaluatorConfig    `yaml:"evaluator"`
	Notification NotificationConfig `yaml:"notification"`
	Retention    RetentionConfig    `yaml:"retention"`
}

type HTTPConfig struct {
	Address         string        `yaml:"address"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	RateLimitPerSec int           `yaml:"rate_limit_per_sec"`
	Burst           int           `yaml:"burst"`
}

type GRPCConfig struct {
	Address string `yaml:"address"`
}

type DatabaseConfig struct {
	DSN            string        `yaml:"dsn"`
	MaxConnections int           `yaml:"max_connections"`
	MinConnections int           `yaml:"min_connections"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	MigrationPath  string        `yaml:"migration_path"`
	AutoMigrate    bool          `yaml:"auto_migrate"`
	QueryExecMode  string        `yaml:"query_exec_mode"`
}

type EvaluatorConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Interval       time.Duration `yaml:"interval"`
	WorkerCount    int           `yaml:"worker_count"`
	ShardCount     int           `yaml:"shard_count"`
	LockTTL        time.Duration `yaml:"lock_ttl"`
	BatchSize      int           `yaml:"batch_size"`
	DryRunDelivery bool          `yaml:"dry_run_delivery"`
}

type NotificationConfig struct {
	Enabled         bool          `yaml:"enabled"`
	Workers         int           `yaml:"workers"`
	PollInterval    time.Duration `yaml:"poll_interval"`
	DefaultCoolDown time.Duration `yaml:"default_cooldown"`
	MaxAttempts     int           `yaml:"max_attempts"`
}

type RetentionConfig struct {
	AlertHistoryDays int `yaml:"alert_history_days"`
	AuditHistoryDays int `yaml:"audit_history_days"`
}

func Default() Config {
	return Config{
		Environment: "development",
		Tenant:      "default",
		LogLevel:    "info",
		HTTP: HTTPConfig{
			Address:         ":8080",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    20 * time.Second,
			ShutdownTimeout: 15 * time.Second,
			RateLimitPerSec: 200,
			Burst:           400,
		},
		GRPC: GRPCConfig{Address: ":9090"},
		Database: DatabaseConfig{
			DSN:            "postgres://observability:observability@localhost:5432/observability?sslmode=disable",
			MaxConnections: 20,
			MinConnections: 2,
			ConnectTimeout: 10 * time.Second,
			MigrationPath:  "migrations",
			AutoMigrate:    true,
			QueryExecMode:  "simple",
		},
		Evaluator: EvaluatorConfig{
			Enabled:        true,
			Interval:       5 * time.Second,
			WorkerCount:    8,
			ShardCount:     16,
			LockTTL:        30 * time.Second,
			BatchSize:      200,
			DryRunDelivery: false,
		},
		Notification: NotificationConfig{
			Enabled:         true,
			Workers:         4,
			PollInterval:    time.Second,
			DefaultCoolDown: 5 * time.Minute,
			MaxAttempts:     5,
		},
		Retention: RetentionConfig{AlertHistoryDays: 30, AuditHistoryDays: 90},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config: %w", err)
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	cfg.Environment = envString("APP_ENV", cfg.Environment)
	cfg.Tenant = envString("APP_TENANT", cfg.Tenant)
	cfg.LogLevel = envString("LOG_LEVEL", cfg.LogLevel)
	cfg.HTTP.Address = envString("HTTP_ADDRESS", cfg.HTTP.Address)
	cfg.HTTP.ReadTimeout = envDuration("HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout)
	cfg.HTTP.WriteTimeout = envDuration("HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout)
	cfg.HTTP.ShutdownTimeout = envDuration("HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout)
	cfg.HTTP.RateLimitPerSec = envInt("HTTP_RATE_LIMIT_PER_SEC", cfg.HTTP.RateLimitPerSec)
	cfg.HTTP.Burst = envInt("HTTP_BURST", cfg.HTTP.Burst)
	cfg.GRPC.Address = envString("GRPC_ADDRESS", cfg.GRPC.Address)
	cfg.Database.DSN = envString("DATABASE_DSN", cfg.Database.DSN)
	cfg.Database.MaxConnections = envInt("DATABASE_MAX_CONNECTIONS", cfg.Database.MaxConnections)
	cfg.Database.MinConnections = envInt("DATABASE_MIN_CONNECTIONS", cfg.Database.MinConnections)
	cfg.Database.ConnectTimeout = envDuration("DATABASE_CONNECT_TIMEOUT", cfg.Database.ConnectTimeout)
	cfg.Database.MigrationPath = envString("MIGRATION_PATH", cfg.Database.MigrationPath)
	cfg.Database.AutoMigrate = envBool("DATABASE_AUTO_MIGRATE", cfg.Database.AutoMigrate)
	cfg.Evaluator.Enabled = envBool("EVALUATOR_ENABLED", cfg.Evaluator.Enabled)
	cfg.Evaluator.Interval = envDuration("EVALUATOR_INTERVAL", cfg.Evaluator.Interval)
	cfg.Evaluator.WorkerCount = envInt("EVALUATOR_WORKERS", cfg.Evaluator.WorkerCount)
	cfg.Evaluator.ShardCount = envInt("EVALUATOR_SHARDS", cfg.Evaluator.ShardCount)
	cfg.Evaluator.LockTTL = envDuration("EVALUATOR_LOCK_TTL", cfg.Evaluator.LockTTL)
	cfg.Evaluator.BatchSize = envInt("EVALUATOR_BATCH_SIZE", cfg.Evaluator.BatchSize)
	cfg.Notification.Enabled = envBool("NOTIFICATION_ENABLED", cfg.Notification.Enabled)
	cfg.Notification.Workers = envInt("NOTIFICATION_WORKERS", cfg.Notification.Workers)
	cfg.Notification.PollInterval = envDuration("NOTIFICATION_POLL_INTERVAL", cfg.Notification.PollInterval)
	cfg.Notification.DefaultCoolDown = envDuration("NOTIFICATION_DEFAULT_COOLDOWN", cfg.Notification.DefaultCoolDown)
	cfg.Notification.MaxAttempts = envInt("NOTIFICATION_MAX_ATTEMPTS", cfg.Notification.MaxAttempts)
}

func envString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(strings.TrimSpace(v)); err == nil {
			return d
		}
	}
	return fallback
}

func (c Config) Validate() error {
	if c.HTTP.Address == "" {
		return fmt.Errorf("http.address is required")
	}
	if c.GRPC.Address == "" {
		return fmt.Errorf("grpc.address is required")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if c.Evaluator.Enabled && c.Evaluator.Interval <= 0 {
		return fmt.Errorf("evaluator.interval must be positive")
	}
	if c.Notification.Enabled && c.Notification.PollInterval <= 0 {
		return fmt.Errorf("notification.poll_interval must be positive")
	}
	return nil
}
