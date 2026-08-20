package r002

import (
"safetyplatform/internal/service"
"context"
"github.com/glebarez/sqlite"
"gorm.io/gorm"
"log/slog"
"os"
"safetyplatform/internal/constants"
"safetyplatform/internal/model"
"safetyplatform/internal/repository"
"safetyplatform/internal/util"
"testing"
"time"
)

func incidentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SafetyIncident{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func incidentTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func seedIncident(t *testing.T, db *gorm.DB) *model.SafetyIncident {
	t.Helper()
	svc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), incidentTestLogger())
	inc, err := svc.Report(1, "脚手架扣件松动", "三层东侧", time.Now(), "SITE-A", "三层", constants.SeverityMajor, "坠落", []string{"4"}, nil)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if _, err := svc.Assign(inc.ID); err != nil {
		t.Fatalf("assign: %v", err)
	}
	return inc
}

func TestIncidentRectifyThenCloseWorks(t *testing.T) {
	db := incidentTestDB(t)
	inc := seedIncident(t, db)
	svc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), incidentTestLogger())
	dl := time.Now().AddDate(0, 0, 7)
	rect, err := svc.SubmitRectification(inc.ID, "已更换扣件", &dl)
	if err != nil {
		t.Fatalf("rectify: %v", err)
	}
	if rect.Status != constants.IncidentRectifying {
		t.Fatalf("rectify status = %s, want %s", rect.Status, constants.IncidentRectifying)
	}
	closed, err := svc.Close(inc.ID)
	if err != nil {
		t.Fatalf("close after rectify: %v", err)
	}
	if closed.Status != constants.IncidentClosed {
		t.Fatalf("close status = %s, want %s", closed.Status, constants.IncidentClosed)
	}
}

func TestIncidentPendingListKeepsRectifying(t *testing.T) {
	db := incidentTestDB(t)
	inc := seedIncident(t, db)
	svc := service.NewSafetyIncidentService(repository.NewSafetyIncidentRepository(db), repository.NewUserRepository(db), incidentTestLogger())
	dl := time.Now().AddDate(0, 0, 7)
	if _, err := svc.SubmitRectification(inc.ID, "已整改", &dl); err != nil {
		t.Fatalf("rectify: %v", err)
	}
	pending, err := svc.PendingRectification()
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	for _, p := range pending {
		if p.ID == inc.ID {
			return
		}
	}
	t.Fatalf("rectifying incident %d missing from pending list: %+v", inc.ID, pending)
}

func TestIncidentStatusEnumAdmitsRectifying(t *testing.T) {
	if !constants.IsValidIncidentStatus(constants.IncidentRectifying) {
		t.Fatalf("IsValidIncidentStatus(%q) = false, want true", constants.IncidentRectifying)
	}
}

func TestIncidentStatusTextMapsRectifying(t *testing.T) {
	if got := util.IncidentStatusText(constants.IncidentRectifying); got != "整改中" {
		t.Fatalf("IncidentStatusText(rectifying) = %q, want 整改中", got)
	}
}

// keep context import used for future diagnosis flows
var _ = context.Background
