package httpserver

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Roles issued by price-auth (the Authorization Server).
const (
	roleAdmin   = "admin"
	roleCompany = "company"
	roleUser    = "user"
)

// Claims mirrors the payload price-auth signs: who the caller is, what they
// may do (role) and when the token expires. No password or personal data.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type claimsKey struct{}

// authGuard turns price-back into a Resource Server: it reads
// "Authorization: Bearer <token>", checks the HS256 signature with the secret
// shared with price-auth and the expiry, and only then lets the request
// through. With an empty secret (tests that exercise the handlers alone) the
// guard is off and every middleware is a no-op.
type authGuard struct {
	secret []byte
}

func (g authGuard) enabled() bool { return len(g.secret) > 0 }

// parse validates the token. Any change to header or payload (e.g. editing
// "role" on jwt.io) changes the HMAC, so the signature check fails.
func (g authGuard) parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return g.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// require returns a middleware that answers 401 when the token is missing or
// invalid and 403 when it is valid but its role is not in roles. An empty
// roles list accepts any authenticated caller.
func (g authGuard) require(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !g.enabled() {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || strings.TrimSpace(tokenString) == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
				return
			}
			claims, err := g.parse(strings.TrimSpace(tokenString))
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}
			if len(roles) > 0 && !slices.Contains(roles, claims.Role) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient role"})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}
}

// admin: internal staff (Painel Administrativo) and the collection bot.
func (g authGuard) admin() func(http.Handler) http.Handler { return g.require(roleAdmin) }

// staff: admin or a partner company (Página da Empresa).
func (g authGuard) staff() func(http.Handler) http.Handler {
	return g.require(roleAdmin, roleCompany)
}
