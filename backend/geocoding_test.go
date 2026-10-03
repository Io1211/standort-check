package backend

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeGeoapify(t *testing.T, status int, body string, check func(*http.Request)) *Geoapify {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	g := NewGeoapify("test-key")
	g.baseURL = srv.URL
	return g
}

const dresdenResponse = `{"results":[{"country_code":"de","state":"Sachsen","county":"Dresden",
	"city":"Dresden","postcode":"01067","lat":51.05,"lon":13.73,
	"formatted":"Hauptstraße 14, 01067 Dresden, Deutschland","result_type":"building",
	"rank":{"confidence":1}}]}`

func TestGeoapifySendsStructuredAddress(t *testing.T) {
	g := fakeGeoapify(t, 200, dresdenResponse, func(r *http.Request) {
		q := r.URL.Query()
		want := map[string]string{
			"street": "Hauptstraße", "housenumber": "14", "postcode": "01067", "city": "Dresden",
			"filter": "countrycode:de", "lang": "de", "format": "json", "apiKey": "test-key",
		}
		for k, v := range want {
			if q.Get(k) != v {
				t.Errorf("query %s = %q, want %q", k, q.Get(k), v)
			}
		}
	})

	res, err := g.Geocode(context.Background(), Address{Street: "Hauptstraße", HouseNumber: "14", PostalCode: "01067", City: "Dresden"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != GeoStatusOK || res.State != "Sachsen" || res.Municipality != "Dresden" || res.Lat != 51.05 {
		t.Errorf("unexpected result %+v", res)
	}
}

func TestGeoapifyOmitsMissingHouseNumber(t *testing.T) {
	g := fakeGeoapify(t, 200, dresdenResponse, func(r *http.Request) {
		if _, ok := r.URL.Query()["housenumber"]; ok {
			t.Error("housenumber must not be sent for plots without a number")
		}
	})
	if _, err := g.Geocode(context.Background(), Address{Street: "Am Feld", PostalCode: "01067", City: "Dresden"}); err != nil {
		t.Fatal(err)
	}
}

func TestGeoapifyVillageBecomesMunicipality(t *testing.T) {
	g := fakeGeoapify(t, 200, `{"results":[{"country_code":"de","state":"Schleswig-Holstein",
		"county":"Kreis Herzogtum Lauenburg","village":"Groß Grönau","postcode":"23627",
		"result_type":"building","rank":{"confidence":0.95}}]}`, nil)
	res, err := g.Geocode(context.Background(), Address{Street: "Am Mühlenteich", HouseNumber: "7", PostalCode: "23627", City: "Groß Grönau"})
	if err != nil || res.Municipality != "Groß Grönau" || res.County != "Kreis Herzogtum Lauenburg" {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func TestGeoapifyNotFound(t *testing.T) {
	g := fakeGeoapify(t, 200, `{"results":[]}`, nil)
	res, err := g.Geocode(context.Background(), Address{Street: "Gibtsnicht", PostalCode: "99999", City: "Nirgendwo"})
	if !errors.Is(err, ErrNoGeoMatch) || res.Status != GeoStatusNotFound {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func TestGeoapifyRejectsForeignResult(t *testing.T) {
	g := fakeGeoapify(t, 200, `{"results":[{"country_code":"at","state":"Tirol","city":"Innsbruck"}]}`, nil)
	if _, err := g.Geocode(context.Background(), Address{Street: "Annaweg", PostalCode: "60200", City: "Innsbruck"}); !errors.Is(err, ErrNoGeoMatch) {
		t.Fatalf("a result outside Germany must count as not found, got %v", err)
	}
}

// Real case: Geoapify returned a "Annaweg 3" in Baden-Württemberg with
// confidence 0 for an Austrian address with an invented 5-digit postcode.
func TestGeoapifyUnsureGuessElsewhereIsNotFound(t *testing.T) {
	g := fakeGeoapify(t, 200, `{"results":[{"country_code":"de","state":"Baden-Württemberg",
		"city":"Dietenheim","postcode":"89165","result_type":"building","rank":{"confidence":0}}]}`, nil)
	if _, err := g.Geocode(context.Background(), Address{Street: "Annaweg", HouseNumber: "3", PostalCode: "60200", City: "Innsbruck"}); !errors.Is(err, ErrNoGeoMatch) {
		t.Fatalf("unsure match in another postcode must be not found, got %v", err)
	}
}

// Real case: "Am Mühlenteich 7, 23627 Groß Grönau" – only the town is found
// (confidence 0.25), but in the right postcode. That is still useful for the
// service area and must be kept (shown as imprecise).
func TestGeoapifyUnsureMatchInSamePostcodeIsKept(t *testing.T) {
	g := fakeGeoapify(t, 200, `{"results":[{"country_code":"de","state":"Schleswig-Holstein",
		"village":"Groß Grönau","postcode":"23627","result_type":"city","rank":{"confidence":0.25}}]}`, nil)
	res, err := g.Geocode(context.Background(), Address{Street: "Am Mühlenteich", HouseNumber: "7", PostalCode: "23627", City: "Groß Grönau"})
	if err != nil || res.State != "Schleswig-Holstein" {
		t.Fatalf("got %+v, %v", res, err)
	}
}

func TestGeoapifyErrorsDoNotLeakAPIKey(t *testing.T) {
	g := fakeGeoapify(t, 401, `{"message":"Invalid apiKey"}`, nil)
	_, err := g.Geocode(context.Background(), Address{Street: "x", PostalCode: "01067", City: "y"})
	if err == nil || strings.Contains(err.Error(), "test-key") {
		t.Fatalf("expected error without API key, got %v", err)
	}

	// Network error: *url.Error would normally include the full URL.
	g = NewGeoapify("secret-key")
	g.baseURL = "http://127.0.0.1:1"
	_, err = g.Geocode(context.Background(), Address{Street: "x", PostalCode: "01067", City: "y"})
	if err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("network error must not contain the API key: %v", err)
	}
}

func TestServiceAreaAndDerivedSignals(t *testing.T) {
	area := NewServiceArea(" Sachsen, hamburg ,")
	conf := 1.0
	low := 0.4

	cases := []struct {
		name         string
		geo          GeoInfo
		input        string
		wantIn       *bool
		wantMismatch bool
		wantImprec   bool
	}{
		{"in area, exact", GeoInfo{Status: GeoStatusOK, State: "Sachsen", Postcode: "01067", ResultType: "building", Confidence: &conf}, "01067", ptr(true), false, false},
		{"case-insensitive state", GeoInfo{Status: GeoStatusOK, State: "Hamburg", Postcode: "22765", ResultType: "building", Confidence: &conf}, "22765", ptr(true), false, false},
		{"out of area", GeoInfo{Status: GeoStatusOK, State: "Bayern", Postcode: "80331", ResultType: "building", Confidence: &conf}, "80331", ptr(false), false, false},
		{"postcode differs", GeoInfo{Status: GeoStatusOK, State: "Sachsen", Postcode: "01069", ResultType: "building", Confidence: &conf}, "01067", ptr(true), true, false},
		{"only town found", GeoInfo{Status: GeoStatusOK, State: "Sachsen", Postcode: "01067", ResultType: "city", Confidence: &conf}, "01067", ptr(true), false, true},
		{"low confidence", GeoInfo{Status: GeoStatusOK, State: "Sachsen", Postcode: "01067", ResultType: "building", Confidence: &low}, "01067", ptr(true), false, true},
		{"not geocoded", GeoInfo{}, "01067", nil, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.geo
			g.derive(tc.input, area)
			if !sameBoolPtr(g.InServiceArea, tc.wantIn) || g.PostcodeMismatch != tc.wantMismatch || g.Imprecise != tc.wantImprec {
				t.Errorf("in=%v mismatch=%v imprecise=%v", fmtPtr(g.InServiceArea), g.PostcodeMismatch, g.Imprecise)
			}
		})
	}

	g := GeoInfo{Status: GeoStatusOK, State: "Sachsen"}
	g.derive("01067", NewServiceArea(""))
	if g.InServiceArea != nil {
		t.Error("without configured service area the result must be unknown (nil)")
	}
}

func ptr(b bool) *bool { return &b }

func sameBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fmtPtr(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}
