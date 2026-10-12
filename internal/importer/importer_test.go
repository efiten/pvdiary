package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pvdiary/internal/store"
)

func TestRunImportsAndComputesTotals(t *testing.T) {
	dir := t.TempDir()
	// mini-maandbestand: 1 dag, idx0=pv_watt, idx1=dst, dan 3 samples 5-min
	// sum=100+200+50=350 W -> Wh = round(350/12)=29 ; peak=200
	month := map[string]map[string][]int{
		"C": {"20250601": {8960, 1, 100, 200, 50}},
	}
	b, _ := json.Marshal(month)
	os.WriteFile(filepath.Join(dir, "202506.json"), b, 0o644)

	s, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rep, err := Run(s, dir, store.Collector{Key: "C", Name: "C", PVWatt: 8960})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Days != 1 || rep.Samples != 3 {
		t.Fatalf("report = %+v", rep)
	}
	di, _ := s.DayInfo("20250601", "C")
	if di.EnergyWh != 29 || di.PeakW != 200 {
		t.Fatalf("dayinfo = %+v (want energy 29 peak 200)", di)
	}
	c, _ := s.CollectorByKey("C")
	if c.Day1 != "20250601" {
		t.Fatalf("day1 = %s", c.Day1)
	}
}

func TestRunAgainstRealDataValidatesTotals(t *testing.T) {
	real := "../../weerstation/data"
	if _, err := os.Stat(filepath.Join(real, "summary.json")); err != nil {
		t.Skip("echte dataset niet aanwezig")
	}
	s, _ := store.Open(filepath.Join(t.TempDir(), "r.db"))
	defer s.Close()
	rep, err := Run(s, real, store.Collector{Key: "SMA50", Name: "SMA", PVWatt: 8960})
	if err != nil {
		t.Fatal(err)
	}
	// summary.json kwh_year 2025 juni = 1133.9
	ym, _ := s.YearMonths(2025, "SMA50")
	june := float64(ym[5]) / 1000.0
	if diff := june - 1133.9; diff > 0.2 || diff < -0.2 {
		t.Fatalf("juni 2025 = %.1f kWh, want ~1133.9", june)
	}
	// top-dag 2025 = 20250611 @ 48687 Wh
	top, _ := s.TopDays("year", "2025", 1)
	if top[0].Date != "20250611" || abs(top[0].Value-48687) > 5 {
		t.Fatalf("top day 2025 = %+v, want 20250611/48687", top[0])
	}
	// pre-operational placeholder months (1970-1997, 2020) must be dropped
	c, _ := s.CollectorByKey("SMA50")
	if c.Day1 != "20210101" {
		t.Fatalf("day1 = %s, want 20210101", c.Day1)
	}
	for y := range rep.YearKWh {
		if y < 2021 {
			t.Fatalf("YearKWh has placeholder year %d: %v", y, rep.YearKWh)
		}
	}
	t.Logf("real data import: days=%d samples=%d day1=%s", rep.Days, rep.Samples, c.Day1)
}

func TestRunSkipsZeroDaysBeforeFirstNonZeroDay(t *testing.T) {
	dir := t.TempDir()
	// januari: dag 1 = placeholder (all zero samples), dag 2 = eerste echte dag (niet-nul)
	jan := map[string]map[string][]int{
		"C": {
			"20210101": {8960, 0, 0, 0, 0},
			"20210102": {8960, 0, 10, 20, 30},
		},
	}
	b, _ := json.Marshal(jan)
	os.WriteFile(filepath.Join(dir, "202101.json"), b, 0o644)

	// februari: dag met sum=0 NA de eerste niet-nul dag moet WEL bewaard blijven
	// (echte operationele nul-dag, bv. bewolkt/sneeuw), plus nog een niet-nul dag.
	feb := map[string]map[string][]int{
		"C": {
			"20210201": {8960, 0, 0, 0, 0},
			"20210202": {8960, 0, 5, 5, 5},
		},
	}
	b, _ = json.Marshal(feb)
	os.WriteFile(filepath.Join(dir, "202102.json"), b, 0o644)

	s, err := store.Open(filepath.Join(t.TempDir(), "t2.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rep, err := Run(s, dir, store.Collector{Key: "C", Name: "C", PVWatt: 8960})
	if err != nil {
		t.Fatal(err)
	}
	// verwacht: 20210101 (placeholder, sum=0) wordt gedropt; de overige 3 dagen
	// (20210102, 20210201, 20210202) blijven, inclusief de operationele nul-dag 20210201.
	if rep.Days != 3 {
		t.Fatalf("rep.Days = %d, want 3", rep.Days)
	}
	c, _ := s.CollectorByKey("C")
	if c.Day1 != "20210102" {
		t.Fatalf("day1 = %s, want 20210102", c.Day1)
	}
	janDays, _ := s.Month("202101", "C")
	for _, d := range janDays {
		if d.Date == "20210101" {
			t.Fatalf("placeholder day 20210101 should not have been imported: %+v", janDays)
		}
	}
	febDays, _ := s.Month("202102", "C")
	found := false
	for _, d := range febDays {
		if d.Date == "20210201" {
			found = true
			if d.EnergyWh != 0 {
				t.Fatalf("operational zero day 20210201 energy = %d, want 0", d.EnergyWh)
			}
		}
	}
	if !found {
		t.Fatalf("operational zero day 20210201 should be kept: %+v", febDays)
	}
}

func abs(x int) int { if x < 0 { return -x }; return x }
