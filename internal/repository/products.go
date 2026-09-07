package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

var (
	ErrMarketNotFound = errors.New("market not found")
	ErrNotFound       = errors.New("product not found")
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
	Source         string // "" means no filter; otherwise "seed", "company" or "bot"
	SortByPriceAsc bool
}

const productJoinSelect = `
	SELECT p.id, p.name, p.unit, p.category, m.name AS default_market_name, mp.price AS default_price, p.source
	FROM products p
	JOIN markets m ON m.id = p.default_market_id
	JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
`

// List returns products joined with their default market's name and price,
// applying the given category/search/source filters and ordering.
func (r *ProductsRepo) List(ctx context.Context, filter ProductFilter) ([]models.Product, error) {
	query := productJoinSelect + " WHERE 1 = 1"

	var args []any
	if filter.Category != "" && !strings.EqualFold(filter.Category, "Tudo") {
		args = append(args, filter.Category)
		query += fmt.Sprintf(" AND UPPER(p.category) = UPPER($%d)", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		query += fmt.Sprintf(" AND p.name ILIKE $%d", len(args))
	}
	if filter.Source != "" {
		args = append(args, filter.Source)
		query += fmt.Sprintf(" AND p.source = $%d", len(args))
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Unit, &p.Category, &p.DefaultMarketName, &p.DefaultPrice, &p.Source); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// GetByID returns a single product joined with its default market's name
// and price.
func (r *ProductsRepo) GetByID(ctx context.Context, id string) (models.Product, error) {
	var p models.Product
	err := r.pool.QueryRow(ctx, productJoinSelect+" WHERE p.id = $1", id).Scan(
		&p.ID, &p.Name, &p.Unit, &p.Category, &p.DefaultMarketName, &p.DefaultPrice, &p.Source,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, ErrNotFound
		}
		return models.Product{}, err
	}
	return p, nil
}

// CreateProductInput describes a product+price to register at one market
// (Admin's on-demand collection or the Partner Company page).
type CreateProductInput struct {
	Name     string
	Unit     string
	Category string
	MarketID string
	Price    float64
	Source   string // "company" or "bot"
}

// UpsertForMarket creates a product tied to MarketID (as its default
// market) and sets its price there. Resubmitting the same market+name
// updates unit/category/price instead of creating a duplicate — but the
// same name at a *different* market always becomes its own separate
// product, since matching "the same" product across markets isn't
// something this endpoint can decide.
func (r *ProductsRepo) UpsertForMarket(ctx context.Context, input CreateProductInput) (models.Product, error) {
	var marketExists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM markets WHERE id = $1)`, input.MarketID).Scan(&marketExists); err != nil {
		return models.Product{}, err
	}
	if !marketExists {
		return models.Product{}, ErrMarketNotFound
	}

	id := generateID(input.Name, "produto")

	var productID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO products (id, name, unit, category, default_market_id, source)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (default_market_id, lower(name)) DO UPDATE
			SET unit = EXCLUDED.unit, category = EXCLUDED.category
		RETURNING id
	`, id, input.Name, input.Unit, input.Category, input.MarketID, input.Source).Scan(&productID)
	if err != nil {
		return models.Product{}, err
	}

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO market_products (market_id, product_id, price)
		VALUES ($1, $2, $3)
		ON CONFLICT (market_id, product_id) DO UPDATE SET price = EXCLUDED.price
	`, input.MarketID, productID, input.Price); err != nil {
		return models.Product{}, err
	}

	return r.GetByID(ctx, productID)
}
