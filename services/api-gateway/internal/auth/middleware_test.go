package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-secret")
}

func TestAuthMiddleware_MissingHeader_Returns401(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_SetsUserId(t *testing.T) {
	userID := uuid.New()

	token, err := commonauth.GenerateJWT(userID, commonauth.TokenTypeAccess, "test-secret", 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var gotUserID uuid.UUID
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		gotUserID = c.MustGet("userId").(uuid.UUID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gotUserID != userID {
		t.Errorf("expected userId %s, got %s", userID, gotUserID)
	}
}
