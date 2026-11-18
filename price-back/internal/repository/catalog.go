package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

var (
	ErrCatalogItemNotFound = errors.New("catalog item not found")
	ErrSuggestionNotFound  = errors.New("suggestion not found")
	ErrSuggestionDecided   = errors.New("suggestion already decided")
)

// similarNameThreshold is how alike two generic keys (same brand, same size)
// must be before they are worth a person's time. It is deliberately high:
// measured on real data, lower values mostly pair up different *variants*
// of one product ("cerveja" vs "cerveja zero", "desnatado" vs
// "semidesnatado", "ipa" vs "easy ipa") that must stay separate.
const similarNameThreshold = 0.85

// Outcomes of LinkProduct.
const (
	LinkLinked  = "linked"  // attached to a catalog item
	LinkSkipped = "skipped" // nothing to identify it by (no generic key)
	LinkLocked  = "locked"  // a person decided its item; left alone
)

// LinkOutcome describes what LinkProduct did.
type LinkOutcome struct {
	Status      string
	ItemID      string
	Created     bool // a new catalog item was created for it
	Suggestions int  // doubtful matches queued for review
}

type CatalogRepo struct {
	pool *pgxpool.Pool
}

func NewCatalogRepo(pool *pgxpool.Pool) *CatalogRepo {
	return &CatalogRepo{pool: pool}
}

type linkable struct {
	ID           string
	Name         string
	Category     string
	Brand        string
	GenericKey   string
	PackCount    int
	SizeKind     string
	SizeAmount   *float64
	SizeLabel    string
	SoldByWeight bool
	CatalogID    *string
	Locked       bool
}

// LinkProduct attaches a market listing to its catalog item, creating the
// item if this is the first listing of that product.
//
// The identity is brand + generic key + package size, so the same product
// sold at two markets lands on one item automatically. Listings that are
// *probably* the same but differ in a way the identity can't see (the brand
// is missing from one name, or the names are nearly identical) also get a
// suggestion for a person to confirm — never an automatic merge.
func (r *CatalogRepo) LinkProduct(ctx context.Context, productID string) (LinkOutcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return LinkOutcome{}, err
	}
	defer tx.Rollback(ctx)

	var p linkable
	err = tx.QueryRow(ctx, `
		SELECT id, name, category, COALESCE(brand, ''), COALESCE(generic_key, ''), pack_count,
		       COALESCE(size_kind, ''), size_amount, COALESCE(size_label, ''), sold_by_weight,
		       catalog_id, catalog_locked
		FROM products WHERE id = $1 FOR UPDATE
	`, productID).Scan(
		&p.ID, &p.Name, &p.Category, &p.Brand, &p.GenericKey, &p.PackCount,
		&p.SizeKind, &p.SizeAmount, &p.SizeLabel, &p.SoldByWeight, &p.CatalogID, &p.Locked,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LinkOutcome{}, ErrNotFound
		}
		return LinkOutcome{}, err
	}

	if p.Locked {
		itemID := ""
		if p.CatalogID != nil {
			itemID = *p.CatalogID
		}
		return LinkOutcome{Status: LinkLocked, ItemID: itemID}, nil
	}
	if p.GenericKey == "" {
		return LinkOutcome{Status: LinkSkipped}, nil
	}

	sizeAmount := 0.0
	if p.SizeAmount != nil {
		sizeAmount = *p.SizeAmount
	}

	var itemID string
	var created bool
	err = tx.QueryRow(ctx, `
		INSERT INTO catalog_items (
			id, name, category, brand, generic_key, size_kind, size_amount, pack_count, sold_by_weight, size_label
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT ON CONSTRAINT catalog_items_identity DO UPDATE SET id = catalog_items.id
		RETURNING id, (xmax = 0)
	`, generateID(p.Name, "item"), p.Name, p.Category, p.Brand, p.GenericKey, p.SizeKind, sizeAmount,
		p.PackCount, p.SoldByWeight, p.SizeLabel,
	).Scan(&itemID, &created)
	if err != nil {
		return LinkOutcome{}, err
	}

	if p.CatalogID == nil || *p.CatalogID != itemID {
		if _, err := tx.Exec(ctx, `UPDATE products SET catalog_id = $2 WHERE id = $1`, p.ID, itemID); err != nil {
			return LinkOutcome{}, err
		}
		if p.CatalogID != nil {
			if err := deleteIfOrphan(ctx, tx, *p.CatalogID); err != nil {
				return LinkOutcome{}, err
			}
		}
	}

	suggestions := 0
	if created {
		suggestions, err = queueSuggestions(ctx, tx, p, itemID, sizeAmount)
		if err != nil {
			return LinkOutcome{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return LinkOutcome{}, err
	}
	return LinkOutcome{Status: LinkLinked, ItemID: itemID, Created: created, Suggestions: suggestions}, nil
}

// deleteIfOrphan removes a catalog item no listing points at any more.
func deleteIfOrphan(ctx context.Context, tx pgx.Tx, itemID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM catalog_items
		WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM products WHERE catalog_id = $1)
	`, itemID)
	return err
}

// queueSuggestions looks for existing catalog items the newly created one
// might really be the same product as, and queues them for review. Same
// package size is required; the brand must agree or be missing on one side.
func queueSuggestions(ctx context.Context, tx pgx.Tx, p linkable, itemID string, sizeAmount float64) (int, error) {
	type candidate struct {
		id     string
		score  float64
		reason string
	}

	rows, err := tx.Query(ctx, `
		SELECT c.id, similarity(c.generic_key, $2)::float8,
		       CASE WHEN c.generic_key = $2 THEN 'brand-missing' ELSE 'similar-name' END
		FROM catalog_items c
		WHERE c.id <> $1
		  AND c.size_kind = $3 AND c.size_amount = $4 AND c.pack_count = $5 AND c.sold_by_weight = $6
		  AND (c.brand = $7 OR c.brand = '' OR $7 = '')
		  AND (c.generic_key = $2 OR similarity(c.generic_key, $2) >= $8)
	`, itemID, p.GenericKey, p.SizeKind, sizeAmount, p.PackCount, p.SoldByWeight, p.Brand, similarNameThreshold)
	if err != nil {
		return 0, err
	}
	var found []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.score, &c.reason); err != nil {
			rows.Close()
			return 0, err
		}
		found = append(found, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	queued := 0
	for _, c := range found {
		tag, err := tx.Exec(ctx, `
			INSERT INTO catalog_suggestions (product_id, catalog_id, score, reason)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (product_id, catalog_id) DO NOTHING
		`, p.ID, c.id, c.score, c.reason)
		if err != nil {
			return 0, err
		}
		queued += int(tag.RowsAffected())
	}
	return queued, nil
}

// MatchUnlinked links every listing that has a generic key but no catalog
// item yet (or, with all, re-evaluates every listing a person hasn't
// locked). It is idempotent, so it doubles as the backfill for products that
// existed before the catalog.
func (r *CatalogRepo) MatchUnlinked(ctx context.Context, all bool) (models.MatchSummary, error) {
	query := `SELECT id FROM products WHERE COALESCE(generic_key, '') <> '' AND NOT catalog_locked`
	if !all {
		query += ` AND catalog_id IS NULL`
	}
	query += ` ORDER BY created_at, id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return models.MatchSummary{}, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return models.MatchSummary{}, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return models.MatchSummary{}, err
	}

	var summary models.MatchSummary
	for _, id := range ids {
		out, err := r.LinkProduct(ctx, id)
		if err != nil {
			return summary, err
		}
		summary.Processed++
		if out.Status == LinkLinked {
			summary.Linked++
		}
		if out.Created {
			summary.ItemsCreated++
		}
		summary.Suggestions += out.Suggestions
	}

	err = r.pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE COALESCE(generic_key, '') = ''`).Scan(&summary.Skipped)
	return summary, err
}

// CatalogFilter narrows ListItems.
type CatalogFilter struct {
	Search     string // matched against name, brand and generic key
	MinMarkets int    // only items sold by at least this many markets
	Limit      int    // default 100, max 500
	Offset     int
}

const catalogItemColumns = `
	ci.id, ci.name, ci.category, ci.brand, ci.generic_key, ci.size_kind, ci.size_amount,
	ci.size_label, ci.pack_count, ci.sold_by_weight
`

const marketsCountSQL = `(SELECT count(DISTINCT p.default_market_id) FROM products p WHERE p.catalog_id = ci.id)`

func scanCatalogItem(row interface{ Scan(...any) error }) (models.CatalogItem, error) {
	var it models.CatalogItem
	var amount float64
	if err := row.Scan(
		&it.ID, &it.Name, &it.Category, &it.Brand, &it.GenericKey, &it.SizeKind, &amount,
		&it.SizeLabel, &it.PackCount, &it.SoldByWeight,
	); err != nil {
		return models.CatalogItem{}, err
	}
	if it.SizeKind != "" {
		it.SizeAmount = &amount
	}
	it.Offers = []models.CatalogOffer{}
	return it, nil
}

// ListItems returns catalog items — those sold by the most markets first —
// each with its offers, cheapest first.
func (r *CatalogRepo) ListItems(ctx context.Context, f CatalogFilter) ([]models.CatalogItem, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+catalogItemColumns+`
		FROM catalog_items ci
		WHERE ($1 = '' OR ci.name ILIKE '%' || $1 || '%' OR ci.generic_key ILIKE '%' || $1 || '%'
		       OR ci.brand ILIKE '%' || $1 || '%')
		  AND `+marketsCountSQL+` >= $2
		ORDER BY `+marketsCountSQL+` DESC, ci.name
		LIMIT $3 OFFSET $4
	`, f.Search, f.MinMarkets, limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.CatalogItem{}
	for rows.Next() {
		it, err := scanCatalogItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	return items, r.attachOffers(ctx, items)
}

// GetItem returns one catalog item with its offers.
func (r *CatalogRepo) GetItem(ctx context.Context, id string) (models.CatalogItem, error) {
	it, err := scanCatalogItem(r.pool.QueryRow(ctx, `SELECT `+catalogItemColumns+` FROM catalog_items ci WHERE ci.id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.CatalogItem{}, ErrCatalogItemNotFound
		}
		return models.CatalogItem{}, err
	}
	items := []models.CatalogItem{it}
	if err := r.attachOffers(ctx, items); err != nil {
		return models.CatalogItem{}, err
	}
	return items[0], nil
}

// attachOffers fills Offers (cheapest first), MarketsCount and the price
// range on every item, with a single query.
func (r *CatalogRepo) attachOffers(ctx context.Context, items []models.CatalogItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	byID := make(map[string]*models.CatalogItem, len(items))
	for i := range items {
		ids[i] = items[i].ID
		byID[items[i].ID] = &items[i]
	}

	rows, err := r.pool.Query(ctx, `
		SELECT p.catalog_id, p.id, p.name, m.id, m.name, mp.price, mp.collected_at,
		       COALESCE(p.size_kind, ''), p.size_amount, p.pack_count, p.sold_by_weight
		FROM products p
		JOIN markets m ON m.id = p.default_market_id
		JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
		WHERE p.catalog_id = ANY($1)
		ORDER BY mp.price, m.name
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	markets := make(map[string]map[string]bool, len(items))
	for rows.Next() {
		var catalogID string
		var o models.CatalogOffer
		tmp := models.Product{}
		if err := rows.Scan(
			&catalogID, &o.ProductID, &o.ProductName, &o.MarketID, &o.MarketName, &o.Price, &o.CollectedAt,
			&tmp.SizeKind, &tmp.SizeAmount, &tmp.PackCount, &tmp.SoldByWeight,
		); err != nil {
			return err
		}
		tmp.DefaultPrice = o.Price
		tmp.ComputeUnitPrice()
		o.UnitPrice, o.UnitPriceUnit = tmp.UnitPrice, tmp.UnitPriceUnit

		it := byID[catalogID]
		it.Offers = append(it.Offers, o)
		if markets[catalogID] == nil {
			markets[catalogID] = map[string]bool{}
		}
		markets[catalogID][o.MarketID] = true

		price := o.Price
		if it.MinPrice == nil || price < *it.MinPrice {
			it.MinPrice = &price
		}
		if it.MaxPrice == nil || price > *it.MaxPrice {
			p2 := price
			it.MaxPrice = &p2
		}
	}
	for id, set := range markets {
		byID[id].MarketsCount = len(set)
	}
	return rows.Err()
}

// ListSuggestions returns review-queue entries with the listing and the
// catalog item side by side (with its current offers), best score first.
func (r *CatalogRepo) ListSuggestions(ctx context.Context, status string, limit int) ([]models.CatalogSuggestion, error) {
	if status == "" {
		status = "pending"
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.status, s.score::float8, s.reason, s.created_at, s.product_id, s.catalog_id
		FROM catalog_suggestions s
		WHERE s.status = $1
		ORDER BY s.score DESC, s.id
		LIMIT $2
	`, status, limit)
	if err != nil {
		return nil, err
	}
	type raw struct {
		s         models.CatalogSuggestion
		productID string
		catalogID string
	}
	var raws []raw
	for rows.Next() {
		var x raw
		if err := rows.Scan(&x.s.ID, &x.s.Status, &x.s.Score, &x.s.Reason, &x.s.CreatedAt, &x.productID, &x.catalogID); err != nil {
			rows.Close()
			return nil, err
		}
		raws = append(raws, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]models.CatalogSuggestion, 0, len(raws))
	for _, x := range raws {
		offer, err := r.productOffer(ctx, x.productID)
		if err != nil {
			return nil, err
		}
		item, err := r.GetItem(ctx, x.catalogID)
		if err != nil {
			return nil, err
		}
		x.s.Product, x.s.Item = offer, item
		out = append(out, x.s)
	}
	return out, nil
}

// productOffer describes one listing as an offer (used on the product side
// of a suggestion).
func (r *CatalogRepo) productOffer(ctx context.Context, productID string) (models.CatalogOffer, error) {
	var o models.CatalogOffer
	tmp := models.Product{}
	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.name, m.id, m.name, mp.price, mp.collected_at,
		       COALESCE(p.size_kind, ''), p.size_amount, p.pack_count, p.sold_by_weight
		FROM products p
		JOIN markets m ON m.id = p.default_market_id
		JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
		WHERE p.id = $1
	`, productID).Scan(
		&o.ProductID, &o.ProductName, &o.MarketID, &o.MarketName, &o.Price, &o.CollectedAt,
		&tmp.SizeKind, &tmp.SizeAmount, &tmp.PackCount, &tmp.SoldByWeight,
	)
	if err != nil {
		return models.CatalogOffer{}, err
	}
	tmp.DefaultPrice = o.Price
	tmp.ComputeUnitPrice()
	o.UnitPrice, o.UnitPriceUnit = tmp.UnitPrice, tmp.UnitPriceUnit
	return o, nil
}

// AcceptSuggestion moves the listing onto the suggested catalog item and
// locks it there, so re-collecting the product can't undo the decision. The
// listing's other pending suggestions become moot and are closed.
func (r *CatalogRepo) AcceptSuggestion(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var productID, catalogID, status string
	err = tx.QueryRow(ctx, `
		SELECT product_id, catalog_id, status FROM catalog_suggestions WHERE id = $1 FOR UPDATE
	`, id).Scan(&productID, &catalogID, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSuggestionNotFound
		}
		return err
	}
	if status != "pending" {
		return ErrSuggestionDecided
	}

	var oldCatalogID *string
	if err := tx.QueryRow(ctx, `SELECT catalog_id FROM products WHERE id = $1 FOR UPDATE`, productID).Scan(&oldCatalogID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE products SET catalog_id = $2, catalog_locked = true WHERE id = $1`, productID, catalogID); err != nil {
		return err
	}
	if oldCatalogID != nil && *oldCatalogID != catalogID {
		if err := deleteIfOrphan(ctx, tx, *oldCatalogID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE catalog_suggestions SET status = 'accepted', decided_at = now() WHERE id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE catalog_suggestions SET status = 'rejected', decided_at = now()
		WHERE product_id = $1 AND status = 'pending'
	`, productID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RejectSuggestion records that the two are different products; the pair is
// never suggested again.
func (r *CatalogRepo) RejectSuggestion(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE catalog_suggestions SET status = 'rejected', decided_at = now()
		WHERE id = $1 AND status = 'pending'
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM catalog_suggestions WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrSuggestionNotFound
	}
	return ErrSuggestionDecided
}
