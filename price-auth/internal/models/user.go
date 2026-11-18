package models

import "time"

// Role distinguishes internal Price staff, partner markets and the people
// who use the mobile app.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleCompany Role = "company"
	RoleUser    Role = "user"
)

// IsValid reports whether r is one of the known roles.
func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleCompany || r == RoleUser
}

// User is the public-facing representation of an account — never carries
// the password hash.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	// MarketID is the price-back market a "company" account belongs to; nil
	// for every other account and for companies registered without one.
	MarketID  *string   `json:"market_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse is returned by register/login: a bearer token plus the user
// it belongs to.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
}
