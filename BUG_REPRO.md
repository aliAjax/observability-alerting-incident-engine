# BUG_REPRO

## 问题

规则服务查询不存在的规则返回 500，错误链用 `%v` 包装导致 `errors.Is` 无法识别 not found；暂停状态仍被判定为活跃。

## 根因

文件: internal/rule/application/service.go, internal/rule/domain/types.go
符号: Service.Get, Rule.Active
机制: 规则查询与写路径用 %v 包装错误导致 errors.Is 断链，not found 被映射成系统错误；暂停模式仍被判定为活跃

## 复现命令

```bash
go test ./internal/rule/application -run '^TestR002RuleGetAbsentRule$' -count=1
go test ./internal/rule/application -run '^TestR002RuleCreateBadPayload$' -count=1
go test ./internal/rule/application -run '^TestR002RuleUpdateAbsentRule$' -count=1
go test ./internal/rule/application -run '^TestR002RuleListAbsentRule$' -count=1
go test ./internal/rule/application -run '^TestR002RuleSetModeBadEnum$' -count=1
go test ./internal/rule/domain -run '^TestR002PausedRuleExcluded$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
