package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

type ProductsRepo struct {
	pool *pgxpool.Pool
}

func NewProductsRepo(pool *pgxpool.Pool) *ProductsRepo {
	return &ProductsRepo{pool: pool}
}

type ProductFilter struct {
	Category       string // "" or "Tudo" means no filter
	Search         string // matched case-insensitively against the product name
	SortByPriceAsc bool
}

// List returns products joined with their default market's name and price,
// applying the given category/search filters and ordering.
func (r *ProductsRepo) List(ctx context.Context, filter ProductFilter) ([]models.Product, error) {
	query := `
		SELECT p.id, p.name, p.unit, p.category, m.name AS default_market_name, mp.price AS default_price
		FROM products p
		JOIN markets m ON m.id = p.default_market_id
		JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
		WHERE 1 = 1
	`

	var args []any
	if filter.Category != "" && !strings.EqualFold(filter.Category, "Tudo") {
		args = append(args, filter.Category)
		query += fmt.Sprintf(" AND UPPER(p.category) = UPPER($%d)", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		query += fmt.Sprintf(" AND p.name ILIKE $%d", len(args))
	}

	if filter.SortByPriceAsc {
		query += " ORDER BY mp.price ASC"
	} else {
		query += " ORDER BY p.name ASC"
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Unit, &p.Category, &p.DefaultMarketName, &p.DefaultPrice); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
