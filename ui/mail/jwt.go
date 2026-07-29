package mail

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims holds validated identity JWT fields for the webmail BFF.
type TokenClaims struct {
	TenantID string
	Email    string
	Subject  string
}

// ValidateToken parses and verifies an HS256 access token from identity-api.
func ValidateToken(tokenStr string, secret []byte) (*TokenClaims, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("jwt: secret required")
	}
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("jwt: invalid claims")
	}
	if exp, err := claims.GetExpirationTime(); err == nil && exp != nil && time.Now().After(exp.Time) {
		return nil, fmt.Errorf("jwt: expired")
	}
	email, _ := claims["email"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	sub, _ := claims["sub"].(string)
	if email == "" || tenantID == "" {
		return nil, fmt.Errorf("jwt: missing tenant_id or email")
	}
	return &TokenClaims{TenantID: tenantID, Email: email, Subject: sub}, nil
}

// SignTestToken issues a token for unit/e2e tests.
func SignTestToken(secret []byte, tenantID, email, sub string, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":       sub,
		"email":     email,
		"tenant_id": tenantID,
		"exp":       exp.Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}
