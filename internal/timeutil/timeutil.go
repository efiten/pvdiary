package timeutil

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

// Loc is de vaste projectstijdzone. PVdiary werkt volledig in lokale tijd.
var Loc = mustLoad("Europe/Brussels")

func mustLoad(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

func parseDate(date string) (time.Time, error) {
	t, err := time.ParseInLocation("20060102", date, Loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("bad date %q: %w", date, err)
	}
	return t, nil
}

// DayRange geeft [start,end) unix seconds voor de lokale dag YYYYMMDD.
func DayRange(date string) (int64, int64, error) {
	t, err := parseDate(date)
	if err != nil {
		return 0, 0, err
	}
	start := t.Unix()
	end := t.AddDate(0, 0, 1).Unix()
	return start, end, nil
}

// Round5m rondt af op het dichtstbijzijnde 5-min raster (300 s).
func Round5m(ts int64) int64 {
	return (ts + 150) / 300 * 300
}

// SlotTS geeft de unix-ts van intraday-slot n (5-min vanaf 04:00 lokaal;
// dst schuift het startuur 1u op naar 05:00).
func SlotTS(date string, dst bool, n int) (int64, error) {
	t, err := parseDate(date)
	if err != nil {
		return 0, err
	}
	startHour := 4
	if dst {
		startHour = 5
	}
	base := time.Date(t.Year(), t.Month(), t.Day(), startHour, 0, 0, 0, Loc)
	return base.Add(time.Duration(n) * 5 * time.Minute).Unix(), nil
}
