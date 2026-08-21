# BUG_REPRO

## 问题

批量摄入去重时，共享 map 和结果切片并发读写没有互斥，导致 data race、漏事件；去重指纹又漏掉租户和数值字段，造成误去重。

## 根因

文件: internal/ingestion/application/service.go, internal/ingestion/domain/types.go
符号: Service.dedupe, IngestionEvent.DedupeFingerprint
机制: 批量去重共享 map 和结果切片并发读写没有互斥，出现 data race 并漏事件；指纹丢失租户和数值导致误去重

## 复现命令

```bash
go test -race ./internal/ingestion/application -run '^TestR001ConcurrentIngestKeepsAllKeys$' -count=1
go test -race ./internal/ingestion/application -run '^TestR001FingerprintTenantField$' -count=1
go test -race ./internal/ingestion/application -run '^TestR001FingerprintNumericField$' -count=1
go test -race ./internal/ingestion/application -run '^TestR001SamplingExpectedSlots$' -count=1
go test -race ./internal/ingestion/application -run '^TestR001SamplingDisabledKeepsAll$' -count=1
```

## 验证口径

埋错基线 `.base_snapshot` 上逐条执行上述命令全部失败；修复后 `env/` 上逐条执行全部通过，`go build ./...` 与 `go test ./...` 无新增回归。
