# BUG_REPRO

## 问题

审计列表不存在资源返回 500，请求取消后查询仍继续；插入路径错误链断裂；record normalize 未生成 ID 和跟踪标识。

## 根因

文件: internal/audit/adapter/http.go, internal/audit/application/service.go, internal/audit/domain/types.go
符号: Handler.List, Service.List, Record.Normalize
机制: 审计列表错误链断裂且 handler 丢弃 context，not found 被硬编码成 500；记录未生成 ID 和跟踪标识

## 复现命令

```bash
go test ./internal/audit/adapter -run '^TestR009AuditListHttp404$' -count=1
go test ./internal/audit/adapter -run '^TestR009AuditListCancelledQuery$' -count=1
go test ./internal/audit/application -run '^TestR009AuditListMissingMaps404$' -count=1
go test ./internal/audit/application -run '^TestR009AuditRecordMissingSentinel$' -count=1
go test ./internal/audit/application -run '^TestR009AuditRecordAutoID$' -count=1
go test ./internal/audit/application -run '^TestR009AuditRecordAutoTrace$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
