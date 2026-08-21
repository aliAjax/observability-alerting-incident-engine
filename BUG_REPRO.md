# BUG_REPRO

## 问题

通知适配层在发送 webhook 和非 webhook 通知时没有传播取消和超时；HTTP 列表与创建接口也把请求上下文换成了 `context.Background()`。

## 根因

文件: internal/notification/adapter/sender.go, internal/notification/adapter/http.go
符号: Sender.Send, Handler.ListTasks
机制: 出站 webhook 与非 webhook 路径不传播取消/超时；HTTP 列表和创建接口丢弃请求 context

## 复现命令

```bash
go test ./internal/notification/adapter -run '^TestR003WebhookCancelStopsSend$' -count=1
go test ./internal/notification/adapter -run '^TestR003EmailCancelStopsSend$' -count=1
go test ./internal/notification/adapter -run '^TestR003TaskListKeepsRequestCtx$' -count=1
go test ./internal/notification/adapter -run '^TestR003ChannelCreateKeepsRequestCtx$' -count=1
go test ./internal/notification/adapter -run '^TestR003ChannelListKeepsRequestCtx$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
