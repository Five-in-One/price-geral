package repository

import (
	"context"
	"fmt"
	"strings"

	"priceback/internal/models"
)

// Sort orders the grouped product search accepts (`sort` query parameter).
var productSearchSorts = map[string]string{
	"price_asc":      "price ASC",
	"price_desc":     "price DESC",
	"name_asc":       "lower(unaccent(name)) ASC",
	"name_desc":      "lower(unaccent(name)) DESC",
	"unit_price_asc": "unit_price ASC NULLS LAST",
	"markets_desc":   "markets_count DESC, price ASC",
}

// DefaultProductSort is the order used when the request names none.
const DefaultProductSort = "price_asc"

// ValidProductSort says whether sort is one of the supported orders.
func ValidProductSort(sort string) bool {
	_, ok := productSearchSorts[sort]
	return ok
}

// ProductSearch narrows and pages the grouped product search. Every filter is
// optional; the zero value lists everything, cheapest first.
type ProductSearch struct {
	// Search is split into words; a row matches when every word appears in
	// its name or brand, ignoring case and accents.
	Search   string
	Category string // "" or "Tudo" means every category
	Brand    string
	// MarketIDs limits the listings considered to these markets. Nil means
	// every market; an empty, non-nil slice means none (e.g. no market in
	// the radius), so the search finds nothing.
	MarketIDs []string
	// MinPrice/MaxPrice bound the cheapest price of each row.
	MinPrice *float64
	MaxPrice *float64
	Sort     string // a key of productSearchSorts; "" means DefaultProductSort
	Limit    int
	Offset   int
}

// likePattern turns a search word into an ILIKE "contains" pattern, escaping
// the LIKE wildcards so "50%" searches for the literal text.
func likePattern(word string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(word) + "%"
}

// searchListingsCTE builds the WITH clause shared by Search and
// CategoryCounts: `listings` (one row per market offer that passes the
// listing-level filters), `cheapest` (one row per catalog item: its cheapest
// offer) and `grouped` (cheapest plus how many markets sell it and its unit
// price, after the price filters). Listings outside the catalog stay rows of
// their own. withCategory=false skips the category filter, so the category
// counts show what each category would hold.
func searchListingsCTE(f ProductSearch, withCategory bool) (string, []any) {
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	var where []string
	for word := range strings.FieldsSeq(f.Search) {
		where = append(where, fmt.Sprintf(
			"unaccent(p.name || ' ' || COALESCE(p.brand, '')) ILIKE unaccent(%s)", arg(likePattern(word))))
	}
	if withCategory && f.Category != "" && !strings.EqualFold(f.Category, "Tudo") {
		where = append(where, fmt.Sprintf("UPPER(p.category) = UPPER(%s)", arg(f.Category)))
	}
	if f.Brand != "" {
		where = append(where, fmt.Sprintf("lower(unaccent(p.brand)) = lower(unaccent(%s))", arg(f.Brand)))
	}
	if f.MarketIDs != nil {
		where = append(where, fmt.Sprintf("p.default_market_id = ANY(%s)", arg(f.MarketIDs)))
	}
	listingsWhere := ""
	if len(where) > 0 {
		listingsWhere = "WHERE " + strings.Join(where, " AND ")
	}

	var priceWhere []string
	if f.MinPrice != nil {
		priceWhere = append(priceWhere, fmt.Sprintf("c.price >= %s", arg(*f.MinPrice)))
	}
	if f.MaxPrice != nil {
		priceWhere = append(priceWhere, fmt.Sprintf("c.price <= %s", arg(*f.MaxPrice)))
	}
	groupedWhere := ""
	if len(priceWhere) > 0 {
		groupedWhere = "WHERE " + strings.Join(priceWhere, " AND ")
	}

	cte := `
	WITH listings AS (
		SELECT p.id, p.name, p.unit, p.category, m.name AS market_name, mp.price, p.source,
			COALESCE(p.brand, '') AS brand, COALESCE(p.generic_key, '') AS generic_key, p.pack_count,
			COALESCE(p.size_kind, '') AS size_kind, p.size_amount, COALESCE(p.size_label, '') AS size_label,
			p.sold_by_weight, mp.collected_at, COALESCE(p.catalog_id, '') AS catalog_id,
			p.default_market_id AS market_id,
			COALESCE(p.catalog_id, 'p:' || p.id) AS group_key
		FROM products p
		JOIN markets m ON m.id = p.default_market_id
		JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
		` + listingsWhere + `
	),
	cheapest AS (
		SELECT DISTINCT ON (group_key) * FROM listings ORDER BY group_key, price, id
	),
	grouped AS (
		SELECT c.*, n.markets_count,
			CASE
				WHEN c.size_amount > 0 AND c.size_kind IN ('mass', 'volume') THEN c.price / (c.size_amount / 1000)
				WHEN c.size_amount > 0 AND c.size_kind = 'length' THEN c.price / c.size_amount
				WHEN c.sold_by_weight THEN c.price
				WHEN c.pack_count > 1 THEN c.price / c.pack_count
			END AS unit_price
		FROM cheapest c
		JOIN (
			SELECT group_key, COUNT(DISTINCT market_id) AS markets_count FROM listings GROUP BY group_key
		) n USING (group_key)
		` + groupedWhere + `
	)`
	return cte, args
}

// Search returns one page of products, one row per catalog item (its
// cheapest offer, with how many markets sell it), filtered, sorted and paged
// in the database.
func (r *ProductsRepo) Search(ctx context.Context, f ProductSearch) (models.ProductPage, error) {
	order, ok := productSearchSorts[f.Sort]
	if !ok {
		order = productSearchSorts[DefaultProductSort]
	}

	cte, args := searchListingsCTE(f, true)
	args = append(args, f.Limit, f.Offset)
	query := cte + fmt.Sprintf(`
	SELECT id, name, unit, category, market_name, price, source,
		brand, generic_key, pack_count, size_kind, size_amount, size_label,
		sold_by_weight, collected_at, catalog_id, markets_count,
		COUNT(*) OVER () AS total
	FROM grouped
	ORDER BY %s, lower(name), id
	LIMIT $%d OFFSET $%d`, order, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return models.ProductPage{}, err
	}
	defer rows.Close()

	page := models.ProductPage{Items: []models.Product{}, Limit: f.Limit, Offset: f.Offset}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Unit, &p.Category, &p.DefaultMarketName, &p.DefaultPrice, &p.Source,
			&p.Brand, &p.GenericKey, &p.PackCount, &p.SizeKind, &p.SizeAmount, &p.SizeLabel,
			&p.SoldByWeight, &p.CollectedAt, &p.CatalogID, &p.MarketsCount, &page.Total,
		); err != nil {
			return models.ProductPage{}, err
		}
		p.ComputeUnitPrice()
		page.Items = append(page.Items, p)
	}
	if err := rows.Err(); err != nil {
		return models.ProductPage{}, err
	}

	// Past the last row the window count has no row to ride on.
	if len(page.Items) == 0 && f.Offset > 0 {
		countArgs := args[:len(args)-2]
		if err := r.pool.QueryRow(ctx, cte+` SELECT COUNT(*) FROM grouped`, countArgs...).Scan(&page.Total); err != nil {
			return models.ProductPage{}, err
		}
	}

	page.HasMore = f.Offset+len(page.Items) < page.Total
	return page, nil
}

// CategoryCounts returns every category with how many search rows it has
// under the other filters of f (the category itself is ignored), largest
// first.
func (r *ProductsRepo) CategoryCounts(ctx context.Context, f ProductSearch) ([]models.CategoryCount, error) {
	cte, args := searchListingsCTE(f, false)
	rows, err := r.pool.Query(ctx, cte+`
	SELECT category, COUNT(*) FROM grouped GROUP BY category ORDER BY COUNT(*) DESC, category`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := []models.CategoryCount{}
	for rows.Next() {
		var c models.CategoryCount
		if err := rows.Scan(&c.Category, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
