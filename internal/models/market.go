package models

// Market is a supermarket registered in the system.
type Market struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Initials        string  `json:"initials"`
	Latitude        float64 `json:"-"`
	Longitude       float64 `json:"-"`
	PrimaryColor    string  `json:"primary_color"`
	BackgroundColor string  `json:"background_color"`
}

// MarketWithDistance is a Market annotated with its distance from a
// reference point, as returned by the markets listing endpoint.
type MarketWithDistance struct {
	Market
	DistanceKm float64 `json:"distance_km"`
}
