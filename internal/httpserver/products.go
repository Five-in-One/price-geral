package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
	"priceback/internal/repository"
)

func productsRouter(pool *pgxpool.Pool) chi.Router {
	productsRepo := repository.NewProductsRepo(pool)
	marketsRepo := repository.NewMarketsRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listProductsHandler(productsRepo))
	r.Post("/", createProductHandler(productsRepo, marketsRepo))
	r.Patch("/{id}", updateProductHandler(productsRepo))
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
}

func createProductHandler(productsRepo *repository.ProductsRepo, marketsRepo *repository.MarketsRepo) http.HandlerFunc {
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
		})
		if err != nil {
			if errors.Is(err, repository.ErrMarketNotFound) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown market_id"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create product"})
			return
		}

		writeJSON(w, http.StatusCreated, product)
	}
}

type updateProductRequest struct {
	Name     *string  `json:"name"`
	Unit     *string  `json:"unit"`
	Category *string  `json:"category"`
	Price    *float64 `json:"price"`
}

func updateProductHandler(repo *repository.ProductsRepo) http.HandlerFunc {
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

		product, err := repo.Update(req.Context(), id, repository.UpdateProductInput{
			Name:     body.Name,
			Unit:     body.Unit,
			Category: body.Category,
			Price:    body.Price,
		})
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update product"})
			return
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
