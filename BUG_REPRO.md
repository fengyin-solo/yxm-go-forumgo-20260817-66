# Bug Reproduction

## Bug 是什么

评论软删除后仍会被普通读取和列表返回，同时帖子回复数没有同步减少。

## 如何触发

运行：

```bash
go test ./internal/service -run TestCommentDeleteHidesCommentAndUpdatesThreadCount -count=20
```

## 错误信息

```text
deleted comment should not be returned by GetByID
```
