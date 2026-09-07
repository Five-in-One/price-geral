package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"priceauth/internal/httpserver"
	"priceauth/internal/models"
	"priceauth/internal/testutil"
)

func doJSONRequest(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegister_CreatesUserAndReturnsToken(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "Alice@Example.com",
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.User.ID)
	require.Equal(t, "alice@example.com", resp.User.Email) // normalized to lowercase
}

func TestRegister_DuplicateEmail_ReturnsConflict(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	body := map[string]string{"email": "bob@example.com", "password": "correct-horse"}
	first := doJSONRequest(t, router, http.MethodPost, "/auth/register", body, "")
	require.Equal(t, http.StatusCreated, first.Code)

	second := doJSONRequest(t, router, http.MethodPost, "/auth/register", body, "")
	require.Equal(t, http.StatusConflict, second.Code)
}

func TestRegister_InvalidEmail_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "not-an-email",
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_ShortPassword_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "carol@example.com",
		"password": "short",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_CorrectCredentials_ReturnsToken(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"email": "dave@example.com", "password": "correct-horse"}
	require.Equal(t, http.StatusCreated, doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "").Code)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "DAVE@example.com", // case-insensitive
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusOK, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, "dave@example.com", resp.User.Email)
}

func TestLogin_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"email": "erin@example.com", "password": "correct-horse"}
	require.Equal(t, http.StatusCreated, doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "").Code)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "erin@example.com",
		"password": "wrong-password",
	}, "")

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_UnknownEmail_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "whatever123",
	}, "")

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_ValidToken_ReturnsUser(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"email": "frank@example.com", "password": "correct-horse"}
	registerRec := doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "")
	require.Equal(t, http.StatusCreated, registerRec.Code)

	var registered models.AuthResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, registered.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code)

	var user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	require.Equal(t, registered.User.ID, user.ID)
	require.Equal(t, "frank@example.com", user.Email)
}

func TestMe_MissingToken_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, "")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_InvalidToken_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, "garbage.token.value")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_TokenSignedWithDifferentSecret_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"email": "gina@example.com", "password": "correct-horse"}
	otherRouter := httpserver.NewRouter(pool, "a-completely-different-secret")
	registerRec := doJSONRequest(t, otherRouter, http.MethodPost, "/auth/register", registerReq, "")
	require.Equal(t, http.StatusCreated, registerRec.Code)

	var registered models.AuthResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, registered.AccessToken)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
