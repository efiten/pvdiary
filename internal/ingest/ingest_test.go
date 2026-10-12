package ingest

import (
	"path/filepath"
	"testing"

	"pvdiary/internal/store"
	"pvdiary/internal/timeutil"
)

func TestSampleRoundsAndTracksPeak(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "i.db"))
	defer s.Close()
	svc := New(s)

	// 2025-06-01 12:00:07 CEST -> rondt af op 12:00:00
	base, _ := timeutil.SlotTS("20250601", true, 84) // 05:00 + 84*5m = 12:00
	if err := svc.Sample("foxess", base+7, 4000); err != nil {
		t.Fatal(err)
	}
	if err := svc.Sample("foxess", base+300+2, 4500); err != nil {
		t.Fatal(err)
	}
	// dubbele post zelfde slot -> upsert, geen tweede rij
	if err := svc.Sample("foxess", base+4, 3900); err != nil {
		t.Fatal(err)
	}
	sm, _ := s.DaySamples("20250601", "foxess")
	if len(sm) != 2 {
		t.Fatalf("samples = %d, want 2", len(sm))
	}
	di, _ := s.DayInfo("20250601", "foxess")
	if di.PeakW != 4500 {
		t.Fatalf("peak = %d, want 4500", di.PeakW)
	}
	// Energy is only computed as sum/12 when the day row is first created
	// (by the very first Sample of the day: single sample 4000 -> (4000+6)/12=333).
	// Later Sample() posts must NOT recompute energy from the growing sample
	// set -- they only keep peak_w up to date; energy_wh stays authoritative
	// via Day() posts (see recomputeDay).
	if di.EnergyWh != 333 {
		t.Fatalf("energy = %d, want 333", di.EnergyWh)
	}
}

func TestDaySetsTotal(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "d.db"))
	defer s.Close()
	svc := New(s)
	if err := svc.Sample("foxess", mustSlot(t, "20250601", 84), 5000); err != nil {
		t.Fatal(err)
	}
	if err := svc.Day("foxess", "20250601", 42000); err != nil {
		t.Fatal(err)
	}
	di, _ := s.DayInfo("20250601", "foxess")
	if di.EnergyWh != 42000 || di.PeakW != 5000 {
		t.Fatalf("dayinfo = %+v", di)
	}
}

func TestDayBeforeSample(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "d2.db"))
	defer s.Close()
	svc := New(s)

	// Call Day() before any Sample() for that date
	if err := svc.Day("foxess", "20250601", 12345); err != nil {
		t.Fatal(err)
	}
	di, _ := s.DayInfo("20250601", "foxess")
	if di.EnergyWh != 12345 {
		t.Fatalf("energy = %d, want 12345", di.EnergyWh)
	}
	if di.PeakW != 0 {
		t.Fatalf("peak = %d, want 0", di.PeakW)
	}
}

func TestSampleAfterDayPreservesCounterEnergy(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "d3.db"))
	defer s.Close()
	svc := New(s)

	// Day() posts the counter-set energy first, creating the day row.
	if err := svc.Day("foxess", "20250601", 42000); err != nil {
		t.Fatal(err)
	}
	// A Sample() post for the same date must not overwrite energy_wh, only peak_w.
	if err := svc.Sample("foxess", mustSlot(t, "20250601", 84), 5000); err != nil {
		t.Fatal(err)
	}
	di, _ := s.DayInfo("20250601", "foxess")
	if di.EnergyWh != 42000 {
		t.Fatalf("energy = %d, want 42000 (preserved from Day())", di.EnergyWh)
	}
	if di.PeakW != 5000 {
		t.Fatalf("peak = %d, want 5000 (updated by Sample())", di.PeakW)
	}
}

func TestSampleBeforeDayUsesSumOverTwelve(t *testing.T) {
	s, _ := store.Open(filepath.Join(t.TempDir(), "d4.db"))
	defer s.Close()
	svc := New(s)

	// No Day() post yet: recomputeDay creates the row from sum/12 of samples.
	if err := svc.Sample("foxess", mustSlot(t, "20250601", 84), 6000); err != nil {
		t.Fatal(err)
	}
	di, _ := s.DayInfo("20250601", "foxess")
	if di.EnergyWh != 500 { // (6000+6)/12 = 500
		t.Fatalf("energy = %d, want 500", di.EnergyWh)
	}
	if di.PeakW != 6000 {
		t.Fatalf("peak = %d, want 6000", di.PeakW)
	}
}

func mustSlot(t *testing.T, d string, n int) int64 {
	ts, err := timeutil.SlotTS(d, true, n)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
