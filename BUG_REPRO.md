# BUG_REPRO

## Bug 是什么
检查项切片处理错误：handler 用 make(len) 预分配导致前段零值检查项、service Create 用 range 拷贝未写回 InspectionID、Execute 回退统计误用请求切片、仓库 ListByInspection 截断切片，导致创建多出空白项、检查项未绑定检查单。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r003 -run '^TestInspectionHttpCreateKeepsItemCount$' -count=1
go test ./test/r003 -run '^TestInspectionHttpCreateBindsAllItems$' -count=1
go test ./test/r003 -run '^TestInspectionExecuteRecountsStoredItems$' -count=1
```

## 错误信息
```
TestInspectionHttpCreateKeepsItemCount: stored items = 0, want 3
TestInspectionHttpCreateBindsAllItems: bound items = 0, want 2
TestInspectionExecuteRecountsStoredItems: execute counts = 1/0, want 1/1
```
