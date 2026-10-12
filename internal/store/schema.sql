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
