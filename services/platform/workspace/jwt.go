package workspace

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func parseWorkspaceJWT(tokStr string, secret []byte) (jwt.MapClaims, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("jwt secret required")
	}
	tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected alg")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid jwt")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("bad claims")
	}
	tenantID, _ := claims["tenant_id"].(string)
	sub, _ := claims["sub"].(string)
	if tenantID == "" || sub == "" {
		return nil, fmt.Errorf("missing tenant_id/sub")
	}
	return claims, nil
}
