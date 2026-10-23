CREATE TABLE IF NOT EXISTS collector (
  id        INTEGER PRIMARY KEY,
  key       TEXT UNIQUE NOT NULL,
  name      TEXT NOT NULL,
  latitude  REAL NOT NULL DEFAULT 0,
  longitude REAL NOT NULL DEFAULT 0,
  location  TEXT NOT NULL DEFAULT '',
  pv_watt   INTEGER NOT NULL DEFAULT 0,
  day1      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS day (
  collector_id INTEGER NOT NULL REFERENCES collector(id),
  date         TEXT NOT NULL,
  energy_wh    INTEGER NOT NULL,
  peak_w       INTEGER NOT NULL,
  PRIMARY KEY (collector_id, date)
);

CREATE TABLE IF NOT EXISTS sample (
  collector_id INTEGER NOT NULL REFERENCES collector(id),
  ts           INTEGER NOT NULL,
  power_w      INTEGER NOT NULL,
  PRIMARY KEY (collector_id, ts)
);

-- Zonder collector filtert DaySamples alleen op ts, en de primary key begint met
-- collector_id. Zonder deze index scande elke dagquery de hele tabel.
CREATE INDEX IF NOT EXISTS sample_ts ON sample(ts);
