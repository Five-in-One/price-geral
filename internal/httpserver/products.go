package httpserver

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
	"priceback/internal/repository"
)

func productsRouter(pool *pgxpool.Pool) chi.Router {
	productsRepo := repository.NewProductsRepo(pool)
	marketsRepo := repository.NewMarketsRepo(pool)
	catalogRepo := repository.NewCatalogRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listProductsHandler(productsRepo))
	r.Post("/", createProductHandler(productsRepo, marketsRepo, catalogRepo))
	r.Patch("/{id}", updateProductHandler(productsRepo, catalogRepo))
	r.Delete("/{id}", deleteProductHandler(productsRepo))
	return r
}

func listProductsHandler(repo *repository.ProductsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()

		filter := repository.ProductFilter{
			Category:       q.Get("category"),
			Search:         q.Get("search"),
			Source:         q.Get("source"),
			MarketID:       q.Get("market_id"),
			SortByPriceAsc: q.Get("sort") == "price_asc",
		}

		products, err := repo.List(req.Context(), filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load products"})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		writeJSON(w, http.StatusOK, products)
	}
}

type createProductRequest struct {
	Name       string  `json:"name"`
	Unit       string  `json:"unit"`
	Category   string  `json:"category"`
	Price      float64 `json:"price"`
	MarketID   string  `json:"market_id"`
	MarketName string  `json:"market_name"`
	Source     string  `json:"source"`

	// Optional fields the bot extracts from the scraped name.
	Brand        string     `json:"brand"`
	GenericKey   string     `json:"generic_key"`
	PackCount    int        `json:"pack_count"`
	SizeKind     string     `json:"size_kind"`
	SizeAmount   *float64   `json:"size_amount"`
	SizeLabel    string     `json:"size_label"`
	SoldByWeight bool       `json:"sold_by_weight"`
	CollectedAt  *time.Time `json:"collected_at"`
}

// validSizeKinds are the measurement families a product size can be in
// (grams, millilitres, metres).
var validSizeKinds = map[string]bool{"mass": true, "volume": true, "length": true}

// validateSize checks that size_kind and size_amount come together and make
// sense. It returns the error message to send back, or "" when fine.
func validateSize(kind string, amount *float64) string {
	if kind == "" && amount == nil {
		return ""
	}
	if !validSizeKinds[kind] {
		return "size_kind must be 'mass', 'volume' or 'length'"
	}
	if amount == nil || *amount <= 0 {
		return "size_amount must be greater than zero when size_kind is set"
	}
	return ""
}

// linkToCatalog attaches a listing to its canonical catalog item right after
// it is saved. Best-effort: a matching problem must never fail (or roll back)
// the price registration itself — `POST /api/catalog/match` can catch up.
func linkToCatalog(req *http.Request, catalogRepo *repository.CatalogRepo, productsRepo *repository.ProductsRepo, product models.Product) models.Product {
	if _, err := catalogRepo.LinkProduct(req.Context(), product.ID); err != nil {
		log.Printf("linking product %q to the catalog failed: %v", product.ID, err)
		return product
	}
	if linked, err := productsRepo.GetByID(req.Context(), product.ID); err == nil {
		return linked
	}
	return product
}

func createProductHandler(productsRepo *repository.ProductsRepo, marketsRepo *repository.MarketsRepo, catalogRepo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body createProductRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		name := strings.TrimSpace(body.Name)
		unit := strings.TrimSpace(body.Unit)
		category := strings.TrimSpace(body.Category)
		marketName := strings.TrimSpace(body.MarketName)
		marketID := strings.TrimSpace(body.MarketID)

		if name == "" || unit == "" || category == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, unit and category are required"})
			return
		}
		if body.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price must be greater than zero"})
			return
		}
		if marketID == "" && marketName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "market_id or market_name is required"})
			return
		}

		source := body.Source
		if source == "" {
			source = "company"
		}
		if source != "company" && source != "bot" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source must be 'company' or 'bot'"})
			return
		}

		body.SizeKind = strings.TrimSpace(body.SizeKind)
		if msg := validateSize(body.SizeKind, body.SizeAmount); msg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}
		if body.PackCount < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pack_count cannot be negative"})
			return
		}
		// A price cannot have been observed in the future (allow some clock skew).
		if body.CollectedAt != nil && body.CollectedAt.After(time.Now().Add(5*time.Minute)) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "collected_at cannot be in the future"})
			return
		}

		if marketID == "" {
			primary, background := pickPalette(marketName)
			market, err := marketsRepo.FindOrCreate(req.Context(), repository.CreateMarketInput{
				Name:            marketName,
				Initials:        deriveInitials(marketName),
				Latitude:        defaultUserLat,
				Longitude:       defaultUserLng,
				PrimaryColor:    primary,
				BackgroundColor: background,
				Source:          source,
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to resolve market"})
				return
			}
			marketID = market.ID
		}

		product, err := productsRepo.UpsertForMarket(req.Context(), repository.CreateProductInput{
			Name:     name,
			Unit:     unit,
			Category: category,
			MarketID: marketID,
			Price:    body.Price,
			Source:   source,

			HasParsed: body.Brand != "" || body.GenericKey != "" || body.PackCount > 0 ||
				body.SizeKind != "" || body.SizeLabel != "" || body.SoldByWeight,
			Brand:        strings.TrimSpace(body.Brand),
			GenericKey:   strings.TrimSpace(body.GenericKey),
			PackCount:    body.PackCount,
			SizeKind:     body.SizeKind,
			SizeAmount:   body.SizeAmount,
			SizeLabel:    strings.TrimSpace(body.SizeLabel),
			SoldByWeight: body.SoldByWeight,
			CollectedAt:  body.CollectedAt,
		})
		if err != nil {
			if errors.Is(err, repository.ErrMarketNotFound) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown market_id"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create product"})
			return
		}

		writeJSON(w, http.StatusCreated, linkToCatalog(req, catalogRepo, productsRepo, product))
	}
}

type updateProductRequest struct {
	Name     *string  `json:"name"`
	Unit     *string  `json:"unit"`
	Category *string  `json:"category"`
	Price    *float64 `json:"price"`

	Brand        *string  `json:"brand"`
	GenericKey   *string  `json:"generic_key"`
	PackCount    *int     `json:"pack_count"`
	SizeKind     *string  `json:"size_kind"`
	SizeAmount   *float64 `json:"size_amount"`
	SizeLabel    *string  `json:"size_label"`
	SoldByWeight *bool    `json:"sold_by_weight"`
}

func updateProductHandler(repo *repository.ProductsRepo, catalogRepo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")

		var body updateProductRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		if body.Name != nil {
			trimmed := strings.TrimSpace(*body.Name)
			if trimmed == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name cannot be empty"})
				return
			}
			body.Name = &trimmed
		}
		if body.Unit != nil {
			trimmed := strings.TrimSpace(*body.Unit)
			if trimmed == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unit cannot be empty"})
				return
			}
			body.Unit = &trimmed
		}
		if body.Category != nil {
			trimmed := strings.TrimSpace(*body.Category)
			if trimmed == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category cannot be empty"})
				return
			}
			body.Category = &trimmed
		}
		if body.Price != nil && *body.Price <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price must be greater than zero"})
			return
		}
		if body.PackCount != nil && *body.PackCount < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pack_count must be at least 1"})
			return
		}
		if body.SizeKind != nil {
			trimmed := strings.TrimSpace(*body.SizeKind)
			body.SizeKind = &trimmed
			if trimmed == "" {
				body.SizeAmount = nil // clears the size
			} else if msg := validateSize(trimmed, body.SizeAmount); msg != "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
				return
			}
		} else if body.SizeAmount != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "size_amount needs size_kind"})
			return
		}

		product, err := repo.Update(req.Context(), id, repository.UpdateProductInput{
			Name:     body.Name,
			Unit:     body.Unit,
			Category: body.Category,
			Price:    body.Price,

			Brand:        body.Brand,
			GenericKey:   body.GenericKey,
			PackCount:    body.PackCount,
			SizeKind:     body.SizeKind,
			SizeAmount:   body.SizeAmount,
			SizeLabel:    body.SizeLabel,
			SoldByWeight: body.SoldByWeight,
		})
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update product"})
			return
		}

		// New parsed fields can change which catalog item the listing belongs to.
		if body.Brand != nil || body.GenericKey != nil || body.PackCount != nil ||
			body.SizeKind != nil || body.SoldByWeight != nil {
			product = linkToCatalog(req, catalogRepo, repo, product)
		}

		writeJSON(w, http.StatusOK, product)
	}
}

func deleteProductHandler(repo *repository.ProductsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")

		if err := repo.Delete(req.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete product"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
