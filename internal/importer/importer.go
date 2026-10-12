package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"pvdiary/internal/store"
	"pvdiary/internal/timeutil"
)

type Report struct {
	Days    int
	Samples int
	YearKWh map[int]float64
}

var monthFile = regexp.MustCompile(`^\d{6}\.json$`)

// Run importeert alle YYYYMM.json uit dataDir onder de collector uit meta.
func Run(s *store.Store, dataDir string, meta store.Collector) (Report, error) {
	rep := Report{YearKWh: map[int]float64{}}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return rep, err
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && monthFile.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	id, err := s.UpsertCollector(meta)
	if err != nil {
		return rep, err
	}

	// dayRecord holds one parsed day, kept in memory until we know the
	// cutoff date (earliest day with a non-zero sample-sum).
	type dayRecord struct {
		date     string
		dst      bool
		powers   []int
		sum      int
		peak     int
		energyWh int
	}
	days := make([]dayRecord, 0, 12*366)

	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			return rep, err
		}
		// { "<coll>": { "YYYYMMDD": [pv_watt, dst, p0, p1, ...] } }
		var doc map[string]map[string][]int
		umErr := json.Unmarshal(raw, &doc)

		// Some legacy files have corrupted JSON - handle common cases
		if umErr != nil {
			// Case 1: File ends with ]} - missing final closing brace
			if len(raw) >= 2 && raw[len(raw)-1] == '}' && raw[len(raw)-2] == ']' {
				raw = append(raw, '}')
				umErr = json.Unmarshal(raw, &doc)
			}
			// Case 2: File has extra closing brace - try removing it
			if umErr != nil && len(raw) > 3 && raw[len(raw)-1] == '}' && raw[len(raw)-2] == '}' && raw[len(raw)-3] == '}' {
				umErr = json.Unmarshal(raw[:len(raw)-1], &doc)
			}
		}

		if umErr != nil {
			return rep, umErr
		}
		for _, dayMap := range doc {
			for date, arr := range dayMap {
				if len(arr) < 2 {
					continue
				}
				dst := arr[1] != 0
				powers := arr[2:]
				sum, peak := 0, 0
				for _, p := range powers {
					sum += p
					if p > peak {
						peak = p
					}
				}
				days = append(days, dayRecord{
					date:     date,
					dst:      dst,
					powers:   powers,
					sum:      sum,
					peak:     peak,
					energyWh: (sum + 6) / 12, // round(sum/12)
				})
			}
		}
	}

	// Two-pass filter: determine the earliest date whose sample-sum > 0
	// across the whole dataset. Legacy placeholder months (all-zero days)
	// before that date are dropped entirely; a real zero-sum day on or
	// after the cutoff is kept.
	cutoff := ""
	for _, d := range days {
		if d.sum > 0 && (cutoff == "" || d.date < cutoff) {
			cutoff = d.date
		}
	}

	earliest := ""
	for _, d := range days {
		if cutoff != "" && d.date < cutoff {
			continue
		}
		samples := make([]store.Sample, 0, len(d.powers))
		for n, p := range d.powers {
			ts, err := timeutil.SlotTS(d.date, d.dst, n)
			if err != nil {
				return rep, err
			}
			samples = append(samples, store.Sample{TS: ts, PowerW: p})
		}
		if err := s.UpsertDay(id, d.date, d.energyWh, d.peak); err != nil {
			return rep, err
		}
		if err := s.UpsertSamples(id, samples); err != nil {
			return rep, err
		}
		rep.Days++
		rep.Samples += len(samples)
		y := int(d.date[0]-'0')*1000 + int(d.date[1]-'0')*100 + int(d.date[2]-'0')*10 + int(d.date[3]-'0')
		rep.YearKWh[y] += float64(d.energyWh) / 1000.0
		if earliest == "" || d.date < earliest {
			earliest = d.date
		}
	}

	if earliest != "" {
		meta.ID = id
		meta.Day1 = earliest
		if _, err := s.UpsertCollector(meta); err != nil {
			return rep, err
		}
	}
	return rep, nil
}
