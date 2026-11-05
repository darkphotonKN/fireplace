package commonauth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret"

func mint(t *testing.T, tokenType TokenType, userID uuid.UUID, secret string, ttl time.Duration) string {
	t.Helper()
	tok, err := GenerateJWT(userID, tokenType, secret, ttl)
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return tok
}

// sign builds tokens GenerateJWT cannot produce: it always sets sub and tokenType.
func sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func TestValidateAccessToken_AccessToken_ReturnsUserID(t *testing.T) {
	userID := uuid.New()

	got, err := ValidateAccessToken(mint(t, TokenTypeAccess, userID, testSecret, time.Hour), testSecret)

	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if got != userID {
		t.Errorf("want user %s, got %s", userID, got)
	}
}

func TestValidateAccessToken_Rejected(t *testing.T) {
	userID := uuid.New()
	exp := time.Now().Add(time.Hour).Unix()
	cases := map[string]string{
		"refresh token":     mint(t, TokenTypeRefresh, userID, testSecret, time.Hour),
		"no tokenType":      sign(t, jwt.MapClaims{"sub": userID.String(), "exp": exp}),
		"unknown tokenType": sign(t, jwt.MapClaims{"sub": userID.String(), "exp": exp, "tokenType": "admin"}),
		"non-string type":   sign(t, jwt.MapClaims{"sub": userID.String(), "exp": exp, "tokenType": 1}),
		"expired":           mint(t, TokenTypeAccess, userID, testSecret, -time.Minute),
		"wrong secret":      mint(t, TokenTypeAccess, userID, "other-secret", time.Hour),
		"garbage":           "not-a-jwt",
		"empty":             "",
		"no sub":            sign(t, jwt.MapClaims{"exp": exp, "tokenType": string(TokenTypeAccess)}),
		"unparseable sub":   sign(t, jwt.MapClaims{"sub": "nope", "exp": exp, "tokenType": string(TokenTypeAccess)}),
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ValidateAccessToken(token, testSecret)
			if err == nil {
				t.Fatalf("want an error, got user %s", got)
			}
			if got != uuid.Nil {
				t.Errorf("want the nil user on rejection, got %s", got)
			}
		})
	}
}

func TestValidateRefreshToken_AccessToken_Rejected(t *testing.T) {
	if _, err := ValidateRefreshToken(mint(t, TokenTypeAccess, uuid.New(), testSecret, time.Hour), testSecret); err == nil {
		t.Fatal("want an error for an access token")
	}
}
