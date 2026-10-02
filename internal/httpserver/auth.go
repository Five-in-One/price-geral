package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"priceauth/internal/auth"
	"priceauth/internal/models"
	"priceauth/internal/repository"
)

const (
	minPasswordLength = 8
	minUsernameLength = 3
	maxUsernameLength = 32
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// registerHandler is the public self-registration route for partner
// companies; signupHandler is the one the mobile app uses for end users.
// Both are public, so neither honors a role sent in the body.
func registerHandler(repo *repository.UsersRepo, jwtSecret string) http.HandlerFunc {
	return selfRegisterHandler(repo, jwtSecret, models.RoleCompany)
}

func signupHandler(repo *repository.UsersRepo, jwtSecret string) http.HandlerFunc {
	return selfRegisterHandler(repo, jwtSecret, models.RoleUser)
}

func selfRegisterHandler(repo *repository.UsersRepo, jwtSecret string, role models.Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		username := normalizeUsername(body.Username)
		if !isValidUsername(username) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username must be 3-32 characters with no spaces"})
			return
		}
		if len(body.Password) < minPasswordLength {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
			return
		}

		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
			return
		}

		// Public self-registration never creates an "admin" account — those
		// can only be created by an existing admin, via POST /auth/admin/users.
		user, err := repo.Create(r.Context(), username, hash, role, nil)
		if err != nil {
			if errors.Is(err, repository.ErrUsernameTaken) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "username already taken"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
			return
		}

		respondWithToken(w, http.StatusCreated, jwtSecret, user)
	}
}

func loginHandler(repo *repository.UsersRepo, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		username := normalizeUsername(body.Username)

		user, err := repo.VerifyCredentials(r.Context(), username, body.Password, auth.CheckPassword)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
			return
		}

		respondWithToken(w, http.StatusOK, jwtSecret, user)
	}
}

func meHandler(repo *repository.UsersRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		user, err := repo.FindByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load user"})
			return
		}

		writeJSON(w, http.StatusOK, user)
	}
}

func respondWithToken(w http.ResponseWriter, status int, jwtSecret string, user models.User) {
	token, err := auth.GenerateToken(jwtSecret, user.ID, user.Username, user.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
		return
	}
	writeJSON(w, status, models.AuthResponse{AccessToken: token, User: user})
}

type createUserRequest struct {
	Username string      `json:"username"`
	Password string      `json:"password"`
	Role     models.Role `json:"role"`
	// MarketID is the price-back market of a "company" account (optional);
	// rejected for any other role.
	MarketID string `json:"market_id"`
}

// createUserHandler lets an authenticated admin create another account —
// with any role. This is the only way an "admin" account can come into
// existence besides the seeded bootstrap user; public self-registration
// creates "company" (registerHandler) or "user" (signupHandler) accounts.
func createUserHandler(repo *repository.UsersRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body createUserRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		username := normalizeUsername(body.Username)
		if !isValidUsername(username) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username must be 3-32 characters with no spaces"})
			return
		}
		if len(body.Password) < minPasswordLength {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
			return
		}
		if !body.Role.IsValid() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be 'admin', 'company' or 'user'"})
			return
		}

		var marketID *string
		if id := strings.TrimSpace(body.MarketID); id != "" {
			if body.Role != models.RoleCompany {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "market_id is only allowed for 'company' accounts"})
				return
			}
			marketID = &id
		}

		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
			return
		}

		user, err := repo.Create(r.Context(), username, hash, body.Role, marketID)
		if err != nil {
			if errors.Is(err, repository.ErrUsernameTaken) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "username already taken"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
			return
		}

		writeJSON(w, http.StatusCreated, user)
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func isValidUsername(username string) bool {
	if len(username) < minUsernameLength || len(username) > maxUsernameLength {
		return false
	}
	for _, r := range username {
		if unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
