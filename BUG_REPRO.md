# BUG_REPRO

## Bug 是什么
事件状态机新增 rectifying 中间态后未同步：IncidentStatusValues 未收录、Close 转换表不接受 rectifying、PendingRectification 查询漏计该状态、IncidentStatusText 未映射，导致提交整改后事件无法关闭且从待整改列表消失。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r002 -run '^TestIncidentRectifyThenCloseWorks$' -count=1
go test ./test/r002 -run '^TestIncidentPendingListKeepsRectifying$' -count=1
go test ./test/r002 -run '^TestIncidentStatusEnumAdmitsRectifying$' -count=1
go test ./test/r002 -run '^TestIncidentStatusTextMapsRectifying$' -count=1
```

## 错误信息
```
TestIncidentRectifyThenCloseWorks: close after rectify: code=40901 conflict: status=rectifying
TestIncidentPendingListKeepsRectifying: rectifying incident missing from pending list
TestIncidentStatusEnumAdmitsRectifying: IsValidIncidentStatus(rectifying) = false
TestIncidentStatusTextMapsRectifying: IncidentStatusText(rectifying) = "rectifying"
```
