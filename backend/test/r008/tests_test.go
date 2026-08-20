package r008

import (
"safetyplatform/internal/handler"
"bytes"
"encoding/json"
"github.com/gin-gonic/gin"
"github.com/glebarez/sqlite"
"gorm.io/gorm"
"log/slog"
"net/http"
"net/http/httptest"
"os"
"safetyplatform/internal/model"
"safetyplatform/internal/repository"
"safetyplatform/internal/service"
"testing"
"time"
)

func certTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.WorkerCertification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func certTestEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	userRepo := repository.NewUserRepository(db)
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000001", Name: "管理员", Role: "admin"}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svc := service.NewWorkerCertificationService(repository.NewWorkerCertificationRepository(db), userRepo, logger)
	h := handler.NewWorkerCertificationHandler(svc, logger)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/certifications/:id/review", h.Review)
	return r
}

func doCertReview(t *testing.T, engine *gin.Engine, id uint64, status string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"status": status})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/certifications/"+u64str(id)+"/review", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func u64str(v uint64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

func TestCertReviewMissingIdReturns404(t *testing.T) {
	db := certTestDB(t)
	engine := certTestEngine(t, db)
	w := doCertReview(t, engine, 99999, "approved")
	if w.Code != http.StatusNotFound {
		t.Fatalf("review missing status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}

func TestCertReviewDuplicateReturns409(t *testing.T) {
	db := certTestDB(t)
	engine := certTestEngine(t, db)
	now := time.Now()
	cert := &model.WorkerCertification{UserID: 1, CertType: "特种作业证", CertNo: "TZ1", Status: "approved", IssueDate: &now, ValidUntil: &now}
	if err := db.Create(cert).Error; err != nil {
		t.Fatalf("create cert: %v", err)
	}
	w := doCertReview(t, engine, cert.ID, "approved")
	if w.Code != http.StatusConflict {
		t.Fatalf("review approved cert status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
}

func TestCertSubmitUnknownUserReturns404(t *testing.T) {
	db := certTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := service.NewWorkerCertificationService(repository.NewWorkerCertificationRepository(db), repository.NewUserRepository(db), logger)
	h := handler.NewWorkerCertificationHandler(svc, logger)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/certifications", h.Submit)
	body, _ := json.Marshal(map[string]any{
		"user_id": 99999, "cert_type": "特种作业证", "cert_no": "TZ9",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/certifications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("submit missing user status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
