package store

import (
	"testing"

	"pvdiary/internal/timeutil"
)

func seed(t *testing.T, s *Store) int64 {
	t.Helper()
	id, _ := s.UpsertCollector(Collector{Key: "c", Name: "c", PVWatt: 8960})
	// twee dagen in juni 2025, één in mei
	s.UpsertDay(id, "20250601", 48687, 5004)
	s.UpsertDay(id, "20250611", 40000, 4900)
	s.UpsertDay(id, "20250515", 30000, 4800)
	s.UpsertSamples(id, []Sample{{1000, 100}, {1300, 250}, {1600, 50}})
	return id
}

func TestDayAndMonth(t *testing.T) {
	s := newTestStore(t)
	seed(t, s)
	m, err := s.Month("202506", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("month days = %d, want 2", len(m))
	}
	if m[0].Date != "20250601" || m[0].EnergyWh != 48687 {
		t.Fatalf("got %+v", m[0])
	}
}

func TestMonthEmptyReturnsNonNilEmptySlice(t *testing.T) {
	s := newTestStore(t)
	seed(t, s)
	m, err := s.Month("209901", "")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("Month() returned nil, want non-nil empty slice")
	}
	if len(m) != 0 {
		t.Fatalf("len(m) = %d, want 0", len(m))
	}
}

func TestTopDaysEmptyReturnsNonNilEmptySlice(t *testing.T) {
	s := newTestStore(t)
	seed(t, s)
	top, err := s.TopDays("year", "1899", 10)
	if err != nil {
		t.Fatal(err)
	}
	if top == nil {
		t.Fatal("TopDays() returned nil, want non-nil empty slice")
	}
	if len(top) != 0 {
		t.Fatalf("len(top) = %d, want 0", len(top))
	}
}

func TestYearMonthsAndTop(t *testing.T) {
	s := newTestStore(t)
	seed(t, s)
	ym, err := s.YearMonths(2025, "")
	if err != nil {
		t.Fatal(err)
	}
	if ym[4] != 30000 { // mei = index 4
		t.Fatalf("may = %d, want 30000", ym[4])
	}
	if ym[5] != 88687 { // juni = 48687+40000
		t.Fatalf("june = %d, want 88687", ym[5])
	}
	top, err := s.TopDays("global", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 || top[0].Date != "20250601" || top[0].Value != 48687 {
		t.Fatalf("top = %+v", top)
	}
	tp, err := s.TopPeak("global", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if tp[0].Date != "20250601" || tp[0].Value != 5004 {
		t.Fatalf("toppeak = %+v", tp)
	}
	last, _ := s.LastDate("")
	if last != "20250611" {
		t.Fatalf("last = %s", last)
	}
}

func TestDaySamples(t *testing.T) {
	s := newTestStore(t)
	id1, err := s.UpsertCollector(Collector{Key: "c1", Name: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.UpsertCollector(Collector{Key: "c2", Name: "c2"})
	if err != nil {
		t.Fatal(err)
	}

	const date = "20250601" // juni -> DST actief in Brussel
	ts0, err := timeutil.SlotTS(date, true, 0)
	if err != nil {
		t.Fatal(err)
	}
	ts1, err := timeutil.SlotTS(date, true, 1)
	if err != nil {
		t.Fatal(err)
	}
	start, end, err := timeutil.DayRange(date)
	if err != nil {
		t.Fatal(err)
	}

	// binnen de dag: twee collectors op hetzelfde ts0, en één op ts1
	if err := s.UpsertSamples(id1, []Sample{{ts0, 100}, {ts1, 200}}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSamples(id2, []Sample{{ts0, 50}}); err != nil {
		t.Fatal(err)
	}
	// buiten de dag: net vóór start, en op end (exclusief) -> moeten genegeerd worden
	if err := s.UpsertSamples(id1, []Sample{{start - 1, 999}, {end, 888}}); err != nil {
		t.Fatal(err)
	}

	// alle collectors samen: som per ts, oplopend geordend, buitendag-samples genegeerd
	all, err := s.DaySamples(date, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("all = %+v, want 2 samples", all)
	}
	if all[0].TS != ts0 || all[0].PowerW != 150 {
		t.Fatalf("all[0] = %+v, want ts=%d power=150", all[0], ts0)
	}
	if all[1].TS != ts1 || all[1].PowerW != 200 {
		t.Fatalf("all[1] = %+v, want ts=%d power=200", all[1], ts1)
	}

	// per collector: alleen eigen waarden
	c1, err := s.DaySamples(date, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(c1) != 2 || c1[0].PowerW != 100 || c1[1].PowerW != 200 {
		t.Fatalf("c1 = %+v", c1)
	}
	c2, err := s.DaySamples(date, "c2")
	if err != nil {
		t.Fatal(err)
	}
	if len(c2) != 1 || c2[0].TS != ts0 || c2[0].PowerW != 50 {
		t.Fatalf("c2 = %+v", c2)
	}
}

func TestDayInfo(t *testing.T) {
	s := newTestStore(t)
	id1, _ := s.UpsertCollector(Collector{Key: "c1", Name: "c1"})
	id2, _ := s.UpsertCollector(Collector{Key: "c2", Name: "c2"})

	if err := s.UpsertDay(id1, "20250601", 1000, 300); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertDay(id2, "20250601", 2000, 150); err != nil {
		t.Fatal(err)
	}

	// alle collectors: som van energie, max van piek
	all, err := s.DayInfo("20250601", "")
	if err != nil {
		t.Fatal(err)
	}
	if all != (DayTotal{Date: "20250601", EnergyWh: 3000, PeakW: 300}) {
		t.Fatalf("all = %+v", all)
	}

	// enkele collector
	c1, err := s.DayInfo("20250601", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if c1 != (DayTotal{Date: "20250601", EnergyWh: 1000, PeakW: 300}) {
		t.Fatalf("c1 = %+v", c1)
	}

	// niet-bestaande datum: NULL-aggregaat -> nulwaarden, geen fout
	absent, err := s.DayInfo("20250602", "")
	if err != nil {
		t.Fatal(err)
	}
	if absent != (DayTotal{Date: "20250602", EnergyWh: 0, PeakW: 0}) {
		t.Fatalf("absent = %+v", absent)
	}
}

func TestDaySamplesEmptyReturnsNonNilEmptySlice(t *testing.T) {
	s := newTestStore(t)
	seed(t, s)
	sm, err := s.DaySamples("20991231", "")
	if err != nil {
		t.Fatal(err)
	}
	if sm == nil {
		t.Fatal("DaySamples() returned nil, want non-nil empty slice")
	}
	if len(sm) != 0 {
		t.Fatalf("len(sm) = %d, want 0", len(sm))
	}
}

func TestKWhYear(t *testing.T) {
	s := newTestStore(t)
	id1, _ := s.UpsertCollector(Collector{Key: "c1", Name: "c1"})
	id2, _ := s.UpsertCollector(Collector{Key: "c2", Name: "c2"})

	// 2025: juni -> beide collectors, mei -> alleen c1
	s.UpsertDay(id1, "20250601", 1500, 300)
	s.UpsertDay(id2, "20250611", 500, 200)
	s.UpsertDay(id1, "20250515", 3000, 400)
	// 2024: mei -> alleen c1
	s.UpsertDay(id1, "20240520", 2500, 250)

	all, err := s.KWhYear("")
	if err != nil {
		t.Fatal(err)
	}
	y2025 := all[2025]
	if y2025[4] != 3.0 { // mei = index 4
		t.Fatalf("2025 mei = %v, want 3.0", y2025[4])
	}
	if y2025[5] != 2.0 { // juni = (1500+500)/1000
		t.Fatalf("2025 juni = %v, want 2.0", y2025[5])
	}
	y2024 := all[2024]
	if y2024[4] != 2.5 {
		t.Fatalf("2024 mei = %v, want 2.5", y2024[4])
	}

	// per collector: enkel eigen dagen
	c2, err := s.KWhYear("c2")
	if err != nil {
		t.Fatal(err)
	}
	if c2[2025][5] != 0.5 {
		t.Fatalf("c2 2025 juni = %v, want 0.5", c2[2025][5])
	}
	if _, ok := c2[2024]; ok {
		t.Fatalf("c2 heeft geen 2024-dagen, kreeg %+v", c2[2024])
	}
}

func TestLastUpdate(t *testing.T) {
	s := newTestStore(t)
	empty, err := s.LastUpdate("")
	if err != nil || empty != 0 {
		t.Fatalf("empty db: ts=%d err=%v, want 0/nil", empty, err)
	}
	id, _ := s.UpsertCollector(Collector{Key: "c", Name: "c"})
	s.UpsertSamples(id, []Sample{{1749610800, 100}, {1749611100, 200}})
	ts, err := s.LastUpdate("")
	if err != nil || ts != 1749611100 {
		t.Fatalf("ts=%d err=%v, want 1749611100/nil", ts, err)
	}
}
