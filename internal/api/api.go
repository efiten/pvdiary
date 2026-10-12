package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"

	"pvdiary/internal/ingest"
	"pvdiary/internal/store"
)

type StringCfg struct {
	Name      string  `json:"name"`
	Wp        int     `json:"wp"`
	Az        float64 `json:"az"`         // kompas-azimut van de string (graden)
	Tilt      float64 `json:"tilt"`       // dakhelling (graden)
	TempCoeff float64 `json:"temp_coeff"` // %/°C, negatief
}

type Config struct {
	EleCost     float64
	CO2         float64
	IngestToken string
	Lat         float64
	Lon         float64
	Strings     []StringCfg
}

type server struct {
	s   *store.Store
	ing *ingest.Service
	cfg Config
}

func New(s *store.Store, ing *ingest.Service, cfg Config) http.Handler {
	srv := &server{s: s, ing: ing, cfg: cfg}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/summary", srv.summary)
	mux.HandleFunc("GET /api/day", srv.day)
	mux.HandleFunc("GET /api/month", srv.month)
	mux.HandleFunc("GET /api/year", srv.year)
	mux.HandleFunc("GET /api/top", srv.top)
	mux.HandleFunc("GET /api/daymax", srv.daymax)
	mux.HandleFunc("POST /api/ingest/sample", srv.auth(srv.ingestSample))
	mux.HandleFunc("POST /api/ingest/day", srv.auth(srv.ingestDay))
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (srv *server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		want := "Bearer " + srv.cfg.IngestToken
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (srv *server) summary(w http.ResponseWriter, r *http.Request) {
	coll := r.URL.Query().Get("collector")
	collectors, err := srv.s.Collectors()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ky, err := srv.s.KWhYear(coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	last, err := srv.s.LastDate(coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	lastUpd, err := srv.s.LastUpdate(coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{
		"collectors":  collectors,
		"kwh_year":    ky,
		"last_date":   last,
		"last_update": lastUpd,
		"config": map[string]any{
			"ele_cost": srv.cfg.EleCost,
			"co2":      srv.cfg.CO2,
			"lat":      srv.cfg.Lat,
			"lon":      srv.cfg.Lon,
			"strings":  srv.cfg.Strings,
		},
	})
}

func isDateDigits(date string) bool {
	if len(date) != 8 {
		return false
	}
	for _, b := range date {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func (srv *server) day(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	date := q.Get("date")
	if !isDateDigits(date) {
		http.Error(w, "bad date", 400)
		return
	}
	coll := q.Get("collector")
	info, err := srv.s.DayInfo(date, coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	samples, err := srv.s.DaySamples(date, coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	pts := make([][2]int64, len(samples))
	for i, sm := range samples {
		pts[i] = [2]int64{sm.TS, int64(sm.PowerW)}
	}
	writeJSON(w, map[string]any{
		"date": date, "energy_wh": info.EnergyWh, "peak_w": info.PeakW, "samples": pts,
	})
}

func (srv *server) month(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	days, err := srv.s.Month(q.Get("ym"), q.Get("collector"))
	writeJSONList(w, days, err)
}

func writeJSONList(w http.ResponseWriter, days []store.DayTotal, err error) {
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, days)
}

func (srv *server) year(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	coll := q.Get("collector")
	y, _ := strconv.Atoi(q.Get("y"))
	months, err := srv.s.YearMonths(y, coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ky, err := srv.s.KWhYear(coll)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	years := make([]int, 0, len(ky))
	for yr := range ky {
		years = append(years, yr)
	}
	sort.Ints(years)
	writeJSON(w, map[string]any{"year": y, "months": months, "kwh_year": ky, "years": years})
}

func (srv *server) top(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scope := q.Get("scope")
	if scope == "" {
		scope = "global"
	}
	key := q.Get("key")
	n := 10
	if v, err := strconv.Atoi(q.Get("n")); err == nil && v > 0 {
		n = v
	}
	days, err := srv.s.TopDays(scope, key, n)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	peak, err := srv.s.TopPeak(scope, key, n)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]any{"days": days, "peak": peak})
}

type sampleReq struct {
	Collector string `json:"collector"`
	TS        int64  `json:"ts"`
	PowerW    int    `json:"power_w"`
}
type dayReq struct {
	Collector string `json:"collector"`
	Date      string `json:"date"`
	EnergyWh  int    `json:"energy_wh"`
}

func (srv *server) ingestSample(w http.ResponseWriter, r *http.Request) {
	var req sampleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Collector == "" || req.TS == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := srv.ing.Sample(req.Collector, req.TS, req.PowerW); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (srv *server) ingestDay(w http.ResponseWriter, r *http.Request) {
	var req dayReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Collector == "" || req.Date == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := srv.ing.Day(req.Collector, req.Date, req.EnergyWh); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (srv *server) daymax(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	date := q.Get("date")
	if !isDateDigits(date) {
		http.Error(w, "bad date", http.StatusBadRequest)
		return
	}
	days := 7
	if v, err := strconv.Atoi(q.Get("days")); err == nil && v > 0 && v <= 31 {
		days = v
	}
	m, err := srv.s.SlotMax(date, q.Get("collector"), days)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, m)
}
