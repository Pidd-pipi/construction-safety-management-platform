package util

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"safetyplatform/internal/constants"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// SaveUploadedImage 保存上传图片，返回可访问相对路径。
// 文件类型、大小、目录与落盘任一环节失败都会返回带业务码的 *AppError。
func SaveUploadedImage(uploadDir string, maxMB int64, file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", NewAppError(constants.CodeBadRequest, constants.MsgUnsupportedFileType)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExts[ext] {
		return "", NewAppError(constants.CodeUnsupportedType, constants.MsgUnsupportedFileType)
	}
	if file.Size > maxMB*1024*1024 {
		return "", NewAppError(constants.CodeUploadTooLarge, constants.MsgUploadTooLarge)
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", NewAppError(constants.CodeInternalError, constants.MsgInternalError)
	}
	// 用随机名避免同一目录下多次上传相互覆盖。
	name := randomFileName(ext)
	dst := filepath.Join(uploadDir, name)
	src, err := file.Open()
	if err != nil {
		return "", NewAppError(constants.CodeInternalError, constants.MsgInternalError)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return "", NewAppError(constants.CodeInternalError, constants.MsgInternalError)
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", NewAppError(constants.CodeInternalError, constants.MsgInternalError)
	}
	return "/uploads/" + name, nil
}

// randomFileName 生成带扩展名的随机文件名，16 字节随机十六进制与 request_id 同源。
func randomFileName(ext string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// 退化为可预测但唯一性极弱的兜底，仍保留扩展名。
		return "image" + ext
	}
	return hex.EncodeToString(b[:]) + ext
}
