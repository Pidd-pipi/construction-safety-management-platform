package r007

import (
"safetyplatform/internal/router"
"safetyplatform/internal/middleware"
"bytes"
"encoding/json"
"github.com/gin-gonic/gin"
"github.com/glebarez/sqlite"
"gorm.io/gorm"
"log/slog"
"net/http"
"net/http/httptest"
"os"
"safetyplatform/internal/config"
"safetyplatform/internal/handler"
"safetyplatform/internal/model"
"safetyplatform/internal/repository"
"safetyplatform/internal/service"
"sync"
"testing"
)

func TestLimiterConcurrentBurstRaceFree(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := middleware.NewRateLimiter(120)
	r := gin.New()
	r.Use(rl.Limit())
	r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	const workers = 8
	const perWorker = 30
	start := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < perWorker; i++ {
				req := httptest.NewRequest(http.MethodGet, "/ping", nil)
				req.RemoteAddr = "203.0.113.10:1234"
				rec := httptest.NewRecorder()
				r.ServeHTTP(rec, req)
			}
		}()
	}
	close(start)
	wg.Wait()
	// 并发结束后令牌状态一致，无数据竞争；-race 会在存在竞态时直接失败
	_ = rl.Stats()
}




func routerTestDB(t *testing.T) *gorm.DB {
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

func routerTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	db := routerTestDB(t)
	seedUserForRateLimit(t, db)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 72, RateLimitPerMinute: 20, UploadDir: t.TempDir(), UploadMaxMB: 10}

	userRepo := repository.NewUserRepository(db)
	incidentRepo := repository.NewSafetyIncidentRepository(db)
	inspectionRepo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	trainingRepo := repository.NewSafetyTrainingRepository(db)
	certRepo := repository.NewWorkerCertificationRepository(db)

	userSvc := service.NewUserService(userRepo, logger)
	incidentSvc := service.NewSafetyIncidentService(incidentRepo, userRepo, logger)
	inspectionSvc := service.NewSafetyInspectionService(db, inspectionRepo, itemRepo, userRepo, logger)
	trainingSvc := service.NewSafetyTrainingService(trainingRepo, userRepo, logger)
	certSvc := service.NewWorkerCertificationService(certRepo, userRepo, logger)
	dashboardSvc := service.NewDashboardService(incidentSvc, inspectionSvc, trainingSvc, certSvc, logger)

	userHandler := handler.NewUserHandler(userSvc, logger)
	incidentHandler := handler.NewSafetyIncidentHandler(incidentSvc, logger)
	inspectionHandler := handler.NewSafetyInspectionHandler(inspectionSvc, logger)
	itemHandler := handler.NewInspectionItemHandler(inspectionSvc, logger)
	trainingHandler := handler.NewSafetyTrainingHandler(trainingSvc, logger)
	certHandler := handler.NewWorkerCertificationHandler(certSvc, logger)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc, logger)
	uploadHandler := handler.NewUploadHandler(cfg, logger)
	auditLogHandler := handler.NewAuditLogHandler(db, logger)

	r := router.New(cfg, db, logger, userHandler, incidentHandler, inspectionHandler, itemHandler,
		trainingHandler, certHandler, dashboardHandler, uploadHandler, auditLogHandler)
	return r.Setup()
}

func seedUserForRateLimit(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := service.NewUserService(repository.NewUserRepository(db), slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))
	if _, err := svc.Register("13900001111", "pass123", "测试用户", "worker"); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func TestLoginEndpointEnforcesLimit(t *testing.T) {
	engine := routerTestEngine(t)
	const workers = 8
	const perWorker = 20 // 共 160 个请求，perMin=20，必然触发限流
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	tooMany := 0
	ok := 0
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < perWorker; i++ {
				body, _ := json.Marshal(map[string]string{"phone": "13900001111", "password": "pass123"})
				req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
				req.RemoteAddr = "198.51.100.7:1234"
				rec := httptest.NewRecorder()
				engine.ServeHTTP(rec, req)
				mu.Lock()
				if rec.Code == http.StatusTooManyRequests {
					tooMany++
				} else if rec.Code == http.StatusOK {
					ok++
				}
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if tooMany == 0 {
		t.Fatalf("login endpoint never rate limited: ok=%d tooMany=%d, want some 429", ok, tooMany)
	}
	if ok == 0 {
		t.Fatalf("no login request succeeded: ok=%d", ok)
	}
}
