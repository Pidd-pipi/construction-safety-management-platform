package r005

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"safetyplatform/internal/config"
	"safetyplatform/internal/middleware"
	"safetyplatform/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestLoginTokenStaysUsable(t *testing.T) {
	token, err := util.GenerateToken("test-secret", util.DurationHours(72), 1, "13800000001", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}
	claims, err := util.ParseToken("test-secret", token)
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}
	if claims.UserID != 1 || claims.Phone != "13800000001" {
		t.Fatalf("claims mismatch: %+v", claims)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.After(time.Now()) {
		t.Fatalf("token should not be expired at issuance: %v", claims.ExpiresAt)
	}
}

func TestExpiredTokenParseRejected(t *testing.T) {
	claims := util.Claims{
		UserID: 1, Phone: "13800000001", Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "safety-platform",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if _, err := util.ParseToken("test-secret", token); err == nil {
		t.Fatalf("expired token should be rejected, got nil error")
	}
}

func authContextEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 72}
	r := gin.New()
	auth := r.Group("/api")
	auth.Use(middleware.AuthRequired(cfg))
	auth.GET("/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": middleware.GetUserID(c), "phone": middleware.GetPhone(c)})
	})
	return r
}

func TestInvalidTokenAccessRejected(t *testing.T) {
	engine := authContextEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token status = %d, want 401", w.Code)
	}
}

func TestExpiredTokenAccessRejected(t *testing.T) {
	engine := authContextEngine(t)
	claims := jwt.MapClaims{
		"user_id": 1, "phone": "13800000001", "role": "admin",
		"exp": time.Now().Add(-time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"iss": "safety-platform",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign expired: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired token status = %d, want 401", w.Code)
	}
}

func TestJwtConfigInjectsExpireHours(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpireHours: 72}
	got := -1
	r := gin.New()
	r.Use(middleware.JWTConfig(cfg))
	r.GET("/probe", func(c *gin.Context) {
		got = c.GetInt("jwt_expire_hours")
		c.JSON(http.StatusOK, gin.H{})
	})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got != 72 {
		t.Fatalf("jwt_expire_hours = %d, want 72", got)
	}
}

func TestAuthContextIncludesPhone(t *testing.T) {
	engine := authContextEngine(t)
	claims := jwt.MapClaims{
		"user_id": 1, "phone": "13800000001", "role": "admin",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"iss": "safety-platform",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign valid: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("valid token status = %d, want 200", w.Code)
	}
	if !containsStr(w.Body.String(), "13800000001") {
		t.Fatalf("phone not injected into context: %s", w.Body.String())
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

func TestConfigDefaultsJwtExpireHours(t *testing.T) {
	cfg := config.Load()
	if cfg.JWTExpireHours != 72 {
		t.Fatalf("JWTExpireHours default = %d, want 72", cfg.JWTExpireHours)
	}
}
