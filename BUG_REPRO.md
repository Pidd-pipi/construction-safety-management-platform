# BUG_REPRO

## Bug 是什么
仪表盘聚合与培训记录存在 nil 路径缺陷：DashboardService.Stats 声明 nil map 直接写入触发 assignment to entry in nil map panic；SafetyTraining.Record 用 passRate>0 跳过 0% 通过率、nil participantIDs 覆盖已有参与人；CompletedRate 返回缺少 rate 键。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r004 -run '^TestDashboardHttpEndpointReturnsPayload$' -count=1
go test ./test/r004 -run '^TestTrainingRecordPersistsZeroPassRate$' -count=1
go test ./test/r004 -run '^TestTrainingCompletedRateExposesRateKey$' -count=1
go test ./test/r004 -run '^TestTrainingRecordPreservesNilParticipants$' -count=1
```

## 错误信息
```
panic: assignment to entry in nil map
safetyplatform/internal/service.(*DashboardService).Stats(...)
  internal/service/dashboard_service.go:51
TestTrainingRecordPersistsZeroPassRate: pass rate after record 0 = 100, want 0
TestTrainingCompletedRateExposesRateKey: CompletedRate() missing rate key
TestTrainingRecordPreservesNilParticipants: participants after record(nil) = []
```
