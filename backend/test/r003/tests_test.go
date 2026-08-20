package r003

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
"safetyplatform/internal/constants"
"safetyplatform/internal/model"
"safetyplatform/internal/repository"
"safetyplatform/internal/service"
"testing"
"time"
)

func inspectionHTTPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SafetyInspection{}, &model.InspectionItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func inspectionHTTPTestEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	userRepo := repository.NewUserRepository(db)
	inspRepo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000003", Name: "李监理", Role: constants.RoleInspector}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	svc := service.NewSafetyInspectionService(db, inspRepo, itemRepo, userRepo, logger)
	h := handler.NewSafetyInspectionHandler(svc, logger)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/inspections", h.Create)
	r.POST("/api/v1/inspections/:id/execute", h.Execute)
	return r
}

func postJSON(t *testing.T, engine *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestInspectionHttpCreateKeepsItemCount(t *testing.T) {
	db := inspectionHTTPTestDB(t)
	engine := inspectionHTTPTestEngine(t, db)
	w := postJSON(t, engine, "/api/v1/inspections", map[string]any{
		"name": "8月例行检查", "inspection_type": "routine", "area": "全工地",
		"inspection_date": time.Now().Format(time.RFC3339), "inspector_id": 1,
		"items": []map[string]any{
			{"item_name": "安全帽佩戴", "passed": true},
			{"item_name": "临边防护栏杆", "passed": false},
			{"item_name": "消防器材齐全", "passed": true},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	var stored int64
	if err := db.Model(&model.InspectionItem{}).Where("inspection_id = ?", created.Data.ID).Count(&stored).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if stored != 3 {
		t.Fatalf("stored items = %d, want 3", stored)
	}
}

func TestInspectionHttpCreateBindsAllItems(t *testing.T) {
	db := inspectionHTTPTestDB(t)
	engine := inspectionHTTPTestEngine(t, db)
	w := postJSON(t, engine, "/api/v1/inspections", map[string]any{
		"name": "高处作业专项检查", "inspection_type": "special", "area": "三层",
		"inspection_date": time.Now().Format(time.RFC3339), "inspector_id": 1,
		"items": []map[string]any{
			{"item_name": "安全帽佩戴", "passed": true},
			{"item_name": "临边防护栏杆", "passed": false},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	var got []model.InspectionItem
	if err := db.Where("inspection_id = ?", created.Data.ID).Find(&got).Error; err != nil {
		t.Fatalf("find items: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("bound items = %d, want 2", len(got))
	}
	for _, it := range got {
		if it.InspectionID != created.Data.ID {
			t.Fatalf("item %d bound to %d, want %d", it.ID, it.InspectionID, created.Data.ID)
		}
	}
}




func TestInspectionCreateItemCount(t *testing.T) {
	db := openTestGorm(t)
	inspRepo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewSafetyInspectionService(db, inspRepo, itemRepo, userRepo, testLogger())
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000003", Name: "李监理", Role: constants.RoleInspector}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	items := []model.InspectionItem{
		{ItemName: "安全帽佩戴", Passed: true},
		{ItemName: "临边防护栏杆", Passed: false},
		{ItemName: "消防器材齐全", Passed: true},
	}
	ins, err := svc.Create("8月例行检查", constants.InspectionRoutine, "全工地", time.Now(), 1, items)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var stored int64
	if err := db.Model(&model.InspectionItem{}).Where("inspection_id = ?", ins.ID).Count(&stored).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if stored != 3 {
		t.Fatalf("stored items = %d, want 3", stored)
	}
}

func TestInspectionCreateItemsBound(t *testing.T) {
	db := openTestGorm(t)
	inspRepo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewSafetyInspectionService(db, inspRepo, itemRepo, userRepo, testLogger())
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000003", Name: "李监理", Role: constants.RoleInspector}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	items := []model.InspectionItem{{ItemName: "安全帽佩戴", Passed: true}, {ItemName: "临边防护栏杆", Passed: false}}
	ins, err := svc.Create("高处作业专项检查", constants.InspectionSpecial, "三层", time.Now(), 1, items)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var got []model.InspectionItem
	if err := db.Where("inspection_id = ?", ins.ID).Find(&got).Error; err != nil {
		t.Fatalf("find items: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("bound items = %d, want 2", len(got))
	}
	for _, it := range got {
		if it.InspectionID != ins.ID {
			t.Fatalf("item %d bound to %d, want %d", it.ID, it.InspectionID, ins.ID)
		}
	}
}

func TestInspectionExecuteRecountsStoredItems(t *testing.T) {
	db := openTestGorm(t)
	inspRepo := repository.NewSafetyInspectionRepository(db)
	itemRepo := repository.NewInspectionItemRepository(db)
	userRepo := repository.NewUserRepository(db)
	svc := service.NewSafetyInspectionService(db, inspRepo, itemRepo, userRepo, testLogger())
	if err := userRepo.Create(&model.User{ID: 1, Phone: "13800000003", Name: "李监理", Role: constants.RoleInspector}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	ins, err := svc.Create("检查", constants.InspectionRoutine, "A区", time.Now(), 1, []model.InspectionItem{
		{ItemName: "项1", Passed: true}, {ItemName: "项2", Passed: false},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 执行时传入与库中不匹配的 items，应回退按已存检查项统计
	out, err := svc.Execute(ins.ID, []model.InspectionItem{{ItemName: "新增项", Passed: true}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.PassedCount != 1 || out.IssueCount != 1 {
		t.Fatalf("execute counts = %d/%d, want 1/1 (recount existing)", out.PassedCount, out.IssueCount)
	}
	if out.Status != constants.InspectionFailed {
		t.Fatalf("execute status = %s, want %s", out.Status, constants.InspectionFailed)
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
