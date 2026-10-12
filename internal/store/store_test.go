package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestUpsertCollector(t *testing.T) {
	s := newTestStore(t)
	id, err := s.UpsertCollector(Collector{Key: "foxess", Name: "FoxESS", PVWatt: 8960, Day1: "20260701"})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("id == 0")
	}
	// upsert again -> same id, updated name
	id2, err := s.UpsertCollector(Collector{Key: "foxess", Name: "FoxESS CQ7", PVWatt: 8960})
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatalf("id changed %d -> %d", id, id2)
	}
	c, err := s.CollectorByKey("foxess")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "FoxESS CQ7" || c.PVWatt != 8960 {
		t.Fatalf("got %+v", c)
	}
}

func TestUpsertDayAndSampleIdempotent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.UpsertCollector(Collector{Key: "c", Name: "c"})

	if err := s.UpsertDay(id, "20250601", 48687, 5004); err != nil {
		t.Fatal(err)
	}
	// tweede keer overschrijft, geen duplicaat
	if err := s.UpsertDay(id, "20250601", 48690, 5010); err != nil {
		t.Fatal(err)
	}
	var n, energy int
	s.db.QueryRow(`SELECT count(*), max(energy_wh) FROM day WHERE collector_id=?`, id).Scan(&n, &energy)
	if n != 1 || energy != 48690 {
		t.Fatalf("n=%d energy=%d", n, energy)
	}

	if err := s.UpsertSamples(id, []Sample{{1000, 100}, {1300, 200}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSample(id, 1300, 250); err != nil { // overschrijf
		t.Fatal(err)
	}
	var cnt, p int
	s.db.QueryRow(`SELECT count(*), max(power_w) FROM sample WHERE collector_id=?`, id).Scan(&cnt, &p)
	if cnt != 2 || p != 250 {
		t.Fatalf("cnt=%d p=%d", cnt, p)
	}
}
