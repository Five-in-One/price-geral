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
		"username": "Alice123",
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.User.ID)
	require.Equal(t, "alice123", resp.User.Username) // normalized to lowercase
	require.Equal(t, models.RoleCompany, resp.User.Role)
}

func TestRegister_IgnoresRoleInBody_AlwaysCreatesCompany(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"username": "wannabe-admin",
		"password": "correct-horse",
		"role":     "admin",
	}, "")

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, models.RoleCompany, resp.User.Role)
}

func TestRegister_DuplicateUsername_ReturnsConflict(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	body := map[string]string{"username": "bob", "password": "correct-horse"}
	first := doJSONRequest(t, router, http.MethodPost, "/auth/register", body, "")
	require.Equal(t, http.StatusCreated, first.Code)

	second := doJSONRequest(t, router, http.MethodPost, "/auth/register", body, "")
	require.Equal(t, http.StatusConflict, second.Code)
}

func TestRegister_InvalidUsername_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"username": "ab",
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_UsernameWithSpaces_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"username": "not a username",
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_ShortPassword_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"username": "carol",
		"password": "short",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_CorrectCredentials_ReturnsToken(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"username": "dave", "password": "correct-horse"}
	require.Equal(t, http.StatusCreated, doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "").Code)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "DAVE", // case-insensitive
		"password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusOK, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, "dave", resp.User.Username)
	require.Equal(t, models.RoleCompany, resp.User.Role)
}

func TestLogin_SeededAdmin_ReturnsAdminRole(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "admin",
		"password": "TrocarSenha123!",
	}, "")

	require.Equal(t, http.StatusOK, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, models.RoleAdmin, resp.User.Role)
}

func TestLogin_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"username": "erin", "password": "correct-horse"}
	require.Equal(t, http.StatusCreated, doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "").Code)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "erin",
		"password": "wrong-password",
	}, "")

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogin_UnknownUsername_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "nobody",
		"password": "whatever123",
	}, "")

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_ValidToken_ReturnsUser(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"username": "frank", "password": "correct-horse"}
	registerRec := doJSONRequest(t, router, http.MethodPost, "/auth/register", registerReq, "")
	require.Equal(t, http.StatusCreated, registerRec.Code)

	var registered models.AuthResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, registered.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code)

	var user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	require.Equal(t, registered.User.ID, user.ID)
	require.Equal(t, "frank", user.Username)
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

func loginAsSeededAdmin(t *testing.T, router http.Handler) string {
	t.Helper()
	rec := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "admin",
		"password": "TrocarSenha123!",
	}, "")
	require.Equal(t, http.StatusOK, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.AccessToken
}

func TestCreateUser_AsAdmin_CanCreateAdminAccount(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")
	adminToken := loginAsSeededAdmin(t, router)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "second-admin",
		"password": "correct-horse",
		"role":     "admin",
	}, adminToken)

	require.Equal(t, http.StatusCreated, rec.Code)

	var user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	require.Equal(t, "second-admin", user.Username)
	require.Equal(t, models.RoleAdmin, user.Role)
}

func TestCreateUser_AsAdmin_CanCreateCompanyAccount(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")
	adminToken := loginAsSeededAdmin(t, router)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "partner",
		"password": "correct-horse",
		"role":     "company",
	}, adminToken)

	require.Equal(t, http.StatusCreated, rec.Code)

	var user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	require.Equal(t, models.RoleCompany, user.Role)
}

func TestCreateUser_WithoutToken_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "x", "password": "correct-horse", "role": "company",
	}, "")

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateUser_AsCompanyUser_ReturnsForbidden(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerRec := doJSONRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"username": "regular-company", "password": "correct-horse",
	}, "")
	require.Equal(t, http.StatusCreated, registerRec.Code)
	var registered models.AuthResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "x", "password": "correct-horse", "role": "company",
	}, registered.AccessToken)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateUser_InvalidRole_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")
	adminToken := loginAsSeededAdmin(t, router)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "x", "password": "correct-horse", "role": "superadmin",
	}, adminToken)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateUser_DuplicateUsername_ReturnsConflict(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")
	adminToken := loginAsSeededAdmin(t, router)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "admin", "password": "correct-horse", "role": "company",
	}, adminToken)

	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestMe_TokenSignedWithDifferentSecret_ReturnsUnauthorized(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	registerReq := map[string]string{"username": "gina", "password": "correct-horse"}
	otherRouter := httpserver.NewRouter(pool, "a-completely-different-secret")
	registerRec := doJSONRequest(t, otherRouter, http.MethodPost, "/auth/register", registerReq, "")
	require.Equal(t, http.StatusCreated, registerRec.Code)

	var registered models.AuthResponse
	require.NoError(t, json.Unmarshal(registerRec.Body.Bytes(), &registered))

	rec := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, registered.AccessToken)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSignup_CreatesUserRoleAccount(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"username": "Maria", "password": "correct-horse",
	}, "")

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, "maria", resp.User.Username)
	require.Equal(t, models.RoleUser, resp.User.Role)

	// The token works against /auth/me and can log in again later.
	me := doJSONRequest(t, router, http.MethodGet, "/auth/me", nil, resp.AccessToken)
	require.Equal(t, http.StatusOK, me.Code)
	login := doJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"username": "maria", "password": "correct-horse",
	}, "")
	require.Equal(t, http.StatusOK, login.Code)
}

func TestSignup_IgnoresRoleInBody(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"username": "sneaky", "password": "correct-horse", "role": "admin",
	}, "")

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp models.AuthResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, models.RoleUser, resp.User.Role)
}

func TestSignup_DuplicateUsernameAndWeakPassword(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	body := map[string]string{"username": "joao", "password": "correct-horse"}
	require.Equal(t, http.StatusCreated, doJSONRequest(t, router, http.MethodPost, "/auth/signup", body, "").Code)
	require.Equal(t, http.StatusConflict, doJSONRequest(t, router, http.MethodPost, "/auth/signup", body, "").Code)

	short := doJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{"username": "ana", "password": "1234"}, "")
	require.Equal(t, http.StatusBadRequest, short.Code)
}

func TestCreateUser_AsAdmin_CanCreateUserRoleAccount(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")
	adminToken := loginAsSeededAdmin(t, router)

	rec := doJSONRequest(t, router, http.MethodPost, "/auth/admin/users", map[string]string{
		"username": "app-user", "password": "correct-horse", "role": "user",
	}, adminToken)

	require.Equal(t, http.StatusCreated, rec.Code)
	var user models.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	require.Equal(t, models.RoleUser, user.Role)
}
