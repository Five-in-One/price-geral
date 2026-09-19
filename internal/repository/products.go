package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	MarketID       string // "" means no filter; otherwise the product's default market
	SortByPriceAsc bool
}

const productJoinSelect = `
	SELECT p.id, p.name, p.unit, p.category, m.name AS default_market_name, mp.price AS default_price, p.source,
		COALESCE(p.brand, ''), COALESCE(p.generic_key, ''), p.pack_count, COALESCE(p.size_kind, ''),
		p.size_amount, COALESCE(p.size_label, ''), p.sold_by_weight, mp.collected_at,
		COALESCE(p.catalog_id, '')
	FROM products p
	JOIN markets m ON m.id = p.default_market_id
	JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
`

// scanProduct reads one row of productJoinSelect and derives the unit price.
func scanProduct(row interface{ Scan(...any) error }) (models.Product, error) {
	var p models.Product
	err := row.Scan(
		&p.ID, &p.Name, &p.Unit, &p.Category, &p.DefaultMarketName, &p.DefaultPrice, &p.Source,
		&p.Brand, &p.GenericKey, &p.PackCount, &p.SizeKind, &p.SizeAmount, &p.SizeLabel,
		&p.SoldByWeight, &p.CollectedAt, &p.CatalogID,
	)
	if err != nil {
		return models.Product{}, err
	}
	p.ComputeUnitPrice()
	return p, nil
}

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
	if filter.MarketID != "" {
		args = append(args, filter.MarketID)
		query += fmt.Sprintf(" AND p.default_market_id = $%d", len(args))
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
		p, err := scanProduct(rows)
		if err != nil {
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
	p, err := scanProduct(r.pool.QueryRow(ctx, productJoinSelect+" WHERE p.id = $1", id))
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

	// Parsed fields (all optional, see migration 0008). HasParsed says whether
	// the caller sent any of them: a resubmission without them (e.g. from the
	// partner page) must not wipe what the bot stored before.
	HasParsed    bool
	Brand        string
	GenericKey   string
	PackCount    int // 0 means 1
	SizeKind     string
	SizeAmount   *float64
	SizeLabel    string
	SoldByWeight bool
	// CollectedAt is when the price was observed; nil means now.
	CollectedAt *time.Time
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

	packCount := input.PackCount
	if packCount < 1 {
		packCount = 1
	}
	var sizeKind *string
	if input.SizeKind != "" {
		sizeKind = &input.SizeKind
	}

	var productID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO products (
			id, name, unit, category, default_market_id, source,
			brand, generic_key, pack_count, size_kind, size_amount, size_label, sold_by_weight
		)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9, $10, $11, NULLIF($12, ''), $13)
		ON CONFLICT (default_market_id, lower(name)) DO UPDATE SET
			unit = EXCLUDED.unit,
			category = EXCLUDED.category,
			brand          = CASE WHEN $14 THEN EXCLUDED.brand          ELSE products.brand          END,
			generic_key    = CASE WHEN $14 THEN EXCLUDED.generic_key    ELSE products.generic_key    END,
			pack_count     = CASE WHEN $14 THEN EXCLUDED.pack_count     ELSE products.pack_count     END,
			size_kind      = CASE WHEN $14 THEN EXCLUDED.size_kind      ELSE products.size_kind      END,
			size_amount    = CASE WHEN $14 THEN EXCLUDED.size_amount    ELSE products.size_amount    END,
			size_label     = CASE WHEN $14 THEN EXCLUDED.size_label     ELSE products.size_label     END,
			sold_by_weight = CASE WHEN $14 THEN EXCLUDED.sold_by_weight ELSE products.sold_by_weight END
		RETURNING id
	`, id, input.Name, input.Unit, input.Category, input.MarketID, input.Source,
		input.Brand, input.GenericKey, packCount, sizeKind, input.SizeAmount, input.SizeLabel, input.SoldByWeight,
		input.HasParsed,
	).Scan(&productID)
	if err != nil {
		return models.Product{}, err
	}

	if _, err := r.pool.Exec(ctx, `
		INSERT INTO market_products (market_id, product_id, price, collected_at)
		VALUES ($1, $2, $3, COALESCE($4::timestamptz, now()))
		ON CONFLICT (market_id, product_id) DO UPDATE
			SET price = EXCLUDED.price, collected_at = EXCLUDED.collected_at
	`, input.MarketID, productID, input.Price, input.CollectedAt); err != nil {
		return models.Product{}, err
	}

	return r.GetByID(ctx, productID)
}

// UpdateProductInput describes fields to change on an existing product.
// Pointer fields distinguish "not provided" from a zero value, so a PATCH
// can touch just one field (e.g. only the price) without clobbering the
// others. Reassigning a product's market isn't supported here — that's a
// bigger structural change than an edit form should trigger.
type UpdateProductInput struct {
	Name     *string
	Unit     *string
	Category *string
	Price    *float64

	// Parsed fields (see migration 0008); used to backfill products that were
	// registered before the bot started extracting them. nil leaves the stored
	// value; SizeKind "" clears the size.
	Brand        *string
	GenericKey   *string
	PackCount    *int
	SizeKind     *string
	SizeAmount   *float64
	SizeLabel    *string
	SoldByWeight *bool
}

// Update applies the given fields to a product (and its default market's
// price, if Price is set), returning ErrNotFound if no product has id.
func (r *ProductsRepo) Update(ctx context.Context, id string, input UpdateProductInput) (models.Product, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return models.Product{}, err
	}

	name, unit, category := current.Name, current.Unit, current.Category
	if input.Name != nil {
		name = *input.Name
	}
	if input.Unit != nil {
		unit = *input.Unit
	}
	if input.Category != nil {
		category = *input.Category
	}

	brand, genericKey, sizeLabel := current.Brand, current.GenericKey, current.SizeLabel
	packCount, soldByWeight := current.PackCount, current.SoldByWeight
	sizeKind, sizeAmount := current.SizeKind, current.SizeAmount
	if input.Brand != nil {
		brand = *input.Brand
	}
	if input.GenericKey != nil {
		genericKey = *input.GenericKey
	}
	if input.SizeLabel != nil {
		sizeLabel = *input.SizeLabel
	}
	if input.PackCount != nil {
		packCount = *input.PackCount
	}
	if input.SoldByWeight != nil {
		soldByWeight = *input.SoldByWeight
	}
	if input.SizeKind != nil {
		sizeKind = *input.SizeKind
		sizeAmount = input.SizeAmount
		if sizeKind == "" {
			sizeAmount = nil
		}
	}
	var sizeKindArg *string
	if sizeKind != "" {
		sizeKindArg = &sizeKind
	}

	if _, err := r.pool.Exec(ctx, `
		UPDATE products SET
			name = $1, unit = $2, category = $3,
			brand = NULLIF($4, ''), generic_key = NULLIF($5, ''), pack_count = $6,
			size_kind = $7, size_amount = $8, size_label = NULLIF($9, ''), sold_by_weight = $10
		WHERE id = $11
	`, name, unit, category, brand, genericKey, packCount, sizeKindArg, sizeAmount, sizeLabel, soldByWeight, id); err != nil {
		return models.Product{}, err
	}

	// A new price is a fresh observation; parsed-field backfills leave the
	// collection time alone.
	if input.Price != nil {
		if _, err := r.pool.Exec(ctx, `
			UPDATE market_products SET price = $1, collected_at = now() WHERE product_id = $2
		`, *input.Price, id); err != nil {
			return models.Product{}, err
		}
	}

	return r.GetByID(ctx, id)
}

// Delete removes a product — its price rows in market_products go with it
// via ON DELETE CASCADE — returning ErrNotFound if no product has id.
func (r *ProductsRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
