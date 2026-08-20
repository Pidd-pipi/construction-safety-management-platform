# BUG_REPRO

## Bug 是什么
用户注册与登录错误处理断裂：user_repository 用 %v 包装丢失 ErrNotFound sentinel，errors.Is 判定失效，导致新手机号注册报 500、未注册手机号登录报 500、缺失用户访问 /users/me 报 500（应为 404）。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r001 -run '^TestUserLoginUnknownPhoneUnauthorized$' -count=1
go test ./test/r001 -run '^TestUserRegisterFreshPhoneCreates$' -count=1
go test ./test/r001 -run '^TestUserMeMissingResourceNotFound$' -count=1
```

## 错误信息
```
TestUserLoginUnknownPhoneUnauthorized: login unknown phone status = 500, want 401
TestUserRegisterFreshPhoneCreates: register new phone status = 500, want 200
TestUserMeMissingResourceNotFound: me missing user status = 500, want 404
```
