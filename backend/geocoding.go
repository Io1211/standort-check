package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Geo enrichment: turns the plot address into municipality, county, state
// and coordinates, so sales can see at a glance whether a request lies in
// the service area.
//
// Geocoding is a bonus. It never blocks or fails a lead submission: errors
// are stored as geo_status 'error' and can be retried from the dashboard.

type GeoStatus string

const (
	GeoStatusOK       GeoStatus = "ok"
	GeoStatusNotFound GeoStatus = "not_found"
	GeoStatusError    GeoStatus = "error"
)

type Address struct {
	Street      string
	HouseNumber string
	PostalCode  string
	City        string
}

type GeoResult struct {
	Status       GeoStatus
	Lat, Lon     float64
	Municipality string
	County       string
	State        string
	Postcode     string
	Formatted    string
	ResultType   string
	Confidence   float64
}

type Geocoder interface {
	Geocode(ctx context.Context, a Address) (GeoResult, error)
}

// ErrNoGeoMatch means the geocoder answered but found no German address.
var ErrNoGeoMatch = errors.New("address not found")

// minConfidenceOtherPostcode: below this confidence, a result in a different
// postcode than entered is treated as "not found" (it is a guess elsewhere).
const minConfidenceOtherPostcode = 0.5

// Geoapify implements Geocoder with the Geoapify geocoding API
// (Geoapify GmbH, Germany; free tier 3,000 requests/day). Only the plot
// address is sent – never name, email or phone.
type Geoapify struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewGeoapify(apiKey string) *Geoapify {
	return &Geoapify{
		apiKey:  apiKey,
		baseURL: "https://api.geoapify.com",
		// Upper bound only; callers set the real deadline via ctx.
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

type geoapifyResponse struct {
	Results []struct {
		CountryCode  string  `json:"country_code"`
		State        string  `json:"state"`
		County       string  `json:"county"`
		City         string  `json:"city"`
		Town         string  `json:"town"`
		Village      string  `json:"village"`
		Municipality string  `json:"municipality"`
		Postcode     string  `json:"postcode"`
		Lat          float64 `json:"lat"`
		Lon          float64 `json:"lon"`
		Formatted    string  `json:"formatted"`
		ResultType   string  `json:"result_type"`
		Rank         struct {
			Confidence float64 `json:"confidence"`
		} `json:"rank"`
	} `json:"results"`
}

// Geocode uses the structured search (separate street, house number,
// postcode, city) restricted to Germany. Structured input is more reliable
// than one free-text line and matches how the form collects the address.
func (g *Geoapify) Geocode(ctx context.Context, a Address) (GeoResult, error) {
	q := url.Values{
		"street":   {a.Street},
		"postcode": {a.PostalCode},
		"city":     {a.City},
		"country":  {"Germany"},
		"filter":   {"countrycode:de"},
		"lang":     {"de"},
		"limit":    {"1"},
		"format":   {"json"},
		"apiKey":   {g.apiKey},
	}
	if a.HouseNumber != "" {
		q.Set("housenumber", a.HouseNumber)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"/v1/geocode/search?"+q.Encode(), nil)
	if err != nil {
		return GeoResult{}, err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		// *url.Error contains the full URL including the API key – never
		// let it reach the logs.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return GeoResult{}, fmt.Errorf("geoapify request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GeoResult{}, fmt.Errorf("geoapify: HTTP %d", resp.StatusCode)
	}
	var body geoapifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return GeoResult{}, fmt.Errorf("geoapify: decode: %w", err)
	}
	if len(body.Results) == 0 || !strings.EqualFold(body.Results[0].CountryCode, "de") {
		return GeoResult{Status: GeoStatusNotFound}, ErrNoGeoMatch
	}

	r := body.Results[0]
	// Geoapify always returns its best guess. A very unsure match in another
	// postcode is a different place (seen in practice: "Annaweg 3, 60200
	// Innsbruck" → a "Annaweg" in Baden-Württemberg with confidence 0).
	// Showing that location would mislead sales, so it counts as not found.
	if r.Rank.Confidence < minConfidenceOtherPostcode && r.Postcode != a.PostalCode {
		return GeoResult{Status: GeoStatusNotFound}, ErrNoGeoMatch
	}
	return GeoResult{
		Status:       GeoStatusOK,
		Lat:          r.Lat,
		Lon:          r.Lon,
		Municipality: firstNonEmpty(r.City, r.Town, r.Village, r.Municipality),
		County:       r.County,
		State:        r.State,
		Postcode:     r.Postcode,
		Formatted:    r.Formatted,
		ResultType:   r.ResultType,
		Confidence:   r.Rank.Confidence,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// GeoInfo is the geo part of a lead as the dashboard sees it, including the
// signals derived from it.
type GeoInfo struct {
	Status       GeoStatus  `json:"status"` // "" = not checked yet
	Lat          *float64   `json:"lat"`
	Lon          *float64   `json:"lon"`
	Municipality string     `json:"municipality"`
	County       string     `json:"county"`
	State        string     `json:"state"`
	Postcode     string     `json:"postcode"`
	Formatted    string     `json:"formatted"`
	ResultType   string     `json:"resultType"`
	Confidence   *float64   `json:"confidence"`
	CheckedAt    *time.Time `json:"checkedAt"`

	// InServiceArea is nil when unknown (no geo data or no area configured).
	InServiceArea *bool `json:"inServiceArea"`
	// PostcodeMismatch: the geocoder places the address in another postcode
	// than the customer entered – a typo, or an invented postcode.
	PostcodeMismatch bool `json:"postcodeMismatch"`
	// Imprecise: only the street, postcode or town was found, or the
	// geocoder is unsure. Coordinates are then only approximate.
	Imprecise bool `json:"imprecise"`
}

// ServiceArea is the list of federal states the company serves, configured
// via SERVICE_AREA_STATES (e.g. "Sachsen,Hamburg,Schleswig-Holstein").
type ServiceArea struct {
	states map[string]bool
}

func NewServiceArea(csv string) ServiceArea {
	sa := ServiceArea{states: map[string]bool{}}
	for _, s := range strings.Split(csv, ",") {
		if s = strings.TrimSpace(s); s != "" {
			sa.states[strings.ToLower(s)] = true
		}
	}
	return sa
}

func (sa ServiceArea) Configured() bool { return len(sa.states) > 0 }

func (sa ServiceArea) States() []string {
	out := make([]string, 0, len(sa.states))
	for s := range sa.states {
		out = append(out, s)
	}
	return out
}

func (sa ServiceArea) Contains(state string) *bool {
	if !sa.Configured() || state == "" {
		return nil
	}
	in := sa.states[strings.ToLower(state)]
	return &in
}

// derive fills the computed fields of a GeoInfo for a lead.
func (g *GeoInfo) derive(inputPostcode string, area ServiceArea) {
	if g.Status != GeoStatusOK {
		return
	}
	g.InServiceArea = area.Contains(g.State)
	g.PostcodeMismatch = g.Postcode != "" && g.Postcode != inputPostcode
	precise := g.ResultType == "building" || g.ResultType == "street" || g.ResultType == "amenity"
	g.Imprecise = !precise || (g.Confidence != nil && *g.Confidence < 0.7)
}
