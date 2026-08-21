# BUG_REPRO

## 问题

告警历史等查询路径把 not found 当成 500 返回，错误链断裂；已恢复告警不能重新回到 firing，状态转换表缺边。

## 根因

文件: internal/alert/application/service.go, internal/alert/domain/types.go
符号: Service.History, ValidateTransition
机制: 告警历史与各查询路径用 %v 包装错误导致 errors.Is 断链；状态转换表缺失 resolved 到 firing 的合法边

## 复现命令

```bash
go test ./internal/alert/application -run '^TestR004HistoryMissingMaps500$' -count=1
go test ./internal/alert/application -run '^TestR004ResolveMissingMaps500$' -count=1
go test ./internal/alert/application -run '^TestR004AcknowledgeMissingMaps500$' -count=1
go test ./internal/alert/application -run '^TestR004ListMissingMaps500$' -count=1
go test ./internal/alert/application -run '^TestR004ObservationsMissingMaps500$' -count=1
go test ./internal/alert/application -run '^TestR004GetMissingMaps500$' -count=1
go test ./internal/alert/domain -run '^TestR004ResolvedBackToFiringEdge$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
