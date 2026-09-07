package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"priceauth/internal/auth"
	"priceauth/internal/models"
	"priceauth/internal/repository"
)

const minPasswordLength = 8

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func registerHandler(repo *repository.UsersRepo, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		email := normalizeEmail(body.Email)
		if !isValidEmail(email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
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

		user, err := repo.Create(r.Context(), email, hash)
		if err != nil {
			if errors.Is(err, repository.ErrEmailTaken) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
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

		email := normalizeEmail(body.Email)

		user, err := repo.VerifyCredentials(r.Context(), email, body.Password, auth.CheckPassword)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
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
	token, err := auth.GenerateToken(jwtSecret, user.ID, user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
		return
	}
	writeJSON(w, status, models.AuthResponse{AccessToken: token, User: user})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	return at > 0 && at < len(email)-1 && !strings.Contains(email[at+1:], "@") && strings.Contains(email[at+1:], ".")
}
