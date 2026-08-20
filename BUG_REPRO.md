# BUG_REPRO

## Bug 是什么
RateLimiter 并发锁范围漂移：Limit 提前解锁使令牌读写逃出锁保护产生 data race；Stats 无锁读 map 竞态；router 登录路由未挂限流中间件导致公开入口不限流。

## 如何触发
在 backend 目录运行：

```bash
go test -race ./test/r007 -run '^TestLimiterConcurrentBurstRaceFree$' -count=1
go test -race ./test/r007 -run '^TestLoginEndpointEnforcesLimit$' -count=1
```

## 错误信息
```
WARNING: DATA RACE
Read at 0x00c0002e8c28 by goroutine 10:
  (*RateLimiter).Limit.func3() internal/middleware/rate_limiter.go:46
TestLoginEndpointEnforcesLimit: login endpoint never rate limited
```
