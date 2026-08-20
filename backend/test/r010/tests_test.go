package r010

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"safetyplatform/internal/constants"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/model"
	"safetyplatform/internal/repository"
	"safetyplatform/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SafetyIncident{}, &model.SafetyInspection{},
		&model.InspectionItem{}, &model.SafetyTraining{}, &model.WorkerCertification{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func slogL() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestSeedAdminKeepsAdminRole(t *testing.T) {
	db := openDB(t)
	svc := service.NewSeedService(db, slogL())
	if err := svc.Seed(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var admin model.User
	if err := db.Where("phone = ?", "13800000001").First(&admin).Error; err != nil {
		t.Fatalf("find admin: %v", err)
	}
	if admin.Role != constants.RoleAdmin {
		t.Fatalf("seeded admin role = %q, want %q", admin.Role, constants.RoleAdmin)
	}
}

func TestSeedAdminLoginWithAdminPass(t *testing.T) {
	db := openDB(t)
	seedSvc := service.NewSeedService(db, slogL())
	if err := seedSvc.Seed(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	userSvc := service.NewUserService(repository.NewUserRepository(db), slogL())
	token, user, err := userSvc.Login("test-secret", 72, "13800000001", "Admin@123")
	if err != nil {
		t.Fatalf("admin login: %v", err)
	}
	if token == "" || user.Phone != "13800000001" {
		t.Fatalf("admin login result mismatch: user=%+v", user)
	}
}

func TestSeedWorkerKeepsWorkerRole(t *testing.T) {
	db := openDB(t)
	svc := service.NewSeedService(db, slogL())
	if err := svc.Seed(); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var worker model.User
	if err := db.Where("phone = ?", "13800000004").First(&worker).Error; err != nil {
		t.Fatalf("find worker: %v", err)
	}
	if worker.Role != constants.RoleWorker {
		t.Fatalf("seeded worker role = %q, want %q", worker.Role, constants.RoleWorker)
	}
}

func TestUserModelDefaultsToWorkerRole(t *testing.T) {
	db := openDB(t)
	u := &model.User{Phone: "13900007777", PasswordHash: "x", Name: "新工人"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	var got model.User
	if err := db.First(&got, u.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Role != constants.RoleWorker {
		t.Fatalf("default role = %q, want %q", got.Role, constants.RoleWorker)
	}
}

func roleEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if role := c.GetHeader("X-Test-Role"); role != "" {
			c.Set("role", role)
		}
		c.Next()
	})
	admin := r.Group("/admin")
	admin.Use(middleware.RequireRole("admin"))
	admin.GET("/users", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	manager := r.Group("/manage")
	manager.Use(middleware.RequireRole("safety_manager"))
	manager.GET("/trainings", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func TestRoleGuardLetsAdminThrough(t *testing.T) {
	r := roleEngine()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.Header.Set("X-Test-Role", "admin")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin role status = %d, want 200", w.Code)
	}
}

func TestRoleGuardLetsManagerThrough(t *testing.T) {
	r := roleEngine()
	req := httptest.NewRequest(http.MethodGet, "/manage/trainings", nil)
	req.Header.Set("X-Test-Role", "safety_manager")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("manager role status = %d, want 200", w.Code)
	}
}
