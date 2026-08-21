# BUG_REPRO

## 问题

队列 Handle、Enqueue、Poll、Complete、Fail 丢弃请求上下文，取消信号未传播；`AvailableAt` 边界判断把恰好到期任务判为不可用。

## 根因

文件: internal/queue/application/service.go, internal/queue/domain/types.go
符号: QueueService.Handle, Task.Available
机制: 队列处理与任务操作丢弃请求 context，取消信号不传播；AvailableAt 边界判断把恰好到期任务排除

## 复现命令

```bash
go test ./internal/queue/application -run '^TestR008HandleCancelledCallback$' -count=1
go test ./internal/queue/application -run '^TestR008HandleCompleteContext$' -count=1
go test ./internal/queue/application -run '^TestR008EnqueueRequestContext$' -count=1
go test ./internal/queue/application -run '^TestR008PollRequestContext$' -count=1
go test ./internal/queue/application -run '^TestR008CompleteRequestContext$' -count=1
go test ./internal/queue/application -run '^TestR008FailRequestContext$' -count=1
go test ./internal/queue/application -run '^TestR008TaskReadyAtBoundary$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
