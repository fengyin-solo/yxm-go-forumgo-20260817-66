# Bug Reproduction

## Bug 是什么

更新版块 slug 时没有按规范化后的 slug 做唯一性检查，带空格或大小写变化的重复 slug 会覆盖已有 slug 索引。

## 如何触发

运行：

```bash
go test ./internal/service -run TestBoardUpdateRejectsDuplicateCanonicalSlug -count=20
```

## 错误信息

```text
updating a board to an existing canonical slug should fail
```
