# BUG_REPRO

## 问题

静默匹配查询在请求取消后仍继续，List 丢弃请求上下文；租户和跟踪标识的 context 工具没有正确写入和读取。

## 根因

文件: internal/common/context.go, internal/silence/application/service.go
符号: TenantFrom, Service.IsSilenced
机制: 租户和跟踪标识上下文未写入/读取，静默查询与列表丢弃调用 context，取消和租户隔离失效

## 复现命令

```bash
go test ./internal/silence_qc_context -run '^TestR010TenantContextWrite$' -count=1
go test ./internal/silence_qc_context -run '^TestR010TraceContextWrite$' -count=1
go test ./internal/silence_qc_context -run '^TestR010TraceMissingGenerate$' -count=1
go test ./internal/silence_qc_context -run '^TestR010SilencedCheckContext$' -count=1
go test ./internal/silence_qc_context -run '^TestR010SilenceListContext$' -count=1
go test ./internal/silence_qc_context -run '^TestR010TenantContextRead$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
