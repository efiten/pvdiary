package store

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type Collector struct {
	ID       int64
	Key      string
	Name     string
	Lat      float64
	Lon      float64
	Location string
	PVWatt   int
	Day1     string
}

func (s *Store) UpsertCollector(c Collector) (int64, error) {
	_, err := s.db.Exec(`
		INSERT INTO collector (key,name,latitude,longitude,location,pv_watt,day1)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(key) DO UPDATE SET
			name=excluded.name, latitude=excluded.latitude, longitude=excluded.longitude,
			location=excluded.location, pv_watt=excluded.pv_watt, day1=excluded.day1`,
		c.Key, c.Name, c.Lat, c.Lon, c.Location, c.PVWatt, c.Day1)
	if err != nil {
		return 0, err
	}
	got, err := s.CollectorByKey(c.Key)
	if err != nil {
		return 0, err
	}
	return got.ID, nil
}

func (s *Store) CollectorByKey(key string) (Collector, error) {
	var c Collector
	err := s.db.QueryRow(`SELECT id,key,name,latitude,longitude,location,pv_watt,day1
		FROM collector WHERE key=?`, key).
		Scan(&c.ID, &c.Key, &c.Name, &c.Lat, &c.Lon, &c.Location, &c.PVWatt, &c.Day1)
	return c, err
}

func (s *Store) Collectors() ([]Collector, error) {
	rows, err := s.db.Query(`SELECT id,key,name,latitude,longitude,location,pv_watt,day1
		FROM collector ORDER BY day1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collector
	for rows.Next() {
		var c Collector
		if err := rows.Scan(&c.ID, &c.Key, &c.Name, &c.Lat, &c.Lon, &c.Location, &c.PVWatt, &c.Day1); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type Sample struct {
	TS     int64
	PowerW int
}

func (s *Store) UpsertDay(collectorID int64, date string, energyWh, peakW int) error {
	_, err := s.db.Exec(`
		INSERT INTO day (collector_id,date,energy_wh,peak_w) VALUES (?,?,?,?)
		ON CONFLICT(collector_id,date) DO UPDATE SET energy_wh=excluded.energy_wh, peak_w=excluded.peak_w`,
		collectorID, date, energyWh, peakW)
	return err
}

func (s *Store) UpsertSample(collectorID int64, ts int64, powerW int) error {
	_, err := s.db.Exec(`
		INSERT INTO sample (collector_id,ts,power_w) VALUES (?,?,?)
		ON CONFLICT(collector_id,ts) DO UPDATE SET power_w=excluded.power_w`,
		collectorID, ts, powerW)
	return err
}

func (s *Store) UpsertSamples(collectorID int64, samples []Sample) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
		INSERT INTO sample (collector_id,ts,power_w) VALUES (?,?,?)
		ON CONFLICT(collector_id,ts) DO UPDATE SET power_w=excluded.power_w`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, sm := range samples {
		if _, err := stmt.Exec(collectorID, sm.TS, sm.PowerW); err != nil {
			return err
		}
	}
	return tx.Commit()
}
