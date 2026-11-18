package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/models"
)

type SavingsHistoryRepo struct {
	pool *pgxpool.Pool
}

func NewSavingsHistoryRepo(pool *pgxpool.Pool) *SavingsHistoryRepo {
	return &SavingsHistoryRepo{pool: pool}
}

// List returns saved comparisons, most recent first.
func (r *SavingsHistoryRepo) List(ctx context.Context) ([]models.SavingsHistoryItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT date_label, market_name, items_count, total_cost, savings
		FROM savings_history
		ORDER BY occurred_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.SavingsHistoryItem
	for rows.Next() {
		var it models.SavingsHistoryItem
		if err := rows.Scan(&it.DateLabel, &it.MarketName, &it.ItemsCount, &it.TotalCost, &it.Savings); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
