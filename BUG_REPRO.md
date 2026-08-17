# Bug Reproduction

## Bug 是什么

HTTP 查询线程列表时没有把 locked 查询参数传入下游过滤条件，导致锁定状态过滤失效。

## 如何触发

运行：

```bash
go test ./internal/httpapi -run TestThreadListLockedQueryExcludesLockedThreads -count=1
```

## 错误信息

```text
want only unlocked thread
```
