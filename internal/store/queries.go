package store

import (
	"database/sql"
	"fmt"
	"time"

	"pvdiary/internal/timeutil"
)

type DayTotal struct {
	Date     string
	EnergyWh int
	PeakW    int
}

type TopEntry struct {
	Date  string
	Value int
}

// collFilter geeft een SQL-fragment + arg voor optionele collector-filter.
func collFilter(collectorKey string) (string, []any) {
	if collectorKey == "" {
		return "", nil
	}
	return " AND collector_id = (SELECT id FROM collector WHERE key=?)", []any{collectorKey}
}

func (s *Store) DaySamples(date, collectorKey string) ([]Sample, error) {
	start, end, err := timeutil.DayRange(date)
	if err != nil {
		return nil, err
	}
	filt, args := collFilter(collectorKey)
	q := `SELECT ts, SUM(power_w) FROM sample WHERE ts >= ? AND ts < ?` + filt +
		` GROUP BY ts ORDER BY ts`
	rows, err := s.db.Query(q, append([]any{start, end}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Sample{}
	for rows.Next() {
		var sm Sample
		if err := rows.Scan(&sm.TS, &sm.PowerW); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

func (s *Store) DayInfo(date, collectorKey string) (DayTotal, error) {
	filt, args := collFilter(collectorKey)
	q := `SELECT SUM(energy_wh), MAX(peak_w) FROM day WHERE date=?` + filt
	var e, p *int
	err := s.db.QueryRow(q, append([]any{date}, args...)...).Scan(&e, &p)
	if err != nil {
		return DayTotal{}, err
	}
	d := DayTotal{Date: date}
	if e != nil {
		d.EnergyWh = *e
	}
	if p != nil {
		d.PeakW = *p
	}
	return d, nil
}

// DayExists geeft aan of er al een day-rij bestaat voor deze collector+datum.
func (s *Store) DayExists(collectorID int64, date string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM day WHERE collector_id=? AND date=?`, collectorID, date).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Month(ym, collectorKey string) ([]DayTotal, error) {
	filt, args := collFilter(collectorKey)
	q := `SELECT date, SUM(energy_wh), MAX(peak_w) FROM day WHERE date LIKE ?` + filt +
		` GROUP BY date ORDER BY date`
	rows, err := s.db.Query(q, append([]any{ym + "%"}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DayTotal{}
	for rows.Next() {
		var d DayTotal
		if err := rows.Scan(&d.Date, &d.EnergyWh, &d.PeakW); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) YearMonths(year int, collectorKey string) ([12]int, error) {
	var out [12]int
	filt, args := collFilter(collectorKey)
	q := `SELECT CAST(substr(date,5,2) AS INT) m, SUM(energy_wh)
		FROM day WHERE date LIKE ?` + filt + ` GROUP BY m`
	rows, err := s.db.Query(q, append([]any{fmt.Sprintf("%04d%%", year)}, args...)...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m, wh int
		if err := rows.Scan(&m, &wh); err != nil {
			return out, err
		}
		if m >= 1 && m <= 12 {
			out[m-1] = wh
		}
	}
	return out, rows.Err()
}

func (s *Store) KWhYear(collectorKey string) (map[int][12]float64, error) {
	filt, args := collFilter(collectorKey)
	q := `SELECT CAST(substr(date,1,4) AS INT) y, CAST(substr(date,5,2) AS INT) m, SUM(energy_wh)
		FROM day WHERE 1=1` + filt + ` GROUP BY y,m ORDER BY y,m`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int][12]float64{}
	for rows.Next() {
		var y, m, wh int
		if err := rows.Scan(&y, &m, &wh); err != nil {
			return nil, err
		}
		arr := out[y]
		if m >= 1 && m <= 12 {
			arr[m-1] = float64(wh) / 1000.0
		}
		out[y] = arr
	}
	return out, rows.Err()
}

// topLike bouwt de LIKE-parameter voor scope/key.
func topLike(scope, key string) (string, error) {
	switch scope {
	case "global":
		return "%", nil
	case "year":
		return key + "%", nil // key = "2025"
	case "month":
		return "____" + key + "%", nil // key = "06" -> matcht YYYY06DD
	default:
		return "", fmt.Errorf("bad scope %q", scope)
	}
}

func (s *Store) topBy(col, scope, key string, n int) ([]TopEntry, error) {
	like, err := topLike(scope, key)
	if err != nil {
		return nil, err
	}
	// som energie per datum, resp. max piek per datum (collectors tijd-disjunct)
	agg := "SUM(energy_wh)"
	if col == "peak" {
		agg = "MAX(peak_w)"
	}
	q := `SELECT date, ` + agg + ` v FROM day WHERE date LIKE ? GROUP BY date ORDER BY v DESC LIMIT ?`
	rows, err := s.db.Query(q, like, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopEntry{}
	for rows.Next() {
		var e TopEntry
		if err := rows.Scan(&e.Date, &e.Value); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) TopDays(scope, key string, n int) ([]TopEntry, error) {
	return s.topBy("energy", scope, key, n)
}

func (s *Store) TopPeak(scope, key string, n int) ([]TopEntry, error) {
	return s.topBy("peak", scope, key, n)
}

func (s *Store) LastDate(collectorKey string) (string, error) {
	filt, args := collFilter(collectorKey)
	q := `SELECT COALESCE(MAX(date),'') FROM day WHERE 1=1` + filt
	var d string
	err := s.db.QueryRow(q, args...).Scan(&d)
	return d, err
}

// LastUpdate geeft de unix-ts van het nieuwste sample (0 als er geen zijn).
func (s *Store) LastUpdate(collectorKey string) (int64, error) {
	filt, args := collFilter(collectorKey)
	q := `SELECT COALESCE(MAX(ts),0) FROM sample WHERE 1=1` + filt
	var ts int64
	err := s.db.QueryRow(q, args...).Scan(&ts)
	return ts, err
}

// SlotMax geeft per 5-min-slot (lokale "HH:MM") het maximum vermogen over de
// dagen in [date-days, date+days]. Bron voor de ±N-dagen-max achtergrond.
func (s *Store) SlotMax(date, collectorKey string, days int) (map[string]int, error) {
	t, err := time.ParseInLocation("20060102", date, timeutil.Loc)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for off := -days; off <= days; off++ {
		d := t.AddDate(0, 0, off).Format("20060102")
		samples, err := s.DaySamples(d, collectorKey)
		if err != nil {
			return nil, err
		}
		for _, sm := range samples {
			label := time.Unix(sm.TS, 0).In(timeutil.Loc).Format("15:04")
			if sm.PowerW > out[label] {
				out[label] = sm.PowerW
			}
		}
	}
	return out, nil
}
