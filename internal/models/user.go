package models

import "time"

// User is the public-facing representation of an account — never carries
// the password hash.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse is returned by register/login: a bearer token plus the user
// it belongs to.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
}
