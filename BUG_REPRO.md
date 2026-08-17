# Bug Reproduction

## Bug 是什么

投票服务只检查评论是否存在，没有排除已经软删除的评论，导致隐藏评论仍能产生投票记录。

## 如何触发

运行：

```bash
go test ./internal/service -run TestVoteRejectsDeletedCommentTarget -count=20
```

## 错误信息

```text
vote on deleted comment should be rejected
```
