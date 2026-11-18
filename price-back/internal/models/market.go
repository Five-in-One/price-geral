package models

// Market is a supermarket registered in the system.
type Market struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Initials  string  `json:"initials"`
	Latitude  float64 `json:"-"`
	Longitude float64 `json:"-"`
	// HasLocation is false while Latitude/Longitude are only a placeholder
	// (market registered without a usable address); no distance is computed
	// for those. Exposed so the Admin panel can flag markets that still need
	// a valid address.
	HasLocation     bool   `json:"has_location"`
	PrimaryColor    string `json:"primary_color"`
	BackgroundColor string `json:"background_color"`
	// Source is "seed" (demo data), "company" (self-registered via the
	// partner page) or "bot" (created by the admin's on-demand collection).
	Source string `json:"source"`
	// Links are the market's storefront URLs — usually one, but a chain can
	// have more than one (different domains, a site that moved). Whichever
	// one is submitted, matching to reuse this market happens by name, not
	// by URL. The Admin panel's "Recarregar" button re-runs the bot's
	// collection against every link here.
	Links []string `json:"links"`
	// Address fields are optional, set by the Admin panel's "coleta sob
	// demanda" form (street/number/neighborhood/city/state auto-filled
	// from CEP via ViaCEP, client-side). Empty for markets registered
	// before this existed.
	CEP          string `json:"cep,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Complement   string `json:"complement,omitempty"`
}

// MarketWithDistance is a Market annotated with its distance from a
// reference point, as returned by the markets listing endpoint. DistanceKm
// is nil (JSON null) when the market has no real location yet.
type MarketWithDistance struct {
	Market
	DistanceKm *float64 `json:"distance_km"`
}
