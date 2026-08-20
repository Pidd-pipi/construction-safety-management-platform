# BUG_REPRO

## Bug 是什么
nil 路径级联：审计中间件声明 nil map 直接写入导致 POST panic；JSONList.Value 对 nil 返回空驱动值导致 JSON 列空值；newRequestID 返回空字符串使请求 ID 缺失；dashboard handler 返回 nil 数据。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r009 -run '^TestJsonListNilValueSerializesEmpty$' -count=1
go test ./test/r009 -run '^TestAuditLogWritesWithoutCrash$' -count=1
go test ./test/r009 -run '^TestHttpRequestIdHeaderAlwaysPresent$' -count=1
go test ./test/r009 -run '^TestStatsEndpointEmitsPayload$' -count=1
```

## 错误信息
```
TestJsonListNilValueSerializesEmpty: Value() of nil JSONList = <nil>
TestAuditLogWritesWithoutCrash: POST handler panicked: assignment to entry in nil map
TestHttpRequestIdHeaderAlwaysPresent: X-Request-Id header empty
TestStatsEndpointEmitsPayload: dashboard response missing data object
```
