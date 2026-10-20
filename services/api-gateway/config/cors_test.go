package config

import (
	"slices"
	"testing"
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
