package httpserver

import (
	"context"
	"net/http"
	"strings"

	"priceuser/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "claims"

// requireAuth validates the Authorization: Bearer <token> header (a token
// issued by price-auth) and stores its claims in the request context.
func requireAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
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

// userIDFromContext returns the authenticated caller's id. Only valid behind
// requireAuth.
func userIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	if !ok {
		return "", false
	}
	return claims.UserID, true
}
