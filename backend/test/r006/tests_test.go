package r006

import (
"safetyplatform/internal/handler"
"safetyplatform/internal/util"
"bytes"
"encoding/json"
"github.com/gin-gonic/gin"
"log/slog"
"mime/multipart"
"net/http"
"net/http/httptest"
"os"
"path/filepath"
"safetyplatform/internal/config"
"testing"
)

// multipartFileHeader 通过真实 HTTP multipart 请求解析出可 Open 的 FileHeader。
func multipartFileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write form: %v", err)
	}
	w.Close()
	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	file, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	defer file.Close()
	return header
}

func TestFileUploadOversizeRejected(t *testing.T) {
	dir := t.TempDir()
	h := multipartFileHeader(t, "big.png", make([]byte, 100))
	h.Size = 11 * 1024 * 1024
	_, err := util.SaveUploadedImage(filepath.Join(dir, "up"), 10, h)
	if err == nil {
		t.Fatalf("oversize upload should return error")
	}
}

func TestFileUploadKeepsTwoFilesDistinct(t *testing.T) {
	dir := t.TempDir()
	up := filepath.Join(dir, "up")
	h1 := multipartFileHeader(t, "a.png", []byte("first"))
	p1, err := util.SaveUploadedImage(up, 10, h1)
	if err != nil {
		t.Fatalf("upload 1: %v", err)
	}
	h2 := multipartFileHeader(t, "b.png", []byte("second"))
	p2, err := util.SaveUploadedImage(up, 10, h2)
	if err != nil {
		t.Fatalf("upload 2: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("two uploads share path %q, want distinct files", p1)
	}
	for _, p := range []string{p1, p2} {
		if _, err := os.Stat(filepath.Join(up, filepath.Base(p))); err != nil {
			t.Fatalf("uploaded file missing: %v", err)
		}
	}
}

func TestFileUploadDirErrorSurfaces(t *testing.T) {
	// uploadDir 指向一个不可创建的路径（父路径是普通文件）
	dir := t.TempDir()
	blocker := filepath.Join(dir, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	h := multipartFileHeader(t, "ok.png", []byte("data"))
	_, err := util.SaveUploadedImage(filepath.Join(blocker, "sub"), 10, h)
	if err == nil {
		t.Fatalf("uncreatable upload dir should return error")
	}
}




func TestFileUploadBadTypeReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{UploadDir: t.TempDir(), UploadMaxMB: 10}
	h := handler.NewUploadHandler(cfg, logger)
	r := gin.New()
	r.POST("/api/v1/upload/image", h.UploadImage)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "bad.exe")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	fw.Write([]byte("not an image"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/image", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("upload bad type status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data *struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := jsonUnmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data != nil && resp.Data.URL != "" {
		t.Fatalf("failed upload returned url %q, want empty", resp.Data.URL)
	}
}

func TestFileUploadMissingFileReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{UploadDir: t.TempDir(), UploadMaxMB: 10}
	h := handler.NewUploadHandler(cfg, logger)
	r := gin.New()
	r.POST("/api/v1/upload/image", h.UploadImage)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/image", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxxx")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing file status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}




func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
