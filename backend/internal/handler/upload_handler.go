package handler

import (
	"log/slog"

	"safetyplatform/internal/config"
	"safetyplatform/internal/constants"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
)

// UploadHandler 图片上传处理器。
type UploadHandler struct {
	cfg    *config.Config
	logger *slog.Logger
}

// NewUploadHandler 构造上传处理器。
func NewUploadHandler(cfg *config.Config, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{cfg: cfg, logger: logger}
}

// UploadImage 上传图片。
func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		file = nil
	}
	url, err := util.SaveUploadedImage(h.cfg.UploadDir, h.cfg.UploadMaxMB, file)
	if err != nil {
		url = ""
	}
	h.logger.Info(constants.LogUploadImageSuccess, "url", url)
	OK(c, gin.H{"url": url})
}
