# BUG_REPRO

## Bug 是什么
上传流程资源与错误生命周期错位：SaveUploadedImage 超限/建目录错误被吞掉、文件名固定导致旧文件被覆盖（漏释放）、handler 吞掉保存错误返回成功、缺文件继续执行，造成文件丢失与假成功。

## 如何触发
在 backend 目录运行：

```bash
go test ./test/r006 -run '^TestFileUploadOversizeRejected$' -count=1
go test ./test/r006 -run '^TestFileUploadKeepsTwoFilesDistinct$' -count=1
go test ./test/r006 -run '^TestFileUploadDirErrorSurfaces$' -count=1
go test ./test/r006 -run '^TestFileUploadBadTypeReturns400$' -count=1
go test ./test/r006 -run '^TestFileUploadMissingFileReturns400$' -count=1
```

## 错误信息
```
TestFileUploadOversizeRejected: oversize upload should return error (got nil)
TestFileUploadKeepsTwoFilesDistinct: two uploads share path "/uploads/image.png"
TestFileUploadDirErrorSurfaces: uncreatable upload dir should return error (got nil)
TestFileUploadBadTypeReturns400: upload bad type status = 200, want 400
TestFileUploadMissingFileReturns400: missing file status = 200, want 400
```
