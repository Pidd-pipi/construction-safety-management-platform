# BUG_REPRO

## Bug 是什么
用户角色状态机错位：种子数据管理员/工人角色写反、管理员密码哈希用普通用户密码、模型默认角色改为 admin、RBAC 中间件读错上下文键导致权限全拒绝。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r010 -run '^TestSeedAdminKeepsAdminRole$' -count=1
go test ./test/r010 -run '^TestSeedAdminLoginWithAdminPass$' -count=1
go test ./test/r010 -run '^TestSeedWorkerKeepsWorkerRole$' -count=1
go test ./test/r010 -run '^TestUserModelDefaultsToWorkerRole$' -count=1
go test ./test/r010 -run '^TestRoleGuardLetsAdminThrough$' -count=1
go test ./test/r010 -run '^TestRoleGuardLetsManagerThrough$' -count=1
```

## 错误信息
```
TestSeedAdminKeepsAdminRole: seeded admin role = "worker", want "admin"
TestSeedAdminLoginWithAdminPass: admin login: code=40101 message=手机号或密码错误
TestSeedWorkerKeepsWorkerRole: seeded worker role = "admin", want "worker"
TestUserModelDefaultsToWorkerRole: default role = "admin", want "worker"
TestRoleGuardLetsAdminThrough: admin role status = 403, want 200
TestRoleGuardLetsManagerThrough: manager role status = 403, want 200
```
