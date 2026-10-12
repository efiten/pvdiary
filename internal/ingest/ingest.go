package ingest

import (
	"time"

	"pvdiary/internal/store"
	"pvdiary/internal/timeutil"
)

type Service struct {
	s *store.Store
}

func New(s *store.Store) *Service { return &Service{s: s} }

func (svc *Service) collectorID(key string) (int64, error) {
	c, err := svc.s.CollectorByKey(key)
	if err == nil {
		return c.ID, nil
	}
	// onbekend -> aanmaken met defaults
	return svc.s.UpsertCollector(store.Collector{Key: key, Name: key})
}

func dateOf(ts int64) string {
	return time.Unix(ts, 0).In(timeutil.Loc).Format("20060102")
}

// Sample slaat een 5-min meetpunt op en houdt het dagtotaal + piek bij.
func (svc *Service) Sample(collectorKey string, ts int64, powerW int) error {
	id, err := svc.collectorID(collectorKey)
	if err != nil {
		return err
	}
	ts = timeutil.Round5m(ts)
	if err := svc.s.UpsertSample(id, ts, powerW); err != nil {
		return err
	}
	date := dateOf(ts)
	return svc.recomputeDay(id, collectorKey, date)
}

// recomputeDay werkt peak_w altijd bij uit de samples van die dag. energy_wh
// (som/12) wordt enkel geschreven bij het aanmaken van de dagrij; bestaat de
// rij al, dan blijft de bestaande energy_wh behouden (die wordt autoritatief
// bijgehouden door de 5-min Day()-post op basis van de omvormer-teller).
func (svc *Service) recomputeDay(id int64, collectorKey, date string) error {
	samples, err := svc.s.DaySamples(date, collectorKey)
	if err != nil {
		return err
	}
	sum, peak := 0, 0
	for _, sm := range samples {
		sum += sm.PowerW
		if sm.PowerW > peak {
			peak = sm.PowerW
		}
	}
	exists, err := svc.s.DayExists(id, date)
	if err != nil {
		return err
	}
	if exists {
		di, err := svc.s.DayInfo(date, collectorKey)
		if err != nil {
			return err
		}
		return svc.s.UpsertDay(id, date, di.EnergyWh, peak)
	}
	return svc.s.UpsertDay(id, date, (sum+6)/12, peak)
}

// Day zet het dagtotaal expliciet (omvormer-teller), behoudt de bestaande piek.
func (svc *Service) Day(collectorKey, date string, energyWh int) error {
	id, err := svc.collectorID(collectorKey)
	if err != nil {
		return err
	}
	di, err := svc.s.DayInfo(date, collectorKey)
	if err != nil {
		return err
	}
	return svc.s.UpsertDay(id, date, energyWh, di.PeakW)
}
