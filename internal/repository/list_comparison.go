package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"priceback/internal/geo"
	"priceback/internal/models"
)

var (
	ErrEmptyList = errors.New("the list has no items")
	// ErrProductNotInCatalog: the product has no catalog item yet (it was
	// registered without the fields the matching needs), so it can't be
	// priced at other markets.
	ErrProductNotInCatalog = errors.New("product is not in the catalog")
)

// ProductNotInCatalogError says which listing couldn't be resolved.
type ProductNotInCatalogError struct{ ProductID string }

func (e ProductNotInCatalogError) Error() string {
	return fmt.Sprintf("%v: %s", ErrProductNotInCatalog, e.ProductID)
}

func (e ProductNotInCatalogError) Unwrap() error { return ErrProductNotInCatalog }

// wanted is one resolved item of the list.
type wanted struct {
	item     catalogRow
	quantity int
}

type catalogRow struct {
	ID           string
	Name         string
	Brand        string
	GenericKey   string
	SizeKind     string
	SizeAmount   float64
	SizeLabel    string
	PackCount    int
	SoldByWeight bool
}

// offer is a market's cheapest listing of one catalog item.
type offer struct {
	catalogID   string
	productID   string
	productName string
	brand       string
	sizeLabel   string
	price       float64
	collectedAt time.Time
	unitPrice   *float64
	unitPer     string
}

// CompareList prices a shopping list at every market.
//
// Each list item is a catalog item, so the same product is found at every
// market that sells it. A market's line is its cheapest listing of the item;
// with AllowEquivalents, a gap can be filled by another brand of the same
// product — never when the requested brand is available, and only for
// products where switching brand is normal (see IsSubstitutable).
func (r *ComparisonsRepo) CompareList(ctx context.Context, req models.ListCompareRequest) (models.ListComparison, error) {
	if len(req.Items) == 0 {
		return models.ListComparison{}, ErrEmptyList
	}

	list, err := r.resolveList(ctx, req.Items)
	if err != nil {
		return models.ListComparison{}, err
	}

	// candidate catalog items: what was asked for, plus substitutes
	equivalents := map[string][]string{} // asked-for id -> equivalent ids
	candidates := map[string]bool{}
	for _, w := range list {
		candidates[w.item.ID] = true
		if !req.AllowEquivalents || !IsSubstitutable(w.item.GenericKey) {
			continue
		}
		ids, err := r.equivalentsOf(ctx, w.item)
		if err != nil {
			return models.ListComparison{}, err
		}
		equivalents[w.item.ID] = ids
		for _, id := range ids {
			candidates[id] = true
		}
	}
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	offers, err := r.loadOffers(ctx, ids)
	if err != nil {
		return models.ListComparison{}, err
	}

	markets, err := NewMarketsRepo(r.pool).ListAll(ctx)
	if err != nil {
		return models.ListComparison{}, err
	}

	comparisons := make([]models.MarketComparison, 0, len(markets))
	for _, m := range markets {
		var distance *float64
		if req.Lat != nil && req.Lng != nil && m.HasLocation {
			d := round2(geo.HaversineKm(*req.Lat, *req.Lng, m.Latitude, m.Longitude))
			if req.RadiusKm != nil && d > *req.RadiusKm {
				continue
			}
			distance = &d
		}

		mc := models.MarketComparison{
			Market: m, DistanceKm: distance, TotalItems: len(list),
			Lines: []models.ListLine{}, Missing: []models.MissingItem{},
		}
		for _, w := range list {
			o, substitute, ok := pickOffer(offers, m.ID, w.item.ID, equivalents[w.item.ID])
			if !ok {
				mc.Missing = append(mc.Missing, models.MissingItem{CatalogID: w.item.ID, Name: w.item.Name, Quantity: w.quantity})
				continue
			}
			mc.Lines = append(mc.Lines, lineFor(w, o, substitute))
			mc.TotalCost += o.price * float64(w.quantity)
			mc.FoundItems++
			if substitute {
				mc.SubstitutesCount++
			}
		}
		mc.TotalCost = round2(mc.TotalCost)
		mc.IsComplete = mc.FoundItems == mc.TotalItems
		comparisons = append(comparisons, mc)
	}

	rankMarkets(comparisons)
	summaries := markCheapest(list, comparisons)

	out := models.ListComparison{
		AllowEquivalents: req.AllowEquivalents,
		Items:            summaries,
		Markets:          comparisons,
	}
	fillSavings(&out)
	out.BestCombination = bestCombination(list, out.Markets)
	return out, nil
}

// resolveList turns the request lines into catalog items, merging repeated
// items by summing their quantities.
func (r *ComparisonsRepo) resolveList(ctx context.Context, items []models.ListCompareItem) ([]wanted, error) {
	// product_id -> catalog_id
	var productIDs []string
	for _, it := range items {
		if it.CatalogID == "" && it.ProductID != "" {
			productIDs = append(productIDs, it.ProductID)
		}
	}
	catalogOf, err := r.catalogIDsForProducts(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	order := []string{}
	quantity := map[string]int{}
	for _, it := range items {
		id := it.CatalogID
		if id == "" {
			cid, known := catalogOf[it.ProductID]
			if !known {
				return nil, ErrUnknownProduct
			}
			if cid == "" {
				return nil, ProductNotInCatalogError{ProductID: it.ProductID}
			}
			id = cid
		}
		if _, seen := quantity[id]; !seen {
			order = append(order, id)
		}
		quantity[id] += it.Quantity
	}

	rows, err := r.loadCatalogRows(ctx, order)
	if err != nil {
		return nil, err
	}
	list := make([]wanted, 0, len(order))
	for _, id := range order {
		row, ok := rows[id]
		if !ok {
			return nil, ErrCatalogItemNotFound
		}
		list = append(list, wanted{item: row, quantity: quantity[id]})
	}
	return list, nil
}

// catalogIDsForProducts maps product id -> catalog id ("" when unlinked);
// ids that don't exist are absent from the result.
func (r *ComparisonsRepo) catalogIDsForProducts(ctx context.Context, productIDs []string) (map[string]string, error) {
	out := map[string]string{}
	if len(productIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, COALESCE(catalog_id, '') FROM products WHERE id = ANY($1)`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, catalogID string
		if err := rows.Scan(&id, &catalogID); err != nil {
			return nil, err
		}
		out[id] = catalogID
	}
	return out, rows.Err()
}

func (r *ComparisonsRepo) loadCatalogRows(ctx context.Context, ids []string) (map[string]catalogRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, brand, generic_key, size_kind, size_amount::float8, size_label, pack_count, sold_by_weight
		FROM catalog_items WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]catalogRow{}
	for rows.Next() {
		var c catalogRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Brand, &c.GenericKey, &c.SizeKind, &c.SizeAmount, &c.SizeLabel, &c.PackCount, &c.SoldByWeight); err != nil {
			return nil, err
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

// equivalentsOf returns the other catalog items that are the same product in
// another brand: identical generic key, package size and pack count.
func (r *ComparisonsRepo) equivalentsOf(ctx context.Context, c catalogRow) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM catalog_items
		WHERE id <> $1 AND generic_key = $2 AND size_kind = $3 AND size_amount = $4
		  AND pack_count = $5 AND sold_by_weight = $6 AND brand <> $7
	`, c.ID, c.GenericKey, c.SizeKind, c.SizeAmount, c.PackCount, c.SoldByWeight, c.Brand)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadOffers returns catalog id -> market id -> that market's cheapest
// listing of the item.
func (r *ComparisonsRepo) loadOffers(ctx context.Context, catalogIDs []string) (map[string]map[string]offer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.catalog_id, p.default_market_id, p.id, p.name, COALESCE(p.brand, ''), COALESCE(p.size_label, ''),
		       mp.price, mp.collected_at,
		       COALESCE(p.size_kind, ''), p.size_amount, p.pack_count, p.sold_by_weight
		FROM products p
		JOIN market_products mp ON mp.market_id = p.default_market_id AND mp.product_id = p.id
		WHERE p.catalog_id = ANY($1)
		ORDER BY mp.price, p.name
	`, catalogIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]map[string]offer{}
	for rows.Next() {
		var marketID string
		var o offer
		tmp := models.Product{}
		if err := rows.Scan(
			&o.catalogID, &marketID, &o.productID, &o.productName, &o.brand, &o.sizeLabel, &o.price, &o.collectedAt,
			&tmp.SizeKind, &tmp.SizeAmount, &tmp.PackCount, &tmp.SoldByWeight,
		); err != nil {
			return nil, err
		}
		tmp.DefaultPrice = o.price
		tmp.ComputeUnitPrice()
		o.unitPrice, o.unitPer = tmp.UnitPrice, tmp.UnitPriceUnit

		if out[o.catalogID] == nil {
			out[o.catalogID] = map[string]offer{}
		}
		// rows come cheapest first, so the first one per market is the one to keep
		if _, taken := out[o.catalogID][marketID]; !taken {
			out[o.catalogID][marketID] = o
		}
	}
	return out, rows.Err()
}

// pickOffer chooses the market's listing for an item: the requested product
// if it sells it, otherwise the cheapest equivalent.
func pickOffer(offers map[string]map[string]offer, marketID, itemID string, equivalents []string) (o offer, substitute, ok bool) {
	if exact, found := offers[itemID][marketID]; found {
		return exact, false, true
	}
	for _, id := range equivalents {
		if eq, found := offers[id][marketID]; found && (!ok || eq.price < o.price) {
			o, substitute, ok = eq, true, true
		}
	}
	return o, substitute, ok
}

func lineFor(w wanted, o offer, substitute bool) models.ListLine {
	return models.ListLine{
		CatalogID: w.item.ID, Name: w.item.Name, Quantity: w.quantity,
		ProductID: o.productID, ProductName: o.productName, Brand: o.brand, SizeLabel: o.sizeLabel,
		Price: o.price, LineTotal: round2(o.price * float64(w.quantity)),
		UnitPrice: o.unitPrice, UnitPriceUnit: o.unitPer, CollectedAt: o.collectedAt,
		IsSubstitute: substitute,
	}
}

// rankMarkets orders markets: those with the whole list first (cheapest, then
// nearest), then the incomplete ones (most items first, then cheapest).
func rankMarkets(ms []models.MarketComparison) {
	nearer := func(a, b models.MarketComparison) (bool, bool) {
		switch {
		case a.DistanceKm != nil && b.DistanceKm != nil && *a.DistanceKm != *b.DistanceKm:
			return *a.DistanceKm < *b.DistanceKm, true
		case a.DistanceKm != nil && b.DistanceKm == nil:
			return true, true
		case a.DistanceKm == nil && b.DistanceKm != nil:
			return false, true
		}
		return false, false
	}
	sort.SliceStable(ms, func(i, j int) bool {
		a, b := ms[i], ms[j]
		if a.IsComplete != b.IsComplete {
			return a.IsComplete
		}
		if !a.IsComplete && a.FoundItems != b.FoundItems {
			return a.FoundItems > b.FoundItems
		}
		if a.TotalCost != b.TotalCost {
			return a.TotalCost < b.TotalCost
		}
		if less, decided := nearer(a, b); decided {
			return less
		}
		return a.Market.Name < b.Market.Name
	})

	// difference is measured from the cheapest market that has the whole list
	if len(ms) > 0 && ms[0].IsComplete {
		best := ms[0].TotalCost
		for i := range ms {
			if ms[i].IsComplete {
				ms[i].Difference = round2(ms[i].TotalCost - best)
			}
		}
	}
}

// markCheapest flags, for every item, the line(s) no other market beats, and
// summarises each item across markets.
func markCheapest(list []wanted, ms []models.MarketComparison) []models.ListItemSummary {
	summaries := make([]models.ListItemSummary, 0, len(list))
	for _, w := range list {
		s := models.ListItemSummary{
			CatalogID: w.item.ID, Name: w.item.Name, Brand: w.item.Brand,
			SizeLabel: w.item.SizeLabel, Quantity: w.quantity,
		}
		for i := range ms {
			for _, l := range ms[i].Lines {
				if l.CatalogID == w.item.ID && (s.CheapestPrice == nil || l.Price < *s.CheapestPrice) {
					price := l.Price
					s.CheapestPrice, s.CheapestMarketID, s.Available = &price, ms[i].Market.ID, true
				}
			}
		}
		if s.Available {
			for i := range ms {
				for j := range ms[i].Lines {
					l := &ms[i].Lines[j]
					if l.CatalogID == w.item.ID && l.Price == *s.CheapestPrice {
						l.IsCheapest = true
					}
				}
			}
		}
		summaries = append(summaries, s)
	}
	return summaries
}

// fillSavings sets the best single market and the spread between the
// cheapest and most expensive market that have the whole list.
func fillSavings(out *models.ListComparison) {
	var complete []models.MarketComparison
	for _, m := range out.Markets {
		if m.IsComplete {
			complete = append(complete, m)
		}
	}
	if len(complete) == 0 {
		return
	}
	out.BestSingleMarketID = complete[0].Market.ID
	if len(complete) < 2 {
		return
	}
	best, worst := complete[0].TotalCost, complete[len(complete)-1].TotalCost
	for _, m := range complete {
		worst = math.Max(worst, m.TotalCost)
	}
	out.MaxSavings = round2(worst - best)
	if worst > 0 {
		out.MaxSavingsPercent = round2((worst - best) / worst * 100)
	}
}

// bestCombination buys every available item where it is cheapest. On ties it
// prefers a market already on the route, then the nearest one, so it doesn't
// add a stop for nothing.
func bestCombination(list []wanted, ranked []models.MarketComparison) *models.BestCombination {
	type pick struct {
		market int
		line   models.ListLine
	}
	used := map[int]bool{}
	var picks []pick

	for _, w := range list {
		best := -1
		var bestLine models.ListLine
		for mi := range ranked {
			for _, l := range ranked[mi].Lines {
				if l.CatalogID != w.item.ID {
					continue
				}
				switch {
				case best == -1 || l.Price < bestLine.Price-0.005:
					best, bestLine = mi, l
				case math.Abs(l.Price-bestLine.Price) < 0.005 && prefers(ranked, used, mi, best):
					best, bestLine = mi, l
				}
			}
		}
		if best >= 0 {
			used[best] = true
			picks = append(picks, pick{market: best, line: bestLine})
		}
	}
	if len(picks) == 0 {
		return nil
	}

	stops := map[int]*models.Stop{}
	var order []int
	total := 0.0
	for _, p := range picks {
		s, ok := stops[p.market]
		if !ok {
			s = &models.Stop{Market: ranked[p.market].Market, DistanceKm: ranked[p.market].DistanceKm, Lines: []models.ListLine{}}
			stops[p.market] = s
			order = append(order, p.market)
		}
		s.Lines = append(s.Lines, p.line)
		s.Subtotal = round2(s.Subtotal + p.line.LineTotal)
		total += p.line.LineTotal
	}

	combo := &models.BestCombination{TotalCost: round2(total), Stops: make([]models.Stop, 0, len(order))}
	for _, mi := range order {
		combo.Stops = append(combo.Stops, *stops[mi])
	}
	sort.SliceStable(combo.Stops, func(i, j int) bool { return combo.Stops[i].Subtotal > combo.Stops[j].Subtotal })

	if len(ranked) > 0 && ranked[0].IsComplete {
		saved := round2(ranked[0].TotalCost - combo.TotalCost)
		if saved < 0 {
			saved = 0
		}
		combo.SavingsVsBestSingle = &saved
	}
	return combo
}

// prefers reports whether market candidate should win a price tie against the
// current choice: one already on the route, else the nearer one.
func prefers(ms []models.MarketComparison, used map[int]bool, candidate, current int) bool {
	if used[candidate] != used[current] {
		return used[candidate]
	}
	a, b := ms[candidate].DistanceKm, ms[current].DistanceKm
	if a != nil && b != nil {
		return *a < *b
	}
	return a != nil && b == nil
}
