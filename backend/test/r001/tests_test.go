package r001

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"safetyplatform/internal/config"
	"safetyplatform/internal/handler"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func engine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	userSvc := service.NewUserService(repository.NewUserRepository(db), logger)
	userHandler := handler.NewUserHandler(userSvc, logger)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 72}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", userHandler.Register)
	auth.POST("/login", userHandler.Login)
	users := r.Group("/api/v1/users")
	users.Use(middleware.AuthRequired(cfg))
	users.GET("/me", userHandler.Me)
	return r
}

func doReq(t *testing.T, engine *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestUserLoginUnknownPhoneUnauthorized(t *testing.T) {
	db := testDB(t)
	eng := engine(t, db)
	w := doReq(t, eng, "POST", "/api/v1/auth/login", map[string]string{"phone": "13900009999", "password": "x123456"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("login unknown phone status = %d, want 401", w.Code)
	}
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 40101 {
		t.Fatalf("login unknown phone code = %d, want 40101", resp.Code)
	}
}

func TestUserRegisterFreshPhoneCreates(t *testing.T) {
	db := testDB(t)
	eng := engine(t, db)
	w := doReq(t, eng, "POST", "/api/v1/auth/register", map[string]string{"phone": "13900001111", "password": "pass123", "name": "新用户"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("register new phone status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.Model(&model.User{}).Where("phone = ?", "13900001111").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1", count)
	}
}

func TestUserMeMissingResourceNotFound(t *testing.T) {
	db := testDB(t)
	eng := engine(t, db)
	token, err := util.GenerateToken("test-secret", util.DurationHours(72), 99999, "13900009999", "worker")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	w := doReq(t, eng, "GET", "/api/v1/users/me", nil, token)
	if w.Code != http.StatusNotFound {
		t.Fatalf("me missing user status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
}
