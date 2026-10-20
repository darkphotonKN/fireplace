package config

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func TestCorsAllowedOrigins(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want []string
	}{
		{"unset falls back to the dev servers", "", devOrigins},
		{"blank entries only falls back to the dev servers", " , ,", devOrigins},
		{"single origin", "https://fireplace.vercel.app", []string{"https://fireplace.vercel.app"}},
		{
			"comma-separated, trimmed, trailing slash dropped",
			" https://fireplace.vercel.app/ , http://localhost:3010,",
			[]string{"https://fireplace.vercel.app", "http://localhost:3010"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CORS_ALLOWED_ORIGINS", tc.env)
			if got := corsAllowedOrigins(); !slices.Equal(got, tc.want) {
				t.Fatalf("corsAllowedOrigins() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCorsConfigAllowsVercelDeploymentWildcard(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://fireplace-*-kranti-nebhwanis-projects.vercel.app")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cors.New(corsConfig()))
	r.POST("/api/users/signin", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		origin  string
		allowed bool
	}{
		{"https://fireplace-jxhs9hmub-kranti-nebhwanis-projects.vercel.app", true},
		{"https://fireplace-git-main-kranti-nebhwanis-projects.vercel.app", true},
		{"https://fireplace-abc-someone-else.vercel.app", false},
		{"https://evil.example.com", false},
		{"http://localhost:3010", false},
	}
	for _, tc := range cases {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, "/api/users/signin", nil)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			got := w.Header().Get("Access-Control-Allow-Origin") == tc.origin
			if got != tc.allowed {
				t.Fatalf("origin %s: allowed=%v, want %v (status %d)", tc.origin, got, tc.allowed, w.Code)
			}
		})
	}
}
