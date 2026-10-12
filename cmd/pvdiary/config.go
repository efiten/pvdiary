package main

import (
	"fmt"
	"strconv"
	"strings"

	"pvdiary/internal/api"
)

// parseStrings leest PVDIARY_STRINGS: per string "naam:wp:azimut:helling:tempcoeff",
// strings gescheiden door komma's. Bv. "ZO:5760:130:40:-0.38,NW:3200:310:40:-0.38".
func parseStrings(v string) ([]api.StringCfg, error) {
	var out []api.StringCfg
	for _, part := range strings.Split(v, ",") {
		f := strings.Split(strings.TrimSpace(part), ":")
		if len(f) != 5 {
			return nil, fmt.Errorf("string %q: want naam:wp:azimut:helling:tempcoeff", part)
		}
		wp, err := strconv.Atoi(f[1])
		if err != nil {
			return nil, fmt.Errorf("string %q: wp: %v", part, err)
		}
		var nums [3]float64
		for i, s := range f[2:] {
			if nums[i], err = strconv.ParseFloat(s, 64); err != nil {
				return nil, fmt.Errorf("string %q: %v", part, err)
			}
		}
		out = append(out, api.StringCfg{Name: f[0], Wp: wp, Az: nums[0], Tilt: nums[1], TempCoeff: nums[2]})
	}
	return out, nil
}

// parseCoord leest een verplichte coördinaat binnen [-max, max].
func parseCoord(name, v string, max float64) (float64, error) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %v", name, v, err)
	}
	if f < -max || f > max {
		return 0, fmt.Errorf("%s=%v: out of range", name, f)
	}
	return f, nil
}

type site struct {
	Lat, Lon float64
	Strings  []api.StringCfg
}

// PVWatt is het totale piekvermogen over alle strings.
func (s site) PVWatt() int {
	total := 0
	for _, st := range s.Strings {
		total += st.Wp
	}
	return total
}

// siteFromEnv leest de verplichte PVDIARY_LAT, PVDIARY_LON en PVDIARY_STRINGS.
func siteFromEnv(getenv func(string) string) (site, error) {
	lat, err := parseCoord("PVDIARY_LAT", getenv("PVDIARY_LAT"), 90)
	if err != nil {
		return site{}, err
	}
	lon, err := parseCoord("PVDIARY_LON", getenv("PVDIARY_LON"), 180)
	if err != nil {
		return site{}, err
	}
	strs, err := parseStrings(getenv("PVDIARY_STRINGS"))
	if err != nil {
		return site{}, fmt.Errorf("PVDIARY_STRINGS: %v", err)
	}
	return site{Lat: lat, Lon: lon, Strings: strs}, nil
}
