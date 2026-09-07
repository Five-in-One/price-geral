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
	ErrEmailTaken = errors.New("email already registered")
	ErrNotFound   = errors.New("user not found")
)

const uniqueViolationCode = "23505"

type UsersRepo struct {
	pool *pgxpool.Pool
}

func NewUsersRepo(pool *pgxpool.Pool) *UsersRepo {
	return &UsersRepo{pool: pool}
}

// Create inserts a new user with an already-hashed password and returns its
// public representation.
func (r *UsersRepo) Create(ctx context.Context, email, passwordHash string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, passwordHash).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return models.User{}, ErrEmailTaken
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

func (r *UsersRepo) findByEmailWithHash(ctx context.Context, email string) (credentials, error) {
	var c credentials
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, created_at, password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&c.User.ID, &c.User.Email, &c.User.CreatedAt, &c.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return credentials{}, ErrNotFound
		}
		return credentials{}, err
	}
	return c, nil
}

// VerifyCredentials looks up a user by email and reports whether the given
// password matches, using the supplied checker (kept decoupled from the
// bcrypt implementation so the repository has no hashing dependency).
func (r *UsersRepo) VerifyCredentials(ctx context.Context, email, password string, checkPassword func(hash, password string) bool) (models.User, error) {
	c, err := r.findByEmailWithHash(ctx, email)
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
		SELECT id, email, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, err
	}
	return u, nil
}
