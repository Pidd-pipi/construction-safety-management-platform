# BUG_REPRO

## Bug 是什么
登录态生命周期（deadline）被破坏：GenerateToken 签发即过期、ParseToken 跳过有效期校验、AuthRequired 对解析失败不拦截且不注入 phone、JWTConfig 硬编码 0 小时、config 默认 0 小时，导致 token 一签发就失效、所有受保护接口 401。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r005 -run '^TestLoginTokenStaysUsable$' -count=1
go test ./test/r005 -run '^TestExpiredTokenParseRejected$' -count=1
go test ./test/r005 -run '^TestInvalidTokenAccessRejected$' -count=1
go test ./test/r005 -run '^TestExpiredTokenAccessRejected$' -count=1
go test ./test/r005 -run '^TestJwtConfigInjectsExpireHours$' -count=1
go test ./test/r005 -run '^TestConfigDefaultsJwtExpireHours$' -count=1
go test ./test/r005 -run '^TestAuthContextIncludesPhone$' -count=1
```

## 错误信息
```
TestLoginTokenStaysUsable: token should not be expired at issuance
TestExpiredTokenParseRejected: expired token should be rejected, got nil error
TestInvalidTokenAccessRejected: invalid token status = 200, want 401
TestJwtConfigInjectsExpireHours: jwt_expire_hours = 0, want 72
TestConfigDefaultsJwtExpireHours: JWTExpireHours default = 0, want 72
TestAuthContextIncludesPhone: phone not injected: {"phone":"","user_id":1}
```
