# Instructies voor AI-assistenten

Je helpt iemand PVdiary te installeren voor de eigen zonnepanelen, gekoppeld aan diens
Home Assistant. `README.md` is de bron voor alle stappen. Dit bestand zegt hoe je het
aanpakt. Antwoord de gebruiker in diens eigen taal.

## Eerst begrijpen

- PVdiary is passief: het ontvangt data. **Home Assistant (Node-RED) pusht naar PVdiary.**
  PVdiary maakt nooit verbinding met HA. De enige netwerkeis: HA moet de PVdiary-URL kunnen
  bereiken. Leg dat aan de gebruiker uit voordat jullie een host kiezen.
- Normaal hoef je geen code te wijzigen. Alles wat per installatie verschilt, zit in `.env`
  en in de placeholders van de Node-RED flows.

## Vragen die je moet stellen

Vraag dit allemaal vóór je iets uitvoert. Vraag niet wat je zelf kunt opzoeken: heb je
toegang tot Home Assistant (MCP-tools, API), zoek de sensoren dan zelf en laat de gebruiker
bevestigen.

1. **Home Assistant-installatie:** HA OS, Supervised of Container? Draait de **Node-RED
   add-on**? Zonder add-on: de ingest-flow werkt met elke Node-RED die HA-nodes heeft, de
   backfill-flow niet (die gebruikt `SUPERVISOR_TOKEN`).
2. **Waar draait PVdiary?** Welke machine heeft Docker (NAS, Pi, server, VPS)? Wat is het IP
   of de domeinnaam? Moet de site alleen in het LAN zichtbaar zijn of publiek (dan HTTPS
   via een reverse proxy, en de gebruiker moet weten dat de site geen login heeft)?
3. **Sensoren:** het entity-id voor het actuele vermogen en voor de energie van vandaag.
   Controleer de eenheden (`W` en `kWh`) en dat de dagteller 's nachts naar 0 gaat. Zie
   README stap 1 voor andere eenheden of alleen een totaalteller.
4. **Locatie:** breedte- en lengtegraad van de panelen.
5. **Panelen per dakvlak/string:** aantal × Wp, azimut (0=N, 90=O, 180=Z, 270=W), helling,
   temperatuurcoëfficiënt uit de datasheet. Weet de gebruiker de coëfficiënt niet: stel
   `-0.35` voor en zeg dat het een schatting is.
6. **Optioneel:** elektriciteitsprijs per kWh, en of de backfill-flow gewenst is.
7. **Tijdzone:** woont de gebruiker buiten België/Nederland, zeg dan vooraf dat daggrenzen en
   tijden vast op Europe/Brussels staan (README, Beperkingen).

## Uitvoeren en controleren

Volg README stap 2 t/m 5. Ga pas verder als de controle van de vorige stap lukt:

| Stap | Controle |
|---|---|
| 2: container | `docker compose logs pvdiary` toont `pvdiary serving on`, en `curl <host>:8080/api/summary` geeft je lat/lon/strings terug in `config` |
| 3: bereikbaarheid | `curl` **vanaf de HA-machine** naar `<URL>/api/summary` geeft `200` |
| 4: ingest-flow | na ≤ 5 min heeft `/api/summary` een recente `last_update`, de site toont data van vandaag |
| 5: backfill | Node-RED-debug toont `pvdiary backfill <datum>: N gevuld, M onbekend` na de volgende trigger |

`PVDIARY_STRINGS` heeft het formaat `naam:wp:azimut:helling:tempcoeff`, strings gescheiden
door komma's. Een fout formaat laat de container stoppen met een duidelijke melding in de log.

## Regels

- Het `INGEST_TOKEN` is geheim. Genereer het met `openssl rand -hex 32`. Zet het nooit in een
  bestand dat gecommit wordt: `.env` staat in `.gitignore`. Bewaar ingevulde flows buiten
  de repo of als `*.local.json` (ook genegeerd), en wijzig de originele flowbestanden in
  `deploy/` niet.
- Vraag toestemming voordat je iets wijzigt in Home Assistant, Node-RED of op een server.
- Plain HTTP alleen binnen het LAN. Over internet altijd HTTPS.
- Gebruik in de flow-URL geen `localhost`: vanuit Node-RED is dat de Node-RED-container zelf.
- Moet je toch code wijzigen: draai `go test ./...`, `cd web && npx vitest run` en
  `node --test deploy/nodered-backfill-flow.test.mjs`. Na een frontendwijziging:
  `cd web && npm run build` (`web/dist` wordt in de binary ingebakken).
