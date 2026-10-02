package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceauth/internal/models"
)

var (
	ErrUsernameTaken = errors.New("username already taken")
	ErrNotFound      = errors.New("user not found")
)

const uniqueViolationCode = "23505"

type UsersRepo struct {
	pool *pgxpool.Pool
}

func NewUsersRepo(pool *pgxpool.Pool) *UsersRepo {
	return &UsersRepo{pool: pool}
}

// Create inserts a new user with an already-hashed password and the given
// role, and returns its public representation. marketID ties a "company"
// account to its price-back market; nil leaves the account without one.
func (r *UsersRepo) Create(ctx context.Context, username, passwordHash string, role models.Role, marketID *string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, role, market_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, role, market_id, created_at
	`, username, passwordHash, role, marketID).Scan(&u.ID, &u.Username, &u.Role, &u.MarketID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return models.User{}, ErrUsernameTaken
		}
		return models.User{}, err
	}
	return u, nil
}

// credentials pairs a user with their stored password hash, for login
// verification only — never leaves the repository package.
type credentials struct {
	User         models.User
	PasswordHash string
}

func (r *UsersRepo) findByUsernameWithHash(ctx context.Context, username string) (credentials, error) {
	var c credentials
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, role, market_id, created_at, password_hash
		FROM users
		WHERE username = $1
	`, username).Scan(&c.User.ID, &c.User.Username, &c.User.Role, &c.User.MarketID, &c.User.CreatedAt, &c.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return credentials{}, ErrNotFound
		}
		return credentials{}, err
	}
	return c, nil
}

// VerifyCredentials looks up a user by username and reports whether the
// given password matches, using the supplied checker (kept decoupled from
// the bcrypt implementation so the repository has no hashing dependency).
func (r *UsersRepo) VerifyCredentials(ctx context.Context, username, password string, checkPassword func(hash, password string) bool) (models.User, error) {
	c, err := r.findByUsernameWithHash(ctx, username)
	if err != nil {
		return models.User{}, err
	}
	if !checkPassword(c.PasswordHash, password) {
		return models.User{}, ErrNotFound
	}
	return c.User, nil
}

func (r *UsersRepo) FindByID(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, role, market_id, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.Role, &u.MarketID, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, err
	}
	return u, nil
}
