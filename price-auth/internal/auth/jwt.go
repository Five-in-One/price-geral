package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"priceauth/internal/models"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")

	// AccessTokenTTL is how long an issued access token stays valid.
	AccessTokenTTL = 24 * time.Hour
)

// Claims are the JWT claims price-auth issues and verifies. Other services
// that trust the same JWT_SECRET can decode these to identify the caller
// and their role without a DB round-trip.
type Claims struct {
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	Role     models.Role `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed access token for the given user.
func GenerateToken(secret, userID, username string, role models.Role) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken verifies a token's signature and expiration and returns its
// claims.
func ParseToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
