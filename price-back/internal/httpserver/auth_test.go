package httpserver_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/testutil"
)

const testSecret = "test-secret"

func signToken(t *testing.T, secret, role string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  "user-1",
		"username": "alice",
		"role":     role,
		"sub":      "user-1",
		"iat":      now.Unix(),
		"exp":      now.Add(ttl).Unix(),
	})
	s, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}

// tamperRole does what the jwt.io test does: decode the payload, change the
// role claim and re-encode it, keeping the original signature.
func tamperRole(t *testing.T, token, role string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	payload["role"] = role
	changed, err := json.Marshal(payload)
	require.NoError(t, err)
	parts[1] = base64.RawURLEncoding.EncodeToString(changed)
	return strings.Join(parts, ".")
}

func authRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAuth_ProtectedRouteWithoutTokenIs401(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	rec := authRequest(router, http.MethodPost, "/api/products", "", `{}`)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_TamperedRoleIsRejected(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	forged := tamperRole(t, signToken(t, testSecret, "user", time.Hour), "admin")
	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", forged, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_TokenSignedWithAnotherSecretIs401(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", signToken(t, "other-secret", "admin", time.Hour), "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ExpiredTokenIs401(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", signToken(t, testSecret, "admin", -time.Minute), "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_AlgNoneIs401(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"user_id": "user-1", "role": "admin", "exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", s, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_WrongRoleIs403(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", signToken(t, testSecret, "user", time.Hour), "")
	require.Equal(t, http.StatusForbidden, rec.Code)

	rec = authRequest(router, http.MethodGet, "/api/catalog/suggestions", signToken(t, testSecret, "company", time.Hour), "")
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuth_ValidAdminTokenReachesHandler(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	rec := authRequest(router, http.MethodGet, "/api/catalog/suggestions", signToken(t, testSecret, "admin", time.Hour), "")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAuth_CompanyCanWriteProducts(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	// Passes the guard; the empty body is then rejected by the handler itself.
	rec := authRequest(router, http.MethodPost, "/api/products", signToken(t, testSecret, "company", time.Hour), `{}`)
	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
	require.NotEqual(t, http.StatusForbidden, rec.Code)
}

func TestAuth_PublicReadsNeedNoToken(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), httpserver.WithAuth(testSecret))

	for _, path := range []string{"/health", "/api/markets", "/api/products/search?q=arroz", "/api/products/categories"} {
		rec := authRequest(router, http.MethodGet, path, "", "")
		require.Equal(t, http.StatusOK, rec.Code, path)
	}
}

func TestInstanceHeaderIsSent(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	rec := authRequest(router, http.MethodGet, "/health", "", "")
	require.NotEmpty(t, rec.Header().Get("X-Instance"))
}
