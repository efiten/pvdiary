# PVdiary

Een zelf-gehoste website voor de opbrengst van je zonnepanelen: dag-, maand-, jaar- en
alle-jaren-grafieken, topdagen, opbrengst in euro en CO2, een clear-sky-curve en een
weersverwachting voor vandaag. De data komt uit Home Assistant.

Eén binary (Go, frontend ingebakken) met een SQLite-database. Draait als Docker-container.

> **Gebruik je een AI-assistent (Claude Code e.d.) om dit te installeren?**
> Laat hem eerst `CLAUDE.md` lezen: daarin staat welke vragen hij jou moet stellen en hoe
> hij elke stap controleert.

---

## Hoe het werkt

```
 ┌──────────── Home Assistant ────────────┐                ┌──── Docker-host ────┐
 │                                        │                │                     │
 │  omvormer-integratie                   │                │   pvdiary           │
 │   ├─ sensor: vermogen nu (W)           │   HTTP(S) POST │   :8080             │
 │   └─ sensor: energie vandaag (kWh)     │  ────────────► │   /api/ingest/*     │
 │            │                           │  elke 5 min    │   (Bearer-token)    │
 │            ▼                           │                │                     │
 │  Node-RED add-on                       │                │   SQLite in /data   │
 │   └─ flow "pvdiary ingest"             │                │                     │
 └────────────────────────────────────────┘                └──────────┬──────────┘
                                                                      │ HTTP(S)
                                                           browser ◄──┘
                                                              │
                                                              └──► api.open-meteo.com
                                                                   (weersverwachting)
```

**De verbinding is eenrichtingsverkeer: Home Assistant pusht naar PVdiary.**

- Node-RED draait *in* Home Assistant, leest de sensoren lokaal uit en stuurt elke 5 minuten
  een meetpunt en het dagtotaal naar PVdiary.
- PVdiary maakt **nooit** zelf verbinding met Home Assistant. De Docker-host heeft geen
  toegang tot HA nodig en HA hoeft niet extern bereikbaar te zijn.
- Wat wel moet: **Home Assistant moet de PVdiary-URL kunnen bereiken.** Test dat vanaf de
  HA-machine, niet vanaf je laptop.
- De optionele backfill-flow (gaten opvullen na een storing) draait ook in Node-RED: hij
  leest de HA-geschiedenis via de interne supervisor-API en vraagt PVdiary welke
  meetpunten ontbreken. Ook hier legt alleen HA verbindingen.
- De weersverwachting haalt de browser van de bezoeker rechtstreeks bij Open-Meteo.

### Netwerkscenario's

| Waar draait PVdiary? | URL in de flow | Opmerking |
|---|---|---|
| Andere machine in je LAN (NAS, Pi, server) | `http://192.168.x.y:8080` | Eenvoudigst. Site alleen zichtbaar in je LAN. |
| VPS / server op internet | `https://pv.jouwdomein.be` | Zet er een reverse proxy met HTTPS voor (nginx, Caddy, Traefik). Gebruik geen plain HTTP over internet: het token gaat mee in elke request. |
| Zelfde machine als HA (HA Container/Supervised met Docker) | `http://<host-ip>:8080` | Niet `localhost`: vanuit de Node-RED-container is dat de container zelf. |

Home Assistant OS kan geen losse Docker-containers draaien. Gebruik dan een andere machine.

De lees-API en de website hebben **geen login**. Iedereen die de poort kan bereiken, ziet je
opbrengstdata. Alleen schrijven (`/api/ingest/*`) vereist het token.

---

## Wat je nodig hebt

- Home Assistant met de **Node-RED add-on** (bevat de Home Assistant-nodes). De
  backfill-flow werkt alleen met de add-on, niet met een losse Node-RED-installatie.
- Twee sensoren van je omvormer in HA:
  - **actueel vermogen** in **W** (bv. `sensor.foxess_pv_power`)
  - **energie vandaag** in **kWh**, die elke nacht naar 0 gaat (bv. `sensor.foxess_pv_energy_today`)
- Een machine met Docker en Docker Compose, bereikbaar vanaf HA.
- Locatie en paneelgegevens (zie stap 1).

---

## Installatie, stap voor stap

### Stap 1: gegevens verzamelen

| Gegeven | Waar vind je het | Voorbeeld |
|---|---|---|
| Entity-id vermogen | HA → Ontwikkelhulpmiddelen → Statussen, zoek op `power`. Controleer het attribuut `unit_of_measurement`: moet `W` zijn. | `sensor.foxess_pv_power` |
| Entity-id energie vandaag | Idem, zoek op `today` of `daily`. Eenheid `kWh`, reset elke nacht. | `sensor.foxess_pv_energy_today` |
| Breedte- en lengtegraad | Google Maps, rechtsklik op je dak | `50.8467`, `4.3525` |
| Per dakvlak/string: wattpiek | aantal panelen × Wp per paneel | `18 × 320 = 5760` |
| Per dakvlak: azimut | kompasrichting waarin de panelen kijken: 0=N, 90=O, 180=Z, 270=W | `130` (zuidoost) |
| Per dakvlak: helling | dakhelling in graden | `40` |
| Per dakvlak: temperatuurcoëfficiënt | datasheet paneel, "Pmax temperature coefficient" in %/°C | `-0.38` |
| URL waarop PVdiary bereikbaar wordt | zie netwerkscenario's | `http://192.168.1.50:8080` |

**Sensor heeft een andere eenheid?**
- Vermogen in kW: pas in de flow-node `build sample` `Math.round(Number(msg.data.state))`
  aan naar `Math.round(Number(msg.data.state) * 1000)`.
- Alleen een totaalteller (lifetime kWh) en geen "vandaag"-sensor: maak in HA een helper
  **Utility Meter** (Instellingen → Apparaten & diensten → Helpers) met als bron die teller en cyclus **dagelijks**, en gebruik die.

### Stap 2: PVdiary starten

```bash
git clone <deze repo> pvdiary
cd pvdiary
cp .env.example .env
openssl rand -hex 32          # kopieer de uitvoer naar INGEST_TOKEN in .env
nano .env                     # vul PVDIARY_LAT, PVDIARY_LON, PVDIARY_STRINGS in
docker compose up -d --build
```

Voorbeeld `PVDIARY_STRINGS` voor twee dakvlakken:

```
PVDIARY_STRINGS=ZO:5760:130:40:-0.38,NW:3200:310:40:-0.38
```

Formaat per string: `naam:wattpiek:azimut:helling:tempcoeff`, strings gescheiden door een
komma. Eén dakvlak is gewoon één string.

Controle:

```bash
docker compose logs pvdiary        # moet "pvdiary serving on 0.0.0.0:8080" tonen
curl http://localhost:8080/api/summary
```

In `config` van het antwoord moeten je lat, lon en strings staan. Start de container niet,
dan staat de reden in de log (bv. `PVDIARY_STRINGS: ...`).

Gebruik je een bind mount in plaats van het volume `pvdiary-data`? De container draait als
uid `65532`: `sudo chown 65532:65532 <map>`.

### Stap 3: bereikbaarheid vanaf Home Assistant testen

Vanaf HA, niet vanaf je laptop. Bv. via de add-on "Terminal & SSH":

```bash
curl -s -o /dev/null -w "%{http_code}\n" <PVDIARY_URL>/api/summary    # verwacht: 200
```

### Stap 4: Node-RED flow importeren

Het bestand `deploy/nodered-flow.json` bevat placeholders. Kopieer het naar
`deploy/nodered-flow.local.json` (staat in `.gitignore`, want het bevat je token) en vul de
placeholders in **vóór** het importeren (zoek en vervang alle voorkomens):

| Placeholder | Vervang door | Voorbeeld |
|---|---|---|
| `__PVDIARY_URL__` | URL van PVdiary, zonder slash op het einde | `http://192.168.1.50:8080` |
| `__INGEST_TOKEN__` | de `INGEST_TOKEN` uit `.env` | `3f9a...` |
| `__COLLECTOR__` | korte naam voor je installatie, alleen letters/cijfers | `pv` |
| `__PV_POWER_ENTITY__` | entity-id vermogen | `sensor.foxess_pv_power` |
| `__PV_ENERGY_ENTITY__` | entity-id energie vandaag | `sensor.foxess_pv_energy_today` |

Daarna in Node-RED:

1. Menu (☰) → Import → plak de ingevulde JSON → Import.
2. Open de nodes **Get PV Power** en **Get PV Energy Today** en kies bij *Server* je
   Home Assistant-server (die staat op `REPLACE_WITH_SERVER_ID`). Bestaat er nog geen
   server-config, maak die dan aan: in de add-on volstaat "Using the Home Assistant Add-on".
3. Deploy.

Controle: binnen 5 minuten staat bij de HTTP-request-nodes een groene status, en
`curl <PVDIARY_URL>/api/summary` toont een gevulde `last_update`. Open de site in je browser.

Krijg je `401`? Dan klopt het token niet. Zie je een fout zoals `ECONNREFUSED` of `ETIMEDOUT`?
Dan bereikt HA de URL niet (terug naar stap 3).

### Stap 5 (optioneel): backfill-flow

Als HA of de verbinding even wegvalt, vult `deploy/nodered-backfill-flow.json` de ontbrekende
5-minutenpunten achteraf aan uit de HA-geschiedenis. Hij draait 30 minuten na zonsondergang
(vandaag) en om 00:10 (gisteren). Hij vult alleen waarden in die zeker zijn, de rest slaat
hij over.

Zelfde placeholders als stap 4 (`__PV_ENERGY_ENTITY__` komt er niet in voor). Na het
importeren bij de node **zon onder** de server kiezen, dan Deploy.

Vereist: de HA-recorder bewaart de vermogenssensor (standaard 10 dagen).

---

## Configuratie (omgevingsvariabelen)

| Variabele | Verplicht | Standaard | Betekenis |
|---|---|---|---|
| `INGEST_TOKEN` | ja | | token voor `/api/ingest/*` |
| `PVDIARY_LAT` | ja | | breedtegraad, -90..90 |
| `PVDIARY_LON` | ja | | lengtegraad, -180..180 |
| `PVDIARY_STRINGS` | ja | | zie stap 2 |
| `PVDIARY_ELE_COST` | nee | `0.18` | EUR per kWh |
| `PVDIARY_CO2` | nee | `0.596` | kg CO2 per kWh |
| `PVDIARY_ADDR` | nee | `0.0.0.0:8080` (Docker) | luisteradres |
| `PVDIARY_DB` | nee | `/data/pvdiary.db` (Docker) | pad naar SQLite |

## Beperkingen

- **Tijdzone vast op Europe/Brussels.** Daggrenzen en alle tijden in de UI gebruiken die
  tijdzone. Prima voor België en Nederland (zelfde tijdzone), niet daarbuiten.
- Interface is Nederlandstalig.
- Bedoeld voor één installatie (één `__COLLECTOR__`).
- `pvdiary import` leest alleen het oude PVdiary/SMA-JSON-formaat. Bestaande historie uit
  andere bronnen importeren wordt niet ondersteund.
- Geen login op de website (zie netwerkscenario's).

## Beheer

- **Update:** `git pull && docker compose up -d --build`. Het databaseschema migreert
  zichzelf.
- **Backup:** de database staat in het volume `pvdiary-data`. Stop de container even en
  kopieer `pvdiary.db`, of gebruik `sqlite3 pvdiary.db ".backup backup.db"`.

## Ontwikkeling

```bash
go test ./...                                        # backend
(cd web && npm ci && npx vitest run)                 # frontend unit tests
node --test deploy/nodered-backfill-flow.test.mjs    # backfill-flow logica
(cd web && npm run build)                            # frontend → web/dist (wordt ingebakken)
go run ./cmd/pvdiary serve                           # met de env-variabelen hierboven
```

`web/dist` staat in git omdat `go:embed` het nodig heeft. De Docker-build bouwt de frontend
zelf opnieuw.
