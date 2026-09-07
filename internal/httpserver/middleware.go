package httpserver

import (
	"context"
	"net/http"
	"strings"

	"priceauth/internal/auth"
	"priceauth/internal/models"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// requireAuth validates the Authorization: Bearer <token> header and, on
// success, stores the token's claims (user id, email, role) in the
// request context.
func requireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			tokenString, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || tokenString == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
				return
			}

			claims, err := auth.ParseToken(jwtSecret, tokenString)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireRole builds on requireAuth's claims to reject callers whose token
// role doesn't match. Must be mounted behind requireAuth.
func requireRole(role models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := claimsFromContext(r.Context())
			if !ok || claims.Role != role {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func claimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}

func userIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}
