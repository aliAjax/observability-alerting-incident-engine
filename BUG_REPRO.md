# BUG_REPRO

## 问题

空标签 map 直接合并触发 `assignment to entry in nil map` panic；标签克隆共享底层 map；新建静默规则未初始化 matchers，active 被写成 false。

## 根因

文件: internal/common/labels.go, internal/silence/domain/types.go
符号: Labels.Merge, Silence.Normalize
机制: nil 标签 map 直接写入触发 panic，克隆与匹配零值路径错误；静默规则未初始化 matchers 且 active 被写为 false

## 复现命令

```bash
go test ./internal/common -run '^TestR005MergeEmptyLabels$' -count=1
go test ./internal/common -run '^TestR005CloneLabelsDeepCopy$' -count=1
go test ./internal/common -run '^TestR005EmptyLabelsString$' -count=1
go test ./internal/common -run '^TestR005EmptyMatcherOnEmpty$' -count=1
go test ./internal/silence/application -run '^TestR005SilenceNewMatchers$' -count=1
go test ./internal/silence/application -run '^TestR005SilenceNewActiveFlag$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
