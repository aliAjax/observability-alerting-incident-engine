package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"github.com/observability-alerting/engine/internal/alert/application"
	alertdomain "github.com/observability-alerting/engine/internal/alert/domain"
	"github.com/observability-alerting/engine/internal/common"
	incidentapplication "github.com/observability-alerting/engine/internal/incident/application"
	incidentdomain "github.com/observability-alerting/engine/internal/incident/domain"
	ingestionapplication "github.com/observability-alerting/engine/internal/ingestion/application"
	ingestiondomain "github.com/observability-alerting/engine/internal/ingestion/domain"
	ruleapplication "github.com/observability-alerting/engine/internal/rule/application"
	ruledomain "github.com/observability-alerting/engine/internal/rule/domain"
)

func init() {
	encoding.RegisterCodec(rawCodec{})
}

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *slog.Logger
}

type Dependencies struct {
	Ingestion *ingestionapplication.Service
	Rule      *ruleapplication.Service
	Alert     *application.Service
	Incident  *incidentapplication.Service
	Evaluator *application.Evaluator
}

type serviceServer struct {
	deps   Dependencies
	logger *slog.Logger
}

func NewServer(address string, deps Dependencies, logger *slog.Logger) (*Server, error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen grpc %s: %w", address, err)
	}
	grpcServer := grpc.NewServer(grpc.ForceServerCodec(rawCodec{}))
	srv := &serviceServer{deps: deps, logger: logger}
	grpcServer.RegisterService(&grpc.ServiceDesc{
		ServiceName: "observability.alerting.Engine",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "Ingest", Handler: unaryHandler(srv.ingest)},
			{MethodName: "IngestBatch", Handler: unaryHandler(srv.ingestBatch)},
			{MethodName: "CreateRule", Handler: unaryHandler(srv.createRule)},
			{MethodName: "EvaluateNow", Handler: unaryHandler(srv.evaluateNow)},
			{MethodName: "CreateIncident", Handler: unaryHandler(srv.createIncident)},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "observability-alerting-engine",
	}, srv)
	return &Server{grpcServer: grpcServer, listener: lis, logger: logger}, nil
}

func (s *Server) Start() error {
	s.logger.Info("starting grpc server", "address", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *serviceServer) ingest(ctx context.Context, msg *rawMessage) (any, error) {
	var evt ingestiondomain.IngestionEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		return nil, common.Wrap("decode grpc ingest", err)
	}
	evt.Tenant = common.TenantFrom(ctx)
	out, err := s.deps.Ingestion.Ingest(ctx, evt)
	if err != nil {
		return nil, err
	}
	return &rawMessage{Data: common.MustJSON(out)}, nil
}

func (s *serviceServer) ingestBatch(ctx context.Context, msg *rawMessage) (any, error) {
	var batch ingestiondomain.Batch
	if err := json.Unmarshal(msg.Data, &batch); err != nil {
		return nil, common.Wrap("decode grpc batch", err)
	}
	batch.Tenant = common.TenantFrom(ctx)
	n, err := s.deps.Ingestion.IngestBatch(ctx, batch)
	if err != nil {
		return nil, err
	}
	return &rawMessage{Data: common.MustJSON(map[string]int{"accepted": n})}, nil
}

func (s *serviceServer) createRule(ctx context.Context, msg *rawMessage) (any, error) {
	var rule ruledomain.Rule
	if err := json.Unmarshal(msg.Data, &rule); err != nil {
		return nil, common.Wrap("decode grpc rule", err)
	}
	rule.Tenant = common.TenantFrom(ctx)
	out, err := s.deps.Rule.Create(ctx, rule)
	if err != nil {
		return nil, err
	}
	return &rawMessage{Data: common.MustJSON(out)}, nil
}

func (s *serviceServer) evaluateNow(ctx context.Context, msg *rawMessage) (any, error) {
	var body struct {
		RuleID string `json:"rule_id"`
	}
	_ = json.Unmarshal(msg.Data, &body)
	_ = body
	if s.deps.Evaluator == nil {
		return &rawMessage{Data: common.MustJSON(map[string]any{"evaluated": 0, "matches": 0})}, nil
	}
	result, err := s.deps.Evaluator.EvaluateAll(ctx, time.Now(), 100)
	if err != nil {
		return nil, err
	}
	return &rawMessage{Data: common.MustJSON(result)}, nil
}

func (s *serviceServer) createIncident(ctx context.Context, msg *rawMessage) (any, error) {
	var incident incidentdomain.Incident
	if err := json.Unmarshal(msg.Data, &incident); err != nil {
		return nil, common.Wrap("decode grpc incident", err)
	}
	incident.Tenant = common.TenantFrom(ctx)
	out, err := s.deps.Incident.Create(ctx, incident)
	if err != nil {
		return nil, err
	}
	return &rawMessage{Data: common.MustJSON(out)}, nil
}

func unaryHandler(fn func(context.Context, *rawMessage) (any, error)) func(any, context.Context, func(any) error, grpc.UnaryServerInterceptor) (any, error) {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := &rawMessage{}
		if err := dec(req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return fn(ctx, req)
		}
		info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/observability.alerting.Engine/Unknown"}
		handler := func(ctx context.Context, req any) (any, error) {
			return fn(ctx, req.(*rawMessage))
		}
		return interceptor(ctx, req, info, handler)
	}
}

func (s *serviceServer) String() string {
	return "observability alerting grpc service"
}

var _ = alertdomain.StatusPending
var _ = time.Now
