package timeutil

import "testing"

func TestDayRange(t *testing.T) {
	start, end, err := DayRange("20250601")
	if err != nil {
		t.Fatal(err)
	}
	if end-start != 24*3600 {
		t.Fatalf("range = %d s, want 86400", end-start)
	}
	// 2025-06-01 00:00 Brussels = CEST (UTC+2) = 1748728800
	if start != 1748728800 {
		t.Fatalf("start = %d, want 1748728800", start)
	}
}

func TestRound5m(t *testing.T) {
	if got := Round5m(1748728801); got != 1748728800 {
		t.Fatalf("round down = %d, want 1748728800", got)
	}
	if got := Round5m(1748728950); got != 1748729100 {
		t.Fatalf("round up = %d, want 1748729100", got)
	}
}

func TestSlotTS(t *testing.T) {
	// dst day: startuur = 05:00 lokaal, slot 0
	ts, err := SlotTS("20250601", true, 0)
	if err != nil {
		t.Fatal(err)
	}
	// 2025-06-01 05:00 CEST = 1748746800
	if ts != 1748746800 {
		t.Fatalf("slot0 = %d, want 1748746800", ts)
	}
	ts12, _ := SlotTS("20250601", true, 12) // +1h
	if ts12-ts != 3600 {
		t.Fatalf("slot12-slot0 = %d, want 3600", ts12-ts)
	}
}
