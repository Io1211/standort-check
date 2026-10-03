package backend

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Read and update queries for the admin dashboard.

// ErrNotFound is returned when a lead ID does not exist.
var ErrNotFound = errors.New("not found")

// LeadFilter holds the dashboard's search, filter and sort options. It only
// contains validated values (status, sort and order are whitelisted by the
// handler).
type LeadFilter struct {
	Search         string
	Status         LeadStatus // "" = all
	Source         string     // "" = all, NoValue = leads without utm_source
	Campaign       string     // "" = all, NoValue = leads without utm_campaign
	HideDuplicates bool
	Area           string   // "" = all, "in", "out", "unknown" (service area)
	AreaStates     []string // configured service area (lowercase), set by the handler
	Sort           string   // key of sortColumns
	Desc           bool
	Limit          int // 0 = no limit (export)
	Offset         int
}

// NoValue is the filter value for "no source / no campaign" (direct traffic).
const NoValue = "__none__"

// sortColumns maps the public sort keys to SQL. ORDER BY cannot use query
// parameters, so only these fixed expressions ever end up in the query.
var sortColumns = map[string][]string{
	"created_at": {"l.created_at"},
	"name":       {"lower(l.last_name)", "lower(l.first_name)"},
	"city":       {"l.city_normalized"},
	"status": {"CASE l.status WHEN 'new' THEN 0 WHEN 'contacted' THEN 1 " +
		"WHEN 'qualified' THEN 2 ELSE 3 END"},
}

const leadSelect = `
	SELECT l.id, l.first_name, l.last_name, l.email, l.phone,
	       l.street, COALESCE(l.house_number, ''), l.postal_code, l.city, COALESCE(l.parcel_note, ''),
	       COALESCE(l.utm_source, ''), COALESCE(l.utm_medium, ''), COALESCE(l.utm_campaign, ''),
	       COALESCE(l.utm_content, ''), COALESCE(l.utm_term, ''),
	       COALESCE(l.gclid, ''), COALESCE(l.fbclid, ''), COALESCE(l.referrer, ''),
	       l.status, l.duplicate_of::text, l.created_at,
	       (SELECT count(*) FROM leads d WHERE d.duplicate_of = l.id),
	       COALESCE(l.geo_status, ''), l.geo_lat, l.geo_lon,
	       COALESCE(l.geo_municipality, ''), COALESCE(l.geo_county, ''), COALESCE(l.geo_state, ''),
	       COALESCE(l.geo_postcode, ''), COALESCE(l.geo_formatted, ''), COALESCE(l.geo_result_type, ''),
	       l.geo_confidence::float8, l.geo_checked_at
	FROM leads l`

func scanLead(row pgx.Row) (LeadListItem, error) {
	var l LeadListItem
	err := row.Scan(&l.ID, &l.FirstName, &l.LastName, &l.Email, &l.Phone,
		&l.Street, &l.HouseNumber, &l.PostalCode, &l.City, &l.ParcelNote,
		&l.UTMSource, &l.UTMMedium, &l.UTMCampaign, &l.UTMContent, &l.UTMTerm,
		&l.GCLID, &l.FBCLID, &l.Referrer,
		&l.Status, &l.DuplicateOf, &l.CreatedAt, &l.DuplicateCount,
		&l.Geo.Status, &l.Geo.Lat, &l.Geo.Lon,
		&l.Geo.Municipality, &l.Geo.County, &l.Geo.State,
		&l.Geo.Postcode, &l.Geo.Formatted, &l.Geo.ResultType,
		&l.Geo.Confidence, &l.Geo.CheckedAt)
	return l, err
}

// where builds the WHERE clause. All user input goes into args ($1, $2, …),
// never into the SQL string.
func (f LeadFilter) where() (string, []any) {
	var conds []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	if s := strings.TrimSpace(f.Search); s != "" {
		p := arg("%" + escapeLike(s) + "%")
		parts := []string{
			"l.first_name || ' ' || l.last_name ILIKE " + p,
			"l.last_name || ' ' || l.first_name ILIKE " + p,
			"l.email ILIKE " + p,
			"l.phone ILIKE " + p,
			"l.street ILIKE " + p,
			"l.city ILIKE " + p,
			"l.postal_code LIKE " + p,
			"l.utm_campaign ILIKE " + p,
		}
		// "0170 123" should find "+49 170123…": compare digits without trunk 0.
		if digits := strings.TrimLeft(onlyDigits(s), "0"); len(digits) >= 4 {
			parts = append(parts, "l.phone_normalized LIKE "+arg("%"+digits+"%"))
		}
		conds = append(conds, "("+strings.Join(parts, " OR ")+")")
	}
	if f.Status != "" {
		conds = append(conds, "l.status = "+arg(string(f.Status)))
	}
	switch f.Source {
	case "":
	case NoValue:
		conds = append(conds, "l.utm_source IS NULL")
	default:
		conds = append(conds, "l.utm_source = "+arg(f.Source))
	}
	switch f.Campaign {
	case "":
	case NoValue:
		conds = append(conds, "l.utm_campaign IS NULL")
	default:
		conds = append(conds, "l.utm_campaign = "+arg(f.Campaign))
	}
	// Service area: only meaningful when an area is configured.
	if len(f.AreaStates) > 0 {
		switch f.Area {
		case "in":
			conds = append(conds, "l.geo_status = 'ok' AND lower(l.geo_state) = ANY("+arg(f.AreaStates)+")")
		case "out":
			conds = append(conds, "l.geo_status = 'ok' AND NOT (lower(COALESCE(l.geo_state, '')) = ANY("+arg(f.AreaStates)+"))")
		case "unknown":
			conds = append(conds, "(l.geo_status IS NULL OR l.geo_status <> 'ok')")
		}
	}
	if f.HideDuplicates {
		conds = append(conds, "l.duplicate_of IS NULL")
	}

	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (f LeadFilter) orderBy() string {
	cols, ok := sortColumns[f.Sort]
	if !ok {
		cols = sortColumns["created_at"]
	}
	dir := " ASC"
	if f.Desc {
		dir = " DESC"
	}
	parts := make([]string, 0, len(cols)+2)
	for _, c := range cols {
		parts = append(parts, c+dir)
	}
	// Stable order for equal values (and for paging).
	parts = append(parts, "l.created_at DESC", "l.id")
	return " ORDER BY " + strings.Join(parts, ", ")
}

// ListLeads returns one page of leads plus the total number of matches.
func (r *Repository) ListLeads(ctx context.Context, f LeadFilter) ([]LeadListItem, int, error) {
	where, args := f.where()

	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM leads l"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count leads: %w", err)
	}

	query := leadSelect + where + f.orderBy()
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	leads, err := r.queryLeads(ctx, query, args...)
	return leads, total, err
}

func (r *Repository) queryLeads(ctx context.Context, query string, args ...any) ([]LeadListItem, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query leads: %w", err)
	}
	defer rows.Close()

	leads := []LeadListItem{}
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		leads = append(leads, l)
	}
	return leads, rows.Err()
}

func (r *Repository) GetLead(ctx context.Context, id string) (LeadDetail, error) {
	var d LeadDetail
	lead, err := scanLead(r.db.QueryRow(ctx, leadSelect+" WHERE l.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("get lead: %w", err)
	}
	d.Lead = lead

	if lead.DuplicateOf != nil {
		originals, err := r.summaries(ctx, "id = $1", *lead.DuplicateOf)
		if err != nil {
			return d, err
		}
		if len(originals) > 0 {
			d.Original = &originals[0]
		}
	}
	d.Duplicates, err = r.summaries(ctx, "duplicate_of = $1", id)
	return d, err
}

func (r *Repository) summaries(ctx context.Context, cond string, arg any) ([]LeadSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, first_name, last_name, email, status, created_at
		FROM leads WHERE `+cond+` ORDER BY created_at`, arg)
	if err != nil {
		return nil, fmt.Errorf("query summaries: %w", err)
	}
	defer rows.Close()

	out := []LeadSummary{}
	for rows.Next() {
		var s LeadSummary
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan summary: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status LeadStatus) error {
	tag, err := r.db.Exec(ctx, `UPDATE leads SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type LeadFilterOption struct {
	Source   string `json:"source"`
	Campaign string `json:"campaign"`
}

// LeadFilterOptions avoids computing status totals just to populate dropdowns.
func (r *Repository) LeadFilterOptions(ctx context.Context) ([]LeadFilterOption, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT COALESCE(utm_source, ''), COALESCE(utm_campaign, '') FROM leads ORDER BY 1, 2`)
	if err != nil {
		return nil, fmt.Errorf("lead filter options: %w", err)
	}
	options, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (LeadFilterOption, error) {
		var option LeadFilterOption
		err := row.Scan(&option.Source, &option.Campaign)
		return option, err
	})
	if options == nil {
		options = []LeadFilterOption{}
	}
	return options, err
}

// CampaignStats groups leads by source and campaign. Duplicates are counted
// in Total but not in the status columns, so one person submitting twice does
// not make a campaign look better.
//
// days > 0 limits the evaluation to leads received in the last n days.
func (r *Repository) CampaignStats(ctx context.Context, days int) ([]CampaignStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(utm_source, ''), COALESCE(utm_campaign, ''),
		       count(*),
		       count(*) FILTER (WHERE duplicate_of IS NULL),
		       count(*) FILTER (WHERE duplicate_of IS NULL AND status = 'new'),
		       count(*) FILTER (WHERE duplicate_of IS NULL AND status = 'contacted'),
		       count(*) FILTER (WHERE duplicate_of IS NULL AND status = 'qualified'),
		       count(*) FILTER (WHERE duplicate_of IS NULL AND status = 'not_qualified')
		FROM leads
		WHERE $1 <= 0 OR created_at >= now() - make_interval(days => $1)
		GROUP BY 1, 2
		ORDER BY 4 DESC, 1, 2`, days)
	if err != nil {
		return nil, fmt.Errorf("campaign stats: %w", err)
	}
	defer rows.Close()

	out := []CampaignStats{}
	for rows.Next() {
		var s CampaignStats
		if err := rows.Scan(&s.Source, &s.Campaign, &s.Total, &s.Unique,
			&s.New, &s.Contacted, &s.Qualified, &s.NotQualified); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// WeeklyLeads is the number of unique leads per source and calendar week.
type WeeklyLeads struct {
	Week   string `json:"week"` // Monday of the week, YYYY-MM-DD (German time)
	Source string `json:"source"`
	Leads  int    `json:"leads"`
}

// WeeklyLeadsBySource returns the last n calendar weeks (including the
// current one) and the unique leads per source in each. All weeks are listed,
// so the chart shows weeks without leads as 0 instead of skipping them.
func (r *Repository) WeeklyLeadsBySource(ctx context.Context, n int) (weeks []string, rows []WeeklyLeads, err error) {
	weekRows, err := r.db.Query(ctx, `
		SELECT to_char(w, 'YYYY-MM-DD')
		FROM generate_series(
			date_trunc('week', now() AT TIME ZONE 'Europe/Berlin') - make_interval(weeks => $1 - 1),
			date_trunc('week', now() AT TIME ZONE 'Europe/Berlin'),
			interval '1 week') AS w`, n)
	if err != nil {
		return nil, nil, fmt.Errorf("weeks: %w", err)
	}
	weeks, err = pgx.CollectRows(weekRows, pgx.RowTo[string])
	if err != nil {
		return nil, nil, fmt.Errorf("scan weeks: %w", err)
	}

	dataRows, err := r.db.Query(ctx, `
		SELECT to_char(date_trunc('week', created_at AT TIME ZONE 'Europe/Berlin'), 'YYYY-MM-DD'),
		       COALESCE(utm_source, ''), count(*)
		FROM leads
		WHERE duplicate_of IS NULL
		  AND created_at >= (date_trunc('week', now() AT TIME ZONE 'Europe/Berlin')
		      - make_interval(weeks => $1 - 1)) AT TIME ZONE 'Europe/Berlin'
		GROUP BY 1, 2
		ORDER BY 1, 2`, n)
	if err != nil {
		return nil, nil, fmt.Errorf("weekly leads: %w", err)
	}
	rows, err = pgx.CollectRows(dataRows, func(row pgx.CollectableRow) (WeeklyLeads, error) {
		var w WeeklyLeads
		err := row.Scan(&w.Week, &w.Source, &w.Leads)
		return w, err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("scan weekly leads: %w", err)
	}
	return weeks, rows, nil
}

// DeleteLead permanently removes a lead (e.g. a GDPR deletion request or
// test data).
//
// Leads that were marked as duplicates of it are not simply orphaned: in the
// same transaction each of them is checked again against the remaining older
// leads at the same address, with the normal duplicate rule. The address lock
// keeps a new submission for that plot from interleaving.
func (r *Repository) DeleteLead(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var addr Lead
	err = tx.QueryRow(ctx, `
		SELECT postal_code, street_normalized, house_number_normalized, city_normalized
		FROM leads WHERE id = $1`, id).
		Scan(&addr.PostalCode, &addr.StreetNormalized, &addr.HouseNumberNormalized, &addr.CityNormalized)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load lead: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, addressKey(addr)); err != nil {
		return fmt.Errorf("lock address: %w", err)
	}

	// Detach former duplicates first, otherwise the foreign key blocks the delete.
	detached, err := tx.Query(ctx, `UPDATE leads SET duplicate_of = NULL WHERE duplicate_of = $1 RETURNING id`, id)
	if err != nil {
		return fmt.Errorf("detach duplicates: %w", err)
	}
	formerDuplicates, err := pgx.CollectRows(detached, pgx.RowTo[string])
	if err != nil {
		return fmt.Errorf("scan detached: %w", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM leads WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lead: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// Re-link each former duplicate to the oldest remaining match, if any.
	if len(formerDuplicates) > 0 {
		remaining, err := findSameAddress(ctx, tx, addr) // oldest first
		if err != nil {
			return err
		}
		for i, l := range remaining {
			if !slices.Contains(formerDuplicates, l.ID) {
				continue
			}
			if dup := pickDuplicate(l, remaining[:i]); dup != nil {
				if _, err := tx.Exec(ctx, `UPDATE leads SET duplicate_of = $2 WHERE id = $1`, l.ID, dup.ID); err != nil {
					return fmt.Errorf("relink duplicate: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// escapeLike makes %, _ and \ in user input match literally in LIKE.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
