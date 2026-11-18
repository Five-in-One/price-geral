package repository

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

var ErrUnknownProduct = errors.New("unknown product_id")

type ComparisonsRepo struct {
	pool *pgxpool.Pool
}

func NewComparisonsRepo(pool *pgxpool.Pool) *ComparisonsRepo {
	return &ComparisonsRepo{pool: pool}
}

// Compare computes, for every market, the total cost of the given shopping
// list, ranked ascending by total cost with each result's difference from
// the cheapest market.
func (r *ComparisonsRepo) Compare(ctx context.Context, items []models.CompareItem) ([]models.ComparisonResult, error) {
	if len(items) > 0 {
		if err := r.ensureProductsExist(ctx, items); err != nil {
			return nil, err
		}
	}

	markets, err := NewMarketsRepo(r.pool).ListAll(ctx)
	if err != nil {
		return nil, err
	}

	priceMatrix, err := r.loadPriceMatrix(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]models.ComparisonResult, 0, len(markets))
	for _, m := range markets {
		var total float64
		var matched int
		var lines []models.ItemPrice

		for _, item := range items {
			price, ok := priceMatrix[m.ID][item.ProductID]
			if !ok {
				continue
			}
			total += price * float64(item.Quantity)
			matched += item.Quantity
			lines = append(lines, models.ItemPrice{ProductID: item.ProductID, Price: price})
		}

		if lines == nil {
			lines = []models.ItemPrice{}
		}

		results = append(results, models.ComparisonResult{
			Market:            m,
			TotalCost:         round2(total),
			MatchedItemsCount: matched,
			Items:             lines,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].TotalCost < results[j].TotalCost
	})

	if len(results) > 0 {
		lowest := results[0].TotalCost
		for i := range results {
			results[i].Difference = round2(results[i].TotalCost - lowest)
		}
	}

	return results, nil
}

func (r *ComparisonsRepo) ensureProductsExist(ctx context.Context, items []models.CompareItem) error {
	ids := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, it := range items {
		if !seen[it.ProductID] {
			seen[it.ProductID] = true
			ids = append(ids, it.ProductID)
		}
	}

	rows, err := r.pool.Query(ctx, `SELECT id FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	found := make(map[string]bool, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		found[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range ids {
		if !found[id] {
			return ErrUnknownProduct
		}
	}

	return nil
}

// loadPriceMatrix returns market_id -> product_id -> price for every priced
// product/market pair.
func (r *ComparisonsRepo) loadPriceMatrix(ctx context.Context) (map[string]map[string]float64, error) {
	rows, err := r.pool.Query(ctx, `SELECT market_id, product_id, price FROM market_products`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matrix := make(map[string]map[string]float64)
	for rows.Next() {
		var marketID, productID string
		var price float64
		if err := rows.Scan(&marketID, &productID, &price); err != nil {
			return nil, err
		}
		if matrix[marketID] == nil {
			matrix[marketID] = make(map[string]float64)
		}
		matrix[marketID][productID] = price
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matrix, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
