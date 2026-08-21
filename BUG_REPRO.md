# BUG_REPRO

## 问题

通知批次并发处理时，processed 计数多个 goroutine 同时读写产生 data race，返回完成数偶发偏小；任务默认状态、最大尝试次数、可用时间和创建时间初始化错误。

## 根因

文件: internal/notification/application/service.go, internal/notification/domain/types.go
符号: Service.ProcessBatch, Task.Normalize
机制: 通知批次 goroutine 并发累加 processed 且 WaitGroup Add 时机错误，出现 data race 和计数缺失；任务状态和时间默认值错误

## 复现命令

```bash
go test -race ./internal/qcbatch_notification -run '^TestR007ConcurrentBatchCount$' -count=1
go test -race ./internal/qcbatch_notification -run '^TestR007TaskMaxAttemptsDefault$' -count=1
go test -race ./internal/qcbatch_notification -run '^TestR007TaskStatusDefault$' -count=1
go test -race ./internal/qcbatch_notification -run '^TestR007TaskReadyTimeDefault$' -count=1
go test -race ./internal/qcbatch_notification -run '^TestR007TaskCreatedAtDefault$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
