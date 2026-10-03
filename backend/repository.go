package backend

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateLead stores a normalized lead and sets duplicate_of in one transaction.
//
// A transaction-scoped advisory lock on the normalized address serializes
// concurrent submissions for the same plot (e.g. a double-clicked submit
// button), so the second request always sees the first one. Different
// addresses never block each other.
func (r *Repository) CreateLead(ctx context.Context, lead Lead) (Lead, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return lead, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after commit

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, addressKey(lead)); err != nil {
		return lead, fmt.Errorf("lock address: %w", err)
	}

	candidates, err := findSameAddress(ctx, tx, lead)
	if err != nil {
		return lead, err
	}
	if dup := pickDuplicate(lead, candidates); dup != nil {
		lead.DuplicateOf = &dup.ID
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO leads (
			first_name, last_name,
			email, email_normalized,
			phone, phone_normalized,
			street, street_normalized,
			house_number, house_number_normalized,
			postal_code,
			city, city_normalized,
			parcel_note,
			utm_source, utm_medium, utm_campaign, utm_content, utm_term,
			gclid, fbclid, referrer,
			status, duplicate_of
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		RETURNING id, created_at`,
		lead.FirstName, lead.LastName,
		lead.Email, lead.EmailNormalized,
		lead.Phone, lead.PhoneNormalized,
		lead.Street, lead.StreetNormalized,
		nullIfEmpty(lead.HouseNumber), lead.HouseNumberNormalized,
		lead.PostalCode,
		lead.City, lead.CityNormalized,
		nullIfEmpty(lead.ParcelNote),
		nullIfEmpty(lead.UTMSource), nullIfEmpty(lead.UTMMedium), nullIfEmpty(lead.UTMCampaign),
		nullIfEmpty(lead.UTMContent), nullIfEmpty(lead.UTMTerm),
		nullIfEmpty(lead.GCLID), nullIfEmpty(lead.FBCLID),
		nullIfEmpty(lead.Referrer),
		lead.Status, lead.DuplicateOf,
	).Scan(&lead.ID, &lead.CreatedAt)
	if err != nil {
		return lead, fmt.Errorf("insert lead: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return lead, fmt.Errorf("commit: %w", err)
	}
	return lead, nil
}

// findSameAddress returns all leads for the same normalized address, oldest
// first. Whether one of them is a duplicate is decided by IsDuplicate.
func findSameAddress(ctx context.Context, tx pgx.Tx, lead Lead) ([]Lead, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, email_normalized, phone_normalized,
		       street_normalized, house_number_normalized, postal_code, city_normalized
		FROM leads
		WHERE postal_code = $1
		  AND street_normalized = $2
		  AND house_number_normalized = $3
		  AND city_normalized = $4
		ORDER BY created_at ASC, id ASC`,
		lead.PostalCode, lead.StreetNormalized, lead.HouseNumberNormalized, lead.CityNormalized,
	)
	if err != nil {
		return nil, fmt.Errorf("find same address: %w", err)
	}
	defer rows.Close()

	var out []Lead
	for rows.Next() {
		var l Lead
		if err := rows.Scan(&l.ID, &l.EmailNormalized, &l.PhoneNormalized,
			&l.StreetNormalized, &l.HouseNumberNormalized, &l.PostalCode, &l.CityNormalized); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func addressKey(l Lead) string {
	return l.PostalCode + "|" + l.StreetNormalized + "|" + l.HouseNumberNormalized + "|" + l.CityNormalized
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
