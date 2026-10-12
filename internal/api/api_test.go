package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"pvdiary/internal/ingest"
	"pvdiary/internal/store"
)

func newAPI(t *testing.T) (http.Handler, *store.Store) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "a.db"))
	t.Cleanup(func() { s.Close() })
	h := New(s, ingest.New(s), Config{EleCost: 0.18, CO2: 0.596, IngestToken: "secret"})
	return h, s
}

func TestIngestRequiresToken(t *testing.T) {
	h, _ := newAPI(t)
	body, _ := json.Marshal(map[string]any{"collector": "foxess", "ts": 1748746800, "power_w": 4000})
	req := httptest.NewRequest("POST", "/api/ingest/sample", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no token -> %d, want 401", rr.Code)
	}
	req = httptest.NewRequest("POST", "/api/ingest/sample", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("with token -> %d, want 204", rr.Code)
	}
}

func TestDayEndpoint(t *testing.T) {
	h, s := newAPI(t)
	id, _ := s.UpsertCollector(store.Collector{Key: "foxess", Name: "FoxESS", PVWatt: 8960})
	s.UpsertDay(id, "20250601", 48687, 5004)
	req := httptest.NewRequest("GET", "/api/day?date=20250601", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code %d", rr.Code)
	}
	var resp struct {
		Date     string `json:"date"`
		EnergyWh int    `json:"energy_wh"`
		PeakW    int    `json:"peak_w"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.EnergyWh != 48687 || resp.PeakW != 5004 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestMonthEmptyReturnsEmptyArray(t *testing.T) {
	h, _ := newAPI(t)
	req := httptest.NewRequest("GET", "/api/month?ym=209901", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code %d", rr.Code)
	}
	if got := rr.Body.String(); got != "[]\n" {
		t.Fatalf("body = %q, want %q", got, "[]\n")
	}
}

func TestDayBadDate(t *testing.T) {
	h, _ := newAPI(t)

	// Missing date
	req := httptest.NewRequest("GET", "/api/day", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("no date -> %d, want 400", rr.Code)
	}

	// Invalid date format
	req = httptest.NewRequest("GET", "/api/day?date=banana", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("bad date -> %d, want 400", rr.Code)
	}
}

func TestSummaryHasLastUpdate(t *testing.T) {
	h, s := newAPI(t)
	id, _ := s.UpsertCollector(store.Collector{Key: "c", Name: "c"})
	s.UpsertSample(id, 1749611100, 200)
	req := httptest.NewRequest("GET", "/api/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var resp struct {
		LastUpdate int64 `json:"last_update"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.LastUpdate != 1749611100 {
		t.Fatalf("last_update = %d, want 1749611100", resp.LastUpdate)
	}
}

func TestDayMax(t *testing.T) {
	h, s := newAPI(t)
	id, _ := s.UpsertCollector(store.Collector{Key: "c", Name: "c"})
	// 2025-06-01 12:00 CEST = 1748772000 ; dag ervoor zelfde slot hoger vermogen
	s.UpsertSample(id, 1748772000, 3000)
	s.UpsertSample(id, 1748772000-86400, 4500)
	req := httptest.NewRequest("GET", "/api/daymax?date=20250601&days=7", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code %d", rr.Code)
	}
	var m map[string]int
	json.Unmarshal(rr.Body.Bytes(), &m)
	if m["12:00"] != 4500 {
		t.Fatalf("slot 12:00 = %d, want 4500 (max over ±7d)", m["12:00"])
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/api/daymax?date=banana", nil))
	if rr.Code != 400 {
		t.Fatalf("bad date -> %d, want 400", rr.Code)
	}
}

func TestSummaryExposesSiteConfig(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "a.db"))
	t.Cleanup(func() { s.Close() })
	h := New(s, ingest.New(s), Config{Lat: 52.37, Lon: 4.89, Strings: []StringCfg{{Name: "Z", Wp: 4000, Az: 180, Tilt: 35, TempCoeff: -0.35}}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/api/summary", nil))
	var resp struct {
		Config struct {
			Lat     float64     `json:"lat"`
			Lon     float64     `json:"lon"`
			Strings []StringCfg `json:"strings"`
		} `json:"config"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Config.Lat != 52.37 || resp.Config.Lon != 4.89 || len(resp.Config.Strings) != 1 || resp.Config.Strings[0].Wp != 4000 {
		t.Fatalf("config = %+v", resp.Config)
	}
}
