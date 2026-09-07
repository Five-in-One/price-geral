package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

type MarketsRepo struct {
	pool *pgxpool.Pool
}

func NewMarketsRepo(pool *pgxpool.Pool) *MarketsRepo {
	return &MarketsRepo{pool: pool}
}

// ListAll returns every registered market, regardless of location.
func (r *MarketsRepo) ListAll(ctx context.Context) ([]models.Market, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, initials, latitude, longitude, primary_color, background_color, source
		FROM markets
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []models.Market
	for rows.Next() {
		var m models.Market
		if err := rows.Scan(&m.ID, &m.Name, &m.Initials, &m.Latitude, &m.Longitude, &m.PrimaryColor, &m.BackgroundColor, &m.Source); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return markets, nil
}

// CreateMarketInput describes a market to create (Admin's on-demand
// collection or the Partner Company page).
type CreateMarketInput struct {
	Name            string
	Initials        string
	Latitude        float64
	Longitude       float64
	PrimaryColor    string
	BackgroundColor string
	Source          string // "company" or "bot"
}

// FindOrCreate inserts a new market, or returns the existing one if a
// market with the same name (case-insensitive) already exists — so
// resubmitting the same market name is idempotent instead of erroring.
func (r *MarketsRepo) FindOrCreate(ctx context.Context, input CreateMarketInput) (models.Market, error) {
	id := generateID(input.Name, "mercado")

	var m models.Market
	err := r.pool.QueryRow(ctx, `
		INSERT INTO markets (id, name, initials, latitude, longitude, primary_color, background_color, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (lower(name)) DO UPDATE SET name = markets.name
		RETURNING id, name, initials, latitude, longitude, primary_color, background_color, source
	`, id, input.Name, input.Initials, input.Latitude, input.Longitude, input.PrimaryColor, input.BackgroundColor, input.Source,
	).Scan(&m.ID, &m.Name, &m.Initials, &m.Latitude, &m.Longitude, &m.PrimaryColor, &m.BackgroundColor, &m.Source)
	if err != nil {
		return models.Market{}, err
	}

	return m, nil
}
