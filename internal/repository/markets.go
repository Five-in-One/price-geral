package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

type MarketsRepo struct {
	pool *pgxpool.Pool
}

func NewMarketsRepo(pool *pgxpool.Pool) *MarketsRepo {
	return &MarketsRepo{pool: pool}
}

// marketColumns is the column list (with nullable address fields coalesced
// to "") shared by every SELECT that returns a full Market — except Links,
// which lives in market_links and is attached separately (see attachLinks).
const marketColumns = `
	id, name, initials, latitude, longitude, has_location, primary_color, background_color, source,
	COALESCE(cep, ''), COALESCE(street, ''), COALESCE(number, ''),
	COALESCE(neighborhood, ''), COALESCE(city, ''), COALESCE(state, ''), COALESCE(complement, '')
`

func scanMarket(row interface{ Scan(...any) error }, m *models.Market) error {
	return row.Scan(
		&m.ID, &m.Name, &m.Initials, &m.Latitude, &m.Longitude, &m.HasLocation, &m.PrimaryColor, &m.BackgroundColor, &m.Source,
		&m.CEP, &m.Street, &m.Number, &m.Neighborhood, &m.City, &m.State, &m.Complement,
	)
}

// linksFor returns the storefront URLs registered for a single market, in
// the order they were added.
func (r *MarketsRepo) linksFor(ctx context.Context, marketID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT url FROM market_links WHERE market_id = $1 ORDER BY created_at
	`, marketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []string{}
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		links = append(links, url)
	}
	return links, rows.Err()
}

// attachLinks fills in Links on every market in the slice with a single
// extra query, instead of one round-trip per market.
func (r *MarketsRepo) attachLinks(ctx context.Context, markets []models.Market) error {
	if len(markets) == 0 {
		return nil
	}

	ids := make([]string, len(markets))
	byID := make(map[string]*models.Market, len(markets))
	for i := range markets {
		ids[i] = markets[i].ID
		markets[i].Links = []string{}
		byID[markets[i].ID] = &markets[i]
	}

	rows, err := r.pool.Query(ctx, `
		SELECT market_id, url FROM market_links WHERE market_id = ANY($1) ORDER BY created_at
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var marketID, url string
		if err := rows.Scan(&marketID, &url); err != nil {
			return err
		}
		if m, ok := byID[marketID]; ok {
			m.Links = append(m.Links, url)
		}
	}
	return rows.Err()
}

// ListAll returns every registered market, regardless of location.
func (r *MarketsRepo) ListAll(ctx context.Context) ([]models.Market, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+marketColumns+` FROM markets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []models.Market
	for rows.Next() {
		var m models.Market
		if err := scanMarket(rows, &m); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.attachLinks(ctx, markets); err != nil {
		return nil, err
	}

	return markets, nil
}

// GetByID returns a single market by id, or ErrNotFound.
func (r *MarketsRepo) GetByID(ctx context.Context, id string) (models.Market, error) {
	var m models.Market
	err := scanMarket(r.pool.QueryRow(ctx, `SELECT `+marketColumns+` FROM markets WHERE id = $1`, id), &m)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Market{}, ErrNotFound
		}
		return models.Market{}, err
	}

	links, err := r.linksFor(ctx, id)
	if err != nil {
		return models.Market{}, err
	}
	m.Links = links

	return m, nil
}

// CreateMarketInput describes a market to create (Admin's on-demand
// collection or the Partner Company page).
type CreateMarketInput struct {
	Name      string
	Initials  string
	Latitude  float64
	Longitude float64
	// HasLocation says whether Latitude/Longitude are real coordinates
	// (geocoded from the address, or sent explicitly) rather than a
	// placeholder. A placeholder never overwrites real coordinates already
	// stored for a market resubmitted by name.
	HasLocation     bool
	PrimaryColor    string
	BackgroundColor string
	Source          string // "company" or "bot"
	// Link is a single storefront URL to add to this market's link list
	// (see models.Market.Links). Empty means "don't add a link this time"
	// — e.g. when a product push auto-creates its market with no known URL.
	Link string
	// Address fields are optional (see models.Market).
	CEP          string
	Street       string
	Number       string
	Neighborhood string
	City         string
	State        string
	Complement   string
}

// FindOrCreate inserts a new market, or returns the existing one if a
// market with the same name (case-insensitive) already exists — so
// resubmitting the same market name is idempotent instead of erroring.
// Address fields merge in rather than overwrite blindly: a non-empty field
// on input replaces the stored one, an empty one leaves it untouched. This
// lets the admin register a bare market first and fill in its address (or
// correct it) later just by resubmitting the same name.
func (r *MarketsRepo) FindOrCreate(ctx context.Context, input CreateMarketInput) (models.Market, error) {
	id := generateID(input.Name, "mercado")

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Market{}, err
	}
	defer tx.Rollback(ctx)

	var marketID string
	err = tx.QueryRow(ctx, `
		INSERT INTO markets (
			id, name, initials, latitude, longitude, primary_color, background_color, source,
			cep, street, number, neighborhood, city, state, complement, has_location
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (lower(name)) DO UPDATE SET
			name         = markets.name,
			latitude     = CASE WHEN EXCLUDED.has_location THEN EXCLUDED.latitude ELSE markets.latitude END,
			longitude    = CASE WHEN EXCLUDED.has_location THEN EXCLUDED.longitude ELSE markets.longitude END,
			has_location = markets.has_location OR EXCLUDED.has_location,
			cep          = COALESCE(NULLIF(EXCLUDED.cep, ''), markets.cep),
			street       = COALESCE(NULLIF(EXCLUDED.street, ''), markets.street),
			number       = COALESCE(NULLIF(EXCLUDED.number, ''), markets.number),
			neighborhood = COALESCE(NULLIF(EXCLUDED.neighborhood, ''), markets.neighborhood),
			city         = COALESCE(NULLIF(EXCLUDED.city, ''), markets.city),
			state        = COALESCE(NULLIF(EXCLUDED.state, ''), markets.state),
			complement   = COALESCE(NULLIF(EXCLUDED.complement, ''), markets.complement)
		RETURNING id
	`, id, input.Name, input.Initials, input.Latitude, input.Longitude, input.PrimaryColor, input.BackgroundColor, input.Source,
		input.CEP, input.Street, input.Number, input.Neighborhood, input.City, input.State, input.Complement,
		input.HasLocation,
	).Scan(&marketID)
	if err != nil {
		return models.Market{}, err
	}

	if link := strings.TrimSpace(input.Link); link != "" {
		linkID := generateID(link, "link")
		if _, err := tx.Exec(ctx, `
			INSERT INTO market_links (id, market_id, url)
			VALUES ($1, $2, $3)
			ON CONFLICT (market_id, lower(url)) DO NOTHING
		`, linkID, marketID, link); err != nil {
			return models.Market{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Market{}, err
	}

	return r.GetByID(ctx, marketID)
}

// UpdateMarketInput describes fields to change on an existing market.
// Pointer fields distinguish "not provided" from a zero value, so a PATCH
// can touch just one field without clobbering the others.
type UpdateMarketInput struct {
	Name            *string
	Initials        *string
	PrimaryColor    *string
	BackgroundColor *string
	// Address fields; nil leaves the stored value, "" clears it.
	CEP          *string
	Street       *string
	Number       *string
	Neighborhood *string
	City         *string
	State        *string
	Complement   *string
	// Location, when set, overwrites the stored coordinates — used after the
	// address changed and was geocoded again (or could not be).
	Location *LocationUpdate
}

// LocationUpdate is a new set of coordinates for a market. With
// HasLocation false the market goes back to "no known location".
type LocationUpdate struct {
	Latitude    float64
	Longitude   float64
	HasLocation bool
}

// Update applies the given fields to a market, returning ErrNotFound if no
// market has id.
func (r *MarketsRepo) Update(ctx context.Context, id string, input UpdateMarketInput) (models.Market, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return models.Market{}, err
	}

	name, initials := current.Name, current.Initials
	primary, background := current.PrimaryColor, current.BackgroundColor
	cep, street, number := current.CEP, current.Street, current.Number
	neighborhood, city, state, complement := current.Neighborhood, current.City, current.State, current.Complement
	lat, lng, hasLocation := current.Latitude, current.Longitude, current.HasLocation
	if input.Name != nil {
		name = *input.Name
	}
	if input.Initials != nil {
		initials = *input.Initials
	}
	if input.PrimaryColor != nil {
		primary = *input.PrimaryColor
	}
	if input.BackgroundColor != nil {
		background = *input.BackgroundColor
	}
	for _, f := range []struct {
		in  *string
		out *string
	}{
		{input.CEP, &cep}, {input.Street, &street}, {input.Number, &number},
		{input.Neighborhood, &neighborhood}, {input.City, &city},
		{input.State, &state}, {input.Complement, &complement},
	} {
		if f.in != nil {
			*f.out = *f.in
		}
	}
	if input.Location != nil {
		lat, lng, hasLocation = input.Location.Latitude, input.Location.Longitude, input.Location.HasLocation
	}

	if _, err := r.pool.Exec(ctx, `
		UPDATE markets SET
			name = $1, initials = $2, primary_color = $3, background_color = $4,
			cep = $5, street = $6, number = $7, neighborhood = $8, city = $9, state = $10, complement = $11,
			latitude = $12, longitude = $13, has_location = $14
		WHERE id = $15
	`, name, initials, primary, background,
		cep, street, number, neighborhood, city, state, complement,
		lat, lng, hasLocation, id); err != nil {
		return models.Market{}, err
	}

	return r.GetByID(ctx, id)
}

// AddLink appends a storefront URL to a market's link list (a no-op if the
// same URL, case-insensitively, is already there), returning the updated
// market. Returns ErrNotFound if no market has id.
func (r *MarketsRepo) AddLink(ctx context.Context, marketID, url string) (models.Market, error) {
	if _, err := r.GetByID(ctx, marketID); err != nil {
		return models.Market{}, err
	}

	linkID := generateID(url, "link")
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO market_links (id, market_id, url)
		VALUES ($1, $2, $3)
		ON CONFLICT (market_id, lower(url)) DO NOTHING
	`, linkID, marketID, url); err != nil {
		return models.Market{}, err
	}

	return r.GetByID(ctx, marketID)
}

// RemoveLink removes a storefront URL (matched case-insensitively) from a
// market's link list. A market with no matching link is left unchanged —
// only a missing market itself is an error (ErrNotFound).
func (r *MarketsRepo) RemoveLink(ctx context.Context, marketID, url string) (models.Market, error) {
	if _, err := r.GetByID(ctx, marketID); err != nil {
		return models.Market{}, err
	}

	if _, err := r.pool.Exec(ctx, `
		DELETE FROM market_links WHERE market_id = $1 AND lower(url) = lower($2)
	`, marketID, url); err != nil {
		return models.Market{}, err
	}

	return r.GetByID(ctx, marketID)
}

// Delete removes a market. Products whose default market it is are removed
// first (cascading their market_products rows too) since the schema
// otherwise blocks deleting a market still referenced by a product.
// Returns ErrNotFound if no market has id.
func (r *MarketsRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM products WHERE default_market_id = $1`, id); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM markets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return tx.Commit(ctx)
}
