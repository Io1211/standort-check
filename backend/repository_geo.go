package backend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SaveGeo stores the result of a geocoding attempt (also failures, so they
// are visible and can be retried).
func (r *Repository) SaveGeo(ctx context.Context, id string, g GeoResult) error {
	var lat, lon, conf *float64
	if g.Status == GeoStatusOK {
		lat, lon, conf = &g.Lat, &g.Lon, &g.Confidence
	}
	_, err := r.db.Exec(ctx, `
		UPDATE leads SET
			geo_status = $2, geo_lat = $3, geo_lon = $4,
			geo_municipality = $5, geo_county = $6, geo_state = $7, geo_postcode = $8,
			geo_formatted = $9, geo_result_type = $10, geo_confidence = $11,
			geo_checked_at = now()
		WHERE id = $1`,
		id, g.Status, lat, lon,
		nullIfEmpty(g.Municipality), nullIfEmpty(g.County), nullIfEmpty(g.State), nullIfEmpty(g.Postcode),
		nullIfEmpty(g.Formatted), nullIfEmpty(g.ResultType), conf,
	)
	if err != nil {
		return fmt.Errorf("save geo: %w", err)
	}
	return nil
}

type geoTarget struct {
	ID      string
	Address Address
}

// LeadsNeedingGeo returns up to limit leads that were never geocoded or whose
// last attempt failed technically (not "not found" – retrying won't help).
func (r *Repository) LeadsNeedingGeo(ctx context.Context, limit int) ([]geoTarget, int, error) {
	var remaining int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM leads WHERE geo_status IS NULL OR geo_status = 'error'`).Scan(&remaining); err != nil {
		return nil, 0, fmt.Errorf("count missing geo: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, street, COALESCE(house_number, ''), postal_code, city
		FROM leads
		WHERE geo_status IS NULL OR geo_status = 'error'
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("leads needing geo: %w", err)
	}
	targets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (geoTarget, error) {
		var t geoTarget
		err := row.Scan(&t.ID, &t.Address.Street, &t.Address.HouseNumber, &t.Address.PostalCode, &t.Address.City)
		return t, err
	})
	return targets, remaining, err
}

func (r *Repository) leadAddress(ctx context.Context, id string) (Address, error) {
	var a Address
	err := r.db.QueryRow(ctx, `
		SELECT street, COALESCE(house_number, ''), postal_code, city FROM leads WHERE id = $1`, id).
		Scan(&a.Street, &a.HouseNumber, &a.PostalCode, &a.City)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// GeoStats counts unique leads by service area for the dashboard.
type GeoStats struct {
	InArea     int      `json:"inArea"`
	OutOfArea  int      `json:"outOfArea"`
	NoGeo      int      `json:"noGeo"`    // not checked yet or failed
	NotFound   int      `json:"notFound"` // geocoder found no address
	States     []string `json:"states"`   // configured service area
	Configured bool     `json:"configured"`
}

func (r *Repository) GeoStats(ctx context.Context, area ServiceArea) (GeoStats, error) {
	s := GeoStats{States: area.States(), Configured: area.Configured()}
	err := r.db.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE geo_status = 'ok' AND lower(geo_state) = ANY($1)),
		  count(*) FILTER (WHERE geo_status = 'ok' AND NOT (lower(COALESCE(geo_state, '')) = ANY($1))),
		  count(*) FILTER (WHERE geo_status IS NULL OR geo_status = 'error'),
		  count(*) FILTER (WHERE geo_status = 'not_found')
		FROM leads WHERE duplicate_of IS NULL`, area.States()).
		Scan(&s.InArea, &s.OutOfArea, &s.NoGeo, &s.NotFound)
	return s, err
}

// Geoapify answers in 3–15 s in practice, so the limits differ per use:
const (
	// enrichTimeout bounds geocoding during a form submission: the customer
	// should never wait long for a bonus feature. Slower answers are stored
	// as "error" and fetched later from the admin area.
	enrichTimeout = 3 * time.Second
	// adminGeocodeTimeout is used when sales explicitly asks for geo data
	// (detail page, backfill). Must stay below the server's 15 s request limit.
	adminGeocodeTimeout = 12 * time.Second
	// saveGeoTimeout gives storing the outcome its own deadline.
	saveGeoTimeout = 3 * time.Second
)
