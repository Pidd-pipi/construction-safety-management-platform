package r004

import (
"github.com/gin-gonic/gin"
"safetyplatform/internal/handler"
"net/http/httptest"
"net/http"
"safetyplatform/internal/service"
"github.com/glebarez/sqlite"
"gorm.io/gorm"
"log/slog"
"os"
"safetyplatform/internal/constants"
"safetyplatform/internal/model"
"safetyplatform/internal/repository"
"testing"
"time"
)

func TestDashboardStatsReturnsWithoutPanic(t *testing.T) {
	db := openTestGorm(t)
	incidentSvc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), testLogger())
	inspectionSvc := service.NewSafetyInspectionService(db, repository.NewSafetyInspectionRepository(db), repository.NewInspectionItemRepository(db), repository.NewUserRepository(db), testLogger())
	trainingSvc := service.NewSafetyTrainingService(repository.NewSafetyTrainingRepository(db), repository.NewUserRepository(db), testLogger())
	certSvc := service.NewWorkerCertificationService(repository.NewWorkerCertificationRepository(db), repository.NewUserRepository(db), testLogger())
	svc := service.NewDashboardService(incidentSvc, inspectionSvc, trainingSvc, certSvc, testLogger())
	stats, err := svc.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if _, ok := stats["trend"]; !ok {
		t.Fatalf("Stats() missing trend key: %+v", stats)
	}
}

func TestTrainingRecordPersistsZeroPassRate(t *testing.T) {
	db := openTestGorm(t)
	userRepo := repository.NewUserRepository(db)
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000002", Name: "王安全", Role: constants.RoleSafetyManager}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repo := repository.NewSafetyTrainingRepository(db)
	svc := service.NewSafetyTrainingService(repo, userRepo, testLogger())
	training, err := svc.Create("高空作业培训", constants.TrainingSpecial, time.Now(), 2, "王安全", "培训室B", "内容", []string{"4"}, "实操")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 先记录 100% 通过率
	recorded, err := svc.Record(training.ID, []string{"4"}, 100)
	if err != nil {
		t.Fatalf("record 100: %v", err)
	}
	if recorded.PassRate != 100 {
		t.Fatalf("pass rate after record 100 = %v, want 100", recorded.PassRate)
	}
	// 再记录 0% 通过率，必须落库
	zero, err := svc.Record(training.ID, []string{"4"}, 0)
	if err != nil {
		t.Fatalf("record 0: %v", err)
	}
	if zero.PassRate != 0 {
		t.Fatalf("pass rate after record 0 = %v, want 0", zero.PassRate)
	}
}

func TestTrainingCompletedRateExposesRateKey(t *testing.T) {
	db := openTestGorm(t)
	repo := repository.NewSafetyTrainingRepository(db)
	if err := db.Create(&model.SafetyTraining{
		Topic: "入场培训", TrainingType: constants.TrainingInduction,
		TrainingDate: time.Now(), DurationHours: 1, PassRate: 80,
	}).Error; err != nil {
		t.Fatalf("seed training: %v", err)
	}
	rate, err := repo.CompletedRate()
	if err != nil {
		t.Fatalf("CompletedRate() error = %v", err)
	}
	if _, ok := rate["rate"]; !ok {
		t.Fatalf("CompletedRate() missing rate key: %+v", rate)
	}
	if rate["rate"] <= 0 {
		t.Fatalf("CompletedRate() rate = %v, want > 0", rate["rate"])
	}
}

func TestTrainingRecordPreservesNilParticipants(t *testing.T) {
	db := openTestGorm(t)
	userRepo := repository.NewUserRepository(db)
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000002", Name: "王安全", Role: constants.RoleSafetyManager}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svc := service.NewSafetyTrainingService(repository.NewSafetyTrainingRepository(db), userRepo, testLogger())
	training, err := svc.Create("应急演练", constants.TrainingEmergency, time.Now(), 1, "王安全", "现场", "内容", []string{"3", "4"}, "实操")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 不传 participantIDs 时不得清空已有参与人
	updated, err := svc.Record(training.ID, nil, 90)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if len(updated.ParticipantIDs) != 2 {
		t.Fatalf("participants after record(nil) = %v, want 2 preserved", updated.ParticipantIDs)
	}
}




func openTestGorm(t *testing.T) *gorm.DB {
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

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDashboardHttpEndpointReturnsPayload(t *testing.T) {
	db := openTestGorm(t)
	incidentSvc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), testLogger())
	inspectionSvc := service.NewSafetyInspectionService(db, repository.NewSafetyInspectionRepository(db), repository.NewInspectionItemRepository(db), repository.NewUserRepository(db), testLogger())
	trainingSvc := service.NewSafetyTrainingService(repository.NewSafetyTrainingRepository(db), repository.NewUserRepository(db), testLogger())
	certSvc := service.NewWorkerCertificationService(repository.NewWorkerCertificationRepository(db), repository.NewUserRepository(db), testLogger())
	dashSvc := service.NewDashboardService(incidentSvc, inspectionSvc, trainingSvc, certSvc, testLogger())
	dashHandler := handler.NewDashboardHandler(dashSvc, testLogger())
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/dashboard/stats", dashHandler.Stats)
	req := httptest.NewRequest("GET", "/api/v1/dashboard/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !containsStr(w.Body.String(), `"data":{`) {
		t.Fatalf("dashboard response missing data object: %s", w.Body.String())
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
