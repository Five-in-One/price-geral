package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
	"priceback/internal/repository"
)

func productsRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewProductsRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listProductsHandler(repo))
	return r
}

func listProductsHandler(repo *repository.ProductsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()

		filter := repository.ProductFilter{
			Category:       q.Get("category"),
			Search:         q.Get("search"),
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
