package r009

import (
	"safetyplatform/internal/handler"
"safetyplatform/internal/middleware"
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
)

func TestJsonListNilValueSerializesEmpty(t *testing.T) {
	var j model.JSONList
	v, err := j.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	ok := false
	switch x := v.(type) {
	case []byte:
		ok = string(x) == "[]"
	case string:
		ok = x == "[]"
	}
	if !ok {
		t.Fatalf("Value() of nil model.JSONList = %#v, want empty array", v)
	}
}




func auditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAuditLogWritesWithoutCrash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	db := auditTestDB(t)
	r := gin.New()
	r.Use(middleware.AuditLog(db, logger))
	r.POST("/api/v1/incidents", func(c *gin.Context) {
		c.Set("audit_detail", "some detail")
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", nil)
	w := httptest.NewRecorder()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("POST handler panicked: %v", r)
		}
	}()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestHttpRequestIdHeaderAlwaysPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Request-Id"); got == "" {
		t.Fatalf("X-Request-Id header empty")
	}
}




func TestStatsEndpointEmitsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := newDashboardService(t, dashboardTestDB(t), logger)
	h := handler.NewDashboardHandler(svc, logger)
	r := gin.New()
	r.GET("/api/v1/dashboard/stats", h.Stats)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got == "" || !containsStr(got, `"data":{`) {
		t.Fatalf("dashboard response missing data object: %s", got)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}




func dashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SafetyIncident{}, &model.SafetyInspection{},
		&model.SafetyTraining{}, &model.WorkerCertification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newDashboardService(t *testing.T, db *gorm.DB, logger *slog.Logger) *service.DashboardService {
	incidentSvc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), logger)
	inspectionSvc := service.NewSafetyInspectionService(db, repository.NewSafetyInspectionRepository(db), repository.NewInspectionItemRepository(db), repository.NewUserRepository(db), logger)
	trainingSvc := service.NewSafetyTrainingService(repository.NewSafetyTrainingRepository(db), repository.NewUserRepository(db), logger)
	certSvc := service.NewWorkerCertificationService(repository.NewWorkerCertificationRepository(db), repository.NewUserRepository(db), logger)
	return service.NewDashboardService(incidentSvc, inspectionSvc, trainingSvc, certSvc, logger)
}
