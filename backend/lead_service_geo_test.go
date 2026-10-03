package backend

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memStore struct {
	saved []GeoResult
}

func (m *memStore) CreateLead(_ context.Context, l Lead) (Lead, error) {
	l.ID = "lead-1"
	return l, nil
}

// SaveGeo behaves like the database: it fails when its context is done.
func (m *memStore) SaveGeo(ctx context.Context, _ string, g GeoResult) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.saved = append(m.saved, g)
	return nil
}

type stubGeocoder struct {
	res   GeoResult
	err   error
	delay time.Duration
}

func (s stubGeocoder) Geocode(ctx context.Context, _ Address) (GeoResult, error) {
	select {
	case <-time.After(s.delay):
		return s.res, s.err
	case <-ctx.Done():
		return GeoResult{}, ctx.Err()
	}
}

func TestCreateLeadSurvivesGeocodingFailures(t *testing.T) {
	cases := []struct {
		name string
		geo  stubGeocoder
		want GeoStatus
	}{
		{"success", stubGeocoder{res: GeoResult{Status: GeoStatusOK, State: "Sachsen"}}, GeoStatusOK},
		{"not found", stubGeocoder{res: GeoResult{Status: GeoStatusNotFound}, err: ErrNoGeoMatch}, GeoStatusNotFound},
		{"api down", stubGeocoder{err: errors.New("HTTP 503")}, GeoStatusError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &memStore{}
			svc := NewLeadService(store).WithGeocoder(tc.geo, store)

			lead, err := svc.Create(context.Background(), validRequest())
			if err != nil || lead.ID == "" {
				t.Fatalf("lead must be stored regardless of geocoding, got %v", err)
			}
			if len(store.saved) != 1 || store.saved[0].Status != tc.want {
				t.Fatalf("saved geo = %+v, want status %s", store.saved, tc.want)
			}
		})
	}
}

func TestCreateLeadGeocodingIsBounded(t *testing.T) {
	store := &memStore{}
	svc := NewLeadService(store).WithGeocoder(stubGeocoder{delay: time.Minute}, store)

	start := time.Now()
	if _, err := svc.Create(context.Background(), validRequest()); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > enrichTimeout+time.Second {
		t.Fatalf("a hanging geocoder must not block the submission, took %s", took)
	}
	if len(store.saved) != 1 || store.saved[0].Status != GeoStatusError {
		t.Fatalf("timeout should be stored as error, got %+v", store.saved)
	}
}

func TestCreateLeadWithoutGeocoder(t *testing.T) {
	store := &memStore{}
	if _, err := NewLeadService(store).Create(context.Background(), validRequest()); err != nil {
		t.Fatal(err)
	}
	if len(store.saved) != 0 {
		t.Fatal("without GEOAPIFY_API_KEY nothing must be geocoded")
	}
}
