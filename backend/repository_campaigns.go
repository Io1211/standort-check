package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicateCampaign is returned when the same link already exists.
var ErrDuplicateCampaign = errors.New("campaign link already exists")

// Leads are matched to a campaign by their UTM values: same source and
// campaign, and a matching content if the campaign defines one.
//
// Ad platforms fill placeholders such as Google's {creative} or Meta's
// {{ad.name}} on click, so the literal placeholder never appears in a lead.
// Every placeholder therefore acts as a wildcard and the rest of the content
// must match literally: "video_{creative}" matches "video_12345" but not
// "image_12345". A content that is only a placeholder matches every lead of
// the campaign, including leads without utm_content.
//
// contentPattern turns the campaign content into a LIKE pattern: the LIKE
// wildcards \, % and _ in the literal parts are escaped first, then each
// placeholder group becomes %.
const contentPattern = `regexp_replace(
	replace(replace(replace(c.utm_content, '\', '\\'), '%', '\%'), '_', '\_'),
	'\{+[^{}]*\}+', '%', 'g')`

const campaignLeadJoin = `
	LEFT JOIN leads l
	       ON l.utm_source = c.utm_source
	      AND l.utm_campaign = c.utm_campaign
	      AND (c.utm_content IS NULL
	           OR (c.utm_content LIKE '%{%' AND COALESCE(l.utm_content, '') LIKE ` + contentPattern + `)
	           OR l.utm_content = c.utm_content)`

const campaignSelect = `
	SELECT c.id, c.name, c.utm_source, c.utm_medium, c.utm_campaign, COALESCE(c.utm_content, ''),
	       COALESCE(c.notes, ''), c.archived, c.created_at,
	       count(l.id),
	       count(l.id) FILTER (WHERE l.duplicate_of IS NULL),
	       count(l.id) FILTER (WHERE l.duplicate_of IS NULL AND l.status = 'qualified')
	FROM campaigns c` + campaignLeadJoin

func scanCampaign(row pgx.Row) (Campaign, error) {
	var c Campaign
	err := row.Scan(&c.ID, &c.Name, &c.UTMSource, &c.UTMMedium, &c.UTMCampaign, &c.UTMContent,
		&c.Notes, &c.Archived, &c.CreatedAt, &c.Leads, &c.Unique, &c.Qualified)
	return c, err
}

func (r *Repository) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	rows, err := r.db.Query(ctx, campaignSelect+` GROUP BY c.id ORDER BY c.archived, c.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Campaign, error) { return scanCampaign(row) })
	if err != nil {
		return nil, fmt.Errorf("scan campaigns: %w", err)
	}
	return out, nil
}

func (r *Repository) getCampaign(ctx context.Context, id string) (Campaign, error) {
	c, err := scanCampaign(r.db.QueryRow(ctx, campaignSelect+` WHERE c.id = $1 GROUP BY c.id`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (r *Repository) CreateCampaign(ctx context.Context, req CampaignRequest) (Campaign, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO campaigns (name, utm_source, utm_medium, utm_campaign, utm_content, notes, archived)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		req.Name, req.UTMSource, req.UTMMedium, req.UTMCampaign, nullIfEmpty(req.UTMContent),
		nullIfEmpty(req.Notes), req.Archived,
	).Scan(&id)
	if err != nil {
		return Campaign{}, mapCampaignError(err)
	}
	return r.getCampaign(ctx, id)
}

func (r *Repository) UpdateCampaign(ctx context.Context, id string, req CampaignRequest) (Campaign, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE campaigns
		SET name = $2, utm_source = $3, utm_medium = $4, utm_campaign = $5,
		    utm_content = $6, notes = $7, archived = $8
		WHERE id = $1`,
		id, req.Name, req.UTMSource, req.UTMMedium, req.UTMCampaign, nullIfEmpty(req.UTMContent),
		nullIfEmpty(req.Notes), req.Archived,
	)
	if err != nil {
		return Campaign{}, mapCampaignError(err)
	}
	if tag.RowsAffected() == 0 {
		return Campaign{}, ErrNotFound
	}
	return r.getCampaign(ctx, id)
}

// DeleteCampaign removes only the link configuration. Leads keep their UTM
// values and stay fully visible in lists and evaluations.
func (r *Repository) DeleteCampaign(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM campaigns WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func mapCampaignError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrDuplicateCampaign
	}
	return fmt.Errorf("save campaign: %w", err)
}
