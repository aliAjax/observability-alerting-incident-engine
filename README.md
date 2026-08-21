# Observability Alerting and Incident Response Engine

纯 Go 实现的观测告警与事件响应引擎。系统通过 HTTP 和 gRPC 接收指标、日志事件和监控检查结果，按规则执行阈值、百分比、环比、同比、窗口聚合、多条件组合和抑制规则评估，管理告警生命周期，并将告警路由到通知渠道或事件响应流程。

## 特性

- 数据接入：HTTP/gRPC 指标、日志和检查结果，含校验、采样、去重和路由。
- 规则管理：阈值、百分比、环比、同比、窗口聚合、多条件组合、抑制规则；支持启用、暂停、试运行和版本化。
- 告警评估：按规则定时评估，使用 PostgreSQL 指纹 upsert 保证多副本幂等。
- 告警生命周期：pending、firing、resolved、acknowledged、silenced 状态迁移和历史查询。
- 去重分组：按规则、作用域和标签生成指纹，窗口内更新同一条告警并保留触发记录。
- 通知路由：渠道、模板、重试、冷却、升级策略；失败进入可靠 PostgreSQL 队列。
- 排班与静默：值班表、临时静默、规则级静默；静默期间继续评估但抑制通知。
- 事件响应：告警转事件、绑定责任人、记录动作、自动转派和关闭，处理动作可审计。
- 服务治理：HTTP/gRPC 中间件、超时、限流、优雅停机、`/healthz`、`/readyz`、`/metrics`。

## 架构

```text
cmd/
  server/          服务入口
  grpc-client/     轻量 gRPC 验证客户端
api/
  http/            REST API 与中间件
  grpc/            gRPC 服务与 raw JSON codec
internal/
  common/          共享 ID、标签、分页、错误和 HTTP 工具
  config/          YAML + 环境变量配置
  migration/       数据库迁移执行器
  bootstrap/       依赖装配、调度器、通知 worker、队列 worker
  ingestion/       数据接入领域
  rule/            规则领域
  alert/           告警评估与生命周期
  notification/    通知路由领域
  schedule/        排班领域
  silence/         静默领域
  incident/        事件响应领域
  audit/           审计领域
  queue/           可靠 PostgreSQL 队列
```

每个业务领域均拆分为 `domain`、`application`、`adapter`、`infrastructure`。领域通过接口注入，服务层使用 `context.Context`，基础设施使用 PostgreSQL 持久化所有关键状态。

## 快速启动

要求 Go 1.22+、Docker 和 Docker Compose。

```bash
docker compose up -d postgres
go run ./cmd/server configs/config.yaml
```

服务启动后：

- HTTP: `http://localhost:8080`
- gRPC: `localhost:9090`
- 数据库: `postgres://observability:observability@localhost:55433/observability?sslmode=disable`

运行冒烟脚本：

```bash
./scripts/smoke.sh
./scripts/sample_data.sh
```

停止服务和依赖：

```bash
./scripts/stop.sh
```

## 配置

配置默认值见 `internal/config/config.go`，示例见 `configs/config.yaml`。常用环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `development` | 环境名 |
| `APP_TENANT` | `default` | 默认租户 |
| `HTTP_ADDRESS` | `:8080` | HTTP 监听地址 |
| `GRPC_ADDRESS` | `:9090` | gRPC 监听地址 |
| `DATABASE_DSN` | `postgres://observability:observability@localhost:55433/observability?sslmode=disable` | PostgreSQL DSN |
| `DATABASE_AUTO_MIGRATE` | `true` | 启动时自动执行 migrations |
| `EVALUATOR_ENABLED` | `true` | 是否启用定时评估 |
| `EVALUATOR_INTERVAL` | `5s` | 定时评估间隔 |
| `NOTIFICATION_ENABLED` | `true` | 是否启用通知 worker |
| `LOG_LEVEL` | `info` | 日志级别 |

## REST API 示例

### 健康检查

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/metrics
```

### 创建规则

```bash
curl -X POST http://localhost:8080/api/v1/rules \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "cpu-high",
    "data_source": "demo-api",
    "event_type": "metric",
    "type": "threshold",
    "condition": {"field": "avg", "operator": "gt", "value": 0.7},
    "severity": "critical",
    "labels": {"team": "platform"},
    "group_by": ["region"],
    "channels": []
  }'
```

### 写入指标并评估

```bash
curl -X POST http://localhost:8080/api/v1/ingest/batch \
  -H 'Content-Type: application/json' \
  -d '{
    "source": "demo-api",
    "events": [{
      "type": "metric",
      "labels": {"region": "cn-east"},
      "numeric_value": 0.92
    }]
  }'

curl -X POST http://localhost:8080/api/v1/evaluate
curl http://localhost:8080/api/v1/alerts
```

### 告警生命周期

```bash
ALERT_ID=<alert_id>
curl -X POST http://localhost:8080/api/v1/alerts/$ALERT_ID/acknowledge \
  -H 'Content-Type: application/json' -d '{"actor":"oncall"}'
curl -X POST http://localhost:8080/api/v1/alerts/$ALERT_ID/resolve \
  -H 'Content-Type: application/json' -d '{"reason":"deploy finished"}'
curl "http://localhost:8080/api/v1/alerts/$ALERT_ID/history"
```

### 创建事件和记录动作

```bash
curl -X POST http://localhost:8080/api/v1/incidents \
  -H 'Content-Type: application/json' \
  -d '{"alert_id":"<alert_id>","title":"api latency incident","severity":"critical","assignee":"alice"}'

curl -X POST http://localhost:8080/api/v1/incidents/<incident_id>/actions \
  -H 'Content-Type: application/json' \
  -d '{"action":"rolled back","actor":"alice","comment":"reverted bad deploy"}'
```

### 排班与静默

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H 'Content-Type: application/json' \
  -d '{"name":"platform-oncall","timezone":"Asia/Shanghai","enabled":true,"shifts":[{"assignee":"alice","starts_at":"2026-08-19T00:00:00Z","ends_at":"2026-08-20T00:00:00Z"}]}'

curl -X POST http://localhost:8080/api/v1/silences \
  -H 'Content-Type: application/json' \
  -d '{"rule_id":"<rule_id>","matchers":{"region":"cn-east"},"starts_at":"2026-08-19T15:00:00Z","ends_at":"2026-08-19T16:00:00Z","created_by":"smoke"}'
```

## gRPC 验证

项目提供轻量 raw JSON codec，不依赖外部 protoc。使用内置客户端：

```bash
go run ./cmd/grpc-client -addr localhost:9090 -method EvaluateNow -body '{}'
go run ./cmd/grpc-client -addr localhost:9090 -method Ingest -body '{"source":"demo-api","type":"metric","numeric_value":0.9}'
```

可用方法：`Ingest`、`IngestBatch`、`CreateRule`、`EvaluateNow`、`CreateIncident`。

## 数据库迁移

迁移文件位于 `migrations/`，服务启动时按文件名顺序执行。`DATABASE_AUTO_MIGRATE=false` 时跳过自动迁移。

## 验证方法

1. `go build ./...`
2. `go test ./...`
3. `docker compose up -d postgres`
4. `go run ./cmd/server configs/config.yaml`
5. 按上例调用健康检查、创建规则、写入指标、评估、查询告警、确认告警、创建事件。
6. `./scripts/stop.sh` 关闭服务与容器。

## 主要目录

- `cmd/server/main.go`: 服务入口
- `api/http/router.go`: REST 路由
- `api/grpc/server.go`: gRPC 服务
- `internal/bootstrap/app.go`: 依赖装配和后台任务
- `internal/alert/application/evaluator.go`: 规则评估器
- `migrations/000001_init.sql`: PostgreSQL schema
- `configs/config.yaml`: 示例配置
