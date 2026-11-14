package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	commonauth "github.com/darkphotonKN/fireplace/common/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

// A valid signature is not enough: only an access token is a bearer (FS-ZESEF
// R7, R8). Both renderers go through authenticate, so both are exercised, and
// the rejection reads exactly like any other invalid token.
func TestAuthenticate_TokenType_BothRenderers(t *testing.T) {
	userID := uuid.New()
	refresh, err := commonauth.GenerateJWT(userID, commonauth.TokenTypeRefresh, "test-secret", time.Hour)
	if err != nil {
		t.Fatalf("mint refresh token: %v", err)
	}
	access, err := commonauth.GenerateJWT(userID, commonauth.TokenTypeAccess, "test-secret", time.Hour)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	untyped, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign untyped token: %v", err)
	}

	renderers := map[string]struct {
		middleware gin.HandlerFunc
		body       string
		mediaType  string
	}{
		"legacy": {
			AuthMiddleware(),
			`{"message":"unauthorized","statusCode":401}`,
			"application/json",
		},
		"problem": {
			ProblemMiddleware(),
			`{"code":"UNAUTHENTICATED","detail":"unauthorized","errors":[],"status":401,"title":"Unauthorized","type":"about:blank"}`,
			"application/problem+json",
		},
	}
	tokens := map[string]struct {
		token    string
		wantCode int
	}{
		"access token passes":    {access, http.StatusOK},
		"refresh token rejected": {refresh, http.StatusUnauthorized},
		"no tokenType rejected":  {untyped, http.StatusUnauthorized},
	}

	for rendererName, renderer := range renderers {
		for tokenName, tc := range tokens {
			t.Run(rendererName+"/"+tokenName, func(t *testing.T) {
				router := gin.New()
				router.Use(renderer.middleware)
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"ok": true})
				})

				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer "+tc.token)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != tc.wantCode {
					t.Fatalf("want %d, got %d: %s", tc.wantCode, w.Code, w.Body.String())
				}
				if tc.wantCode == http.StatusOK {
					return
				}
				if got := w.Body.String(); got != renderer.body {
					t.Errorf("rejection body:\n got %s\nwant %s", got, renderer.body)
				}
				if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, renderer.mediaType) {
					t.Errorf("content-type: want %s, got %q", renderer.mediaType, ct)
				}
			})
		}
	}
}
