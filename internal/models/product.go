package models

import (
	"math"
	"time"
)

// Product is an item that can be searched for and priced across markets.
type Product struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Unit              string  `json:"unit"`
	Category          string  `json:"category"`
	DefaultMarketName string  `json:"default_market_name"`
	DefaultPrice      float64 `json:"default_price"`
	// Source is "seed" (demo data), "company" (self-registered via the
	// partner page) or "bot" (created by the admin's on-demand collection).
	Source string `json:"source"`

	// Fields the bot extracts from the scraped name (see migration 0008).
	// All optional: products registered elsewhere leave them empty.
	Brand      string `json:"brand,omitempty"`
	GenericKey string `json:"generic_key,omitempty"`
	PackCount  int    `json:"pack_count"`
	// SizeKind is "mass", "volume" or "length"; SizeAmount is the whole pack in
	// grams, millilitres or metres. Both nil/empty when the name has no size.
	SizeKind     string   `json:"size_kind,omitempty"`
	SizeAmount   *float64 `json:"size_amount"`
	SizeLabel    string   `json:"size_label,omitempty"`
	SoldByWeight bool     `json:"sold_by_weight"`

	// UnitPrice is DefaultPrice normalised to R$/kg, R$/L, R$/m or R$/un
	// (UnitPriceUnit), so packs of different sizes can be compared. Nil when
	// the size is unknown. Derived on read, never stored.
	UnitPrice     *float64 `json:"unit_price"`
	UnitPriceUnit string   `json:"unit_price_unit,omitempty"`

	// CollectedAt is when DefaultPrice was observed.
	CollectedAt time.Time `json:"collected_at"`

	// CatalogID is the canonical catalog item this listing is an offer of
	// (see migration 0009); empty until the listing has been matched.
	CatalogID string `json:"catalog_id,omitempty"`
}

// ComputeUnitPrice fills UnitPrice/UnitPriceUnit from DefaultPrice and the
// size fields.
func (p *Product) ComputeUnitPrice() {
	p.UnitPrice, p.UnitPriceUnit = nil, ""

	set := func(value float64, per string) {
		v := math.Round(value*100) / 100
		p.UnitPrice, p.UnitPriceUnit = &v, per
	}

	switch {
	case p.SizeAmount != nil && *p.SizeAmount > 0:
		switch p.SizeKind {
		case "mass":
			set(p.DefaultPrice/(*p.SizeAmount/1000), "kg")
		case "volume":
			set(p.DefaultPrice/(*p.SizeAmount/1000), "L")
		case "length":
			set(p.DefaultPrice/(*p.SizeAmount), "m")
		}
	case p.SoldByWeight:
		set(p.DefaultPrice, "kg")
	case p.PackCount > 1:
		set(p.DefaultPrice/float64(p.PackCount), "un")
	}
}
