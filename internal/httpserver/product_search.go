package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"priceback/internal/geo"
	"priceback/internal/repository"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 100
)

// parseProductSearch reads the filters shared by /api/products/search and
// /api/products/categories:
//
//	search, category, brand, market_id (repeatable or comma-separated),
//	min_price, max_price, lat + lng + radius_km (only markets in the radius;
//	markets with no registered location are kept, as in GET /api/markets),
//	sort, limit (1..100, default 20) and offset.
//
// The error is the message to answer with 400.
func parseProductSearch(req *http.Request, marketsRepo *repository.MarketsRepo) (repository.ProductSearch, error) {
	q := req.URL.Query()
	f := repository.ProductSearch{
		Search:   strings.TrimSpace(q.Get("search")),
		Category: strings.TrimSpace(q.Get("category")),
		Brand:    strings.TrimSpace(q.Get("brand")),
		Sort:     q.Get("sort"),
	}

	if f.Sort == "" {
		f.Sort = repository.DefaultProductSort
	} else if !repository.ValidProductSort(f.Sort) {
		return f, errors.New("invalid sort; use price_asc, price_desc, name_asc, name_desc, unit_price_asc or markets_desc")
	}

	var err error
	if f.Limit, err = intQueryParam(req, "limit", defaultSearchLimit); err != nil || f.Limit < 1 || f.Limit > maxSearchLimit {
		return f, errors.New("limit must be between 1 and 100")
	}
	if f.Offset, err = intQueryParam(req, "offset", 0); err != nil || f.Offset < 0 {
		return f, errors.New("offset must be zero or positive")
	}

	for _, bound := range []struct {
		key string
		dst **float64
	}{{"min_price", &f.MinPrice}, {"max_price", &f.MaxPrice}} {
		v := q.Get(bound.key)
		if v == "" {
			continue
		}
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || n < 0 {
			return f, errors.New(bound.key + " must be a non-negative number")
		}
		*bound.dst = &n
	}
	if f.MinPrice != nil && f.MaxPrice != nil && *f.MinPrice > *f.MaxPrice {
		return f, errors.New("min_price must not exceed max_price")
	}

	var marketIDs []string
	for _, v := range q["market_id"] {
		for id := range strings.SplitSeq(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				marketIDs = append(marketIDs, id)
			}
		}
	}
	if len(marketIDs) > 0 {
		f.MarketIDs = marketIDs
	}

	if q.Get("radius_km") != "" || q.Get("lat") != "" || q.Get("lng") != "" {
		lat, latErr := strconv.ParseFloat(q.Get("lat"), 64)
		lng, lngErr := strconv.ParseFloat(q.Get("lng"), 64)
		radius, radiusErr := strconv.ParseFloat(q.Get("radius_km"), 64)
		if latErr != nil || lngErr != nil || radiusErr != nil || radius <= 0 {
			return f, errors.New("lat, lng and radius_km go together; radius_km must be positive")
		}
		nearby, err := marketsInRadius(req, marketsRepo, lat, lng, radius)
		if err != nil {
			return f, err
		}
		f.MarketIDs = intersectIDs(f.MarketIDs, nearby)
	}

	return f, nil
}

// errMarketsUnavailable marks a database failure (500), not a bad request.
var errMarketsUnavailable = errors.New("failed to load markets")

func marketsInRadius(req *http.Request, repo *repository.MarketsRepo, lat, lng, radiusKm float64) ([]string, error) {
	markets, err := repo.ListAll(req.Context())
	if err != nil {
		return nil, errMarketsUnavailable
	}
	ids := []string{}
	for _, m := range markets {
		if !m.HasLocation || geo.HaversineKm(lat, lng, m.Latitude, m.Longitude) <= radiusKm {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

// intersectIDs keeps the ids of b that are also in a; a nil a means "any".
func intersectIDs(a, b []string) []string {
	if a == nil {
		return b
	}
	in := make(map[string]bool, len(a))
	for _, id := range a {
		in[id] = true
	}
	out := []string{}
	for _, id := range b {
		if in[id] {
			out = append(out, id)
		}
	}
	return out
}

func writeSearchParamError(w http.ResponseWriter, err error) {
	if errors.Is(err, errMarketsUnavailable) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

// searchProductsHandler serves the app's product search: one row per catalog
// item (its cheapest offer, with markets_count), filtered, sorted and paged
// in the database, wrapped as {items, total, limit, offset, has_more}.
func searchProductsHandler(productsRepo *repository.ProductsRepo, marketsRepo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		f, err := parseProductSearch(req, marketsRepo)
		if err != nil {
			writeSearchParamError(w, err)
			return
		}
		page, err := productsRepo.Search(req.Context(), f)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to search products"})
			return
		}
		writeJSON(w, http.StatusOK, page)
	}
}

// productCategoriesHandler lists the categories with how many search rows
// each has under the same filters as the search (minus the category).
func productCategoriesHandler(productsRepo *repository.ProductsRepo, marketsRepo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		f, err := parseProductSearch(req, marketsRepo)
		if err != nil {
			writeSearchParamError(w, err)
			return
		}
		counts, err := productsRepo.CategoryCounts(req.Context(), f)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load categories"})
			return
		}
		writeJSON(w, http.StatusOK, counts)
	}
}
