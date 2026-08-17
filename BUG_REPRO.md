# Bug Reproduction

## Bug 是什么

举报状态在 resolve 和 list 路径没有统一做大小写与空白规范化，导致等价状态值被拒绝或过滤不到。

## 如何触发

运行：

```bash
go test ./internal/service -run TestReportStatusIsCanonicalAcrossResolveAndList -count=1
```

## 错误信息

```text
Resolve should accept canonical status with whitespace: invalid input: status must be resolved or dismissed
```
