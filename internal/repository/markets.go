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
		SELECT id, name, initials, latitude, longitude, primary_color, background_color
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
		if err := rows.Scan(&m.ID, &m.Name, &m.Initials, &m.Latitude, &m.Longitude, &m.PrimaryColor, &m.BackgroundColor); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return markets, nil
}
