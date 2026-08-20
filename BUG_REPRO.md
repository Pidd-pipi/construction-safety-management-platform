# BUG_REPRO

## Bug 是什么
工人资质审核错误链断裂与状态映射错乱：仓储用 %v 丢失 ErrNotFound 使 errors.Is 失效；Review/Submit 未把 not-found 映射 404、冲突错误被 %v 包裹后 errors.As 失效返回 500；handler 误用 errors.Is 导致业务错误全变 500。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r008 -run '^TestCertReviewMissingIdReturns404$' -count=1
go test ./test/r008 -run '^TestCertReviewDuplicateReturns409$' -count=1
go test ./test/r008 -run '^TestCertSubmitUnknownUserReturns404$' -count=1
```

## 错误信息
```
TestCertReviewMissingIdReturns404: review missing status = 500, want 404
TestCertReviewDuplicateReturns409: review approved cert status = 500, want 409
TestCertSubmitUnknownUserReturns404: submit missing user status = 500, want 404
```
