# BUG_REPRO

## 问题

incident 状态机在事件升级后执行 resolve 仍停留在 escalated，assign、escalate、close 状态落点不一致；normalize 默认状态和严重级别错误。

## 根因

文件: internal/incident/application/service.go, internal/incident/domain/types.go
符号: Service.Act, Incident.Normalize
机制: 事件升级、resolve、close 与 assign 的跨层状态机落点错位，缺少 escalated→resolved 的转换表边并造成非法迁移；normalize 默认状态和严重级别写错

## 复现命令

```bash
go test ./internal/incident/application -run '^TestR006EscalatedThenResolve$' -count=1
go test ./internal/incident/application -run '^TestR006AssignInProgress$' -count=1
go test ./internal/incident/application -run '^TestR006EscalateStatus$' -count=1
go test ./internal/incident/application -run '^TestR006EscalateClosedAt$' -count=1
go test ./internal/incident/application -run '^TestR006CloseResolved$' -count=1
go test ./internal/incident/application -run '^TestR006ResolveStatus$' -count=1
go test ./internal/incident/application -run '^TestR006NormalizeOpen$' -count=1
go test ./internal/incident/application -run '^TestR006NormalizeSeverity$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
