package main

import (
	"reflect"
	"testing"

	"pvdiary/internal/api"
)

func TestParseStrings(t *testing.T) {
	got, err := parseStrings("ZO:5760:130:40:-0.38, NW:3200:310:40:-0.38")
	if err != nil {
		t.Fatal(err)
	}
	want := []api.StringCfg{
		{Name: "ZO", Wp: 5760, Az: 130, Tilt: 40, TempCoeff: -0.38},
		{Name: "NW", Wp: 3200, Az: 310, Tilt: 40, TempCoeff: -0.38},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseStringsErrors(t *testing.T) {
	for _, v := range []string{"", "ZO:5760:130:40", "ZO:x:130:40:-0.38", "ZO:5760:zuid:40:-0.38"} {
		if _, err := parseStrings(v); err == nil {
			t.Errorf("parseStrings(%q): want error", v)
		}
	}
}

func TestParseCoord(t *testing.T) {
	if f, err := parseCoord("PVDIARY_LAT", "50.8467", 90); err != nil || f != 50.8467 {
		t.Fatalf("got %v, %v", f, err)
	}
	for _, v := range []string{"", "abc", "91", "-90.5"} {
		if _, err := parseCoord("PVDIARY_LAT", v, 90); err == nil {
			t.Errorf("parseCoord(%q): want error", v)
		}
	}
}

func TestSiteFromEnv(t *testing.T) {
	env := map[string]string{
		"PVDIARY_LAT":     "50.8467",
		"PVDIARY_LON":     "4.3525",
		"PVDIARY_STRINGS": "ZO:5760:130:40:-0.38,NW:3200:310:40:-0.38",
	}
	s, err := siteFromEnv(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if s.Lat != 50.8467 || s.Lon != 4.3525 || len(s.Strings) != 2 || s.PVWatt() != 8960 {
		t.Fatalf("got %+v, pvwatt %d", s, s.PVWatt())
	}
	for _, k := range []string{"PVDIARY_LAT", "PVDIARY_LON", "PVDIARY_STRINGS"} {
		missing := map[string]string{}
		for kk, v := range env {
			if kk != k {
				missing[kk] = v
			}
		}
		if _, err := siteFromEnv(func(k string) string { return missing[k] }); err == nil {
			t.Errorf("without %s: want error", k)
		}
	}
}
