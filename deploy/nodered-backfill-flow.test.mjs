// Tests voor de function-nodes in nodered-backfill-flow.json.
// Draaien met:  node --test deploy/
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

// Vul de placeholders in zoals een gebruiker dat bij installatie doet (zie README).
const flow = JSON.parse(
  readFileSync(new URL('./nodered-backfill-flow.json', import.meta.url), 'utf8')
    .replaceAll('__PVDIARY_URL__', 'https://pv.example.org')
    .replaceAll('__COLLECTOR__', 'foxess')
    .replaceAll('__PV_POWER_ENTITY__', 'sensor.foxess_pv_power')
    .replaceAll('__INGEST_TOKEN__', 'secret'),
)
const body = (id) => flow.find((n) => n.id === id).func

// Node-RED roept function-nodes aan met deze sandbox-variabelen.
function run(id, msg, { now, env = {} } = {}) {
  const node = { status: () => {}, warn: () => {} }
  const realNow = Date.now
  if (now) Date.now = () => now * 1000
  try {
    const f = new Function('msg', 'node', 'env', `${body(id)}`)
    return f(msg, node, { get: (k) => env[k] })
  } finally {
    Date.now = realNow
  }
}

const DAY = '20260806'
// 6 augustus 2026 00:00 in Europe/Brussels = 5 augustus 22:00 UTC
const START = Date.UTC(2026, 7, 5, 22, 0, 0) / 1000
const END = START + 86400

test('bepaal datum: vandaag versus gisteren', () => {
  const now = Date.UTC(2026, 7, 6, 12, 0, 0) / 1000
  assert.equal(run('pvfill_date', { topic: 'today' }, { now }).pvDate, '20260806')
  assert.equal(run('pvfill_date', { topic: 'yesterday' }, { now }).pvDate, '20260805')
})

test('bereken gaten: lege dag geeft alle slots tot nu', () => {
  const msg = run('pvfill_gaps', { pvDate: DAY, payload: { samples: [] } }, { now: END })
  assert.equal(msg.pvMissing.length, 288)
  assert.equal(msg.pvMissing[0], START)
  assert.equal(msg.pvMissing.at(-1), END - 300)
})

test('bereken gaten: stopt bij nu, niet bij middernacht', () => {
  const noon = START + 12 * 3600
  const msg = run('pvfill_gaps', { pvDate: DAY, payload: { samples: [] } }, { now: noon })
  assert.equal(msg.pvMissing.length, 144)
})

test('bereken gaten: vindt precies de ontbrekende slots', () => {
  const all = Array.from({ length: 288 }, (_, i) => START + i * 300)
  const gone = [all[10], all[100], all[287]]
  const samples = all.filter((t) => !gone.includes(t)).map((t) => [t, 1000])
  const msg = run('pvfill_gaps', { pvDate: DAY, payload: { samples } }, { now: END })
  assert.deepEqual(msg.pvMissing, gone)
})

test('bereken gaten: volledige dag geeft null en stuurt niets door', () => {
  const samples = Array.from({ length: 288 }, (_, i) => [START + i * 300, 0])
  assert.equal(run('pvfill_gaps', { pvDate: DAY, payload: { samples } }, { now: END }), null)
})

test('bereken gaten: zet de supervisor-token op de history-call', () => {
  const msg = run('pvfill_gaps', { pvDate: DAY, payload: { samples: [] } },
    { now: END, env: { SUPERVISOR_TOKEN: 'tok' } })
  assert.equal(msg.headers.Authorization, 'Bearer tok')
  assert.match(msg.url, /^http:\/\/supervisor\/core\/api\/history\/period\//)
  assert.match(msg.url, /filter_entity_id=sensor\.foxess_pv_power/)
})

test('bepaal datum: day-URL gebruikt de ingevulde URL en collector', () => {
  const now = Date.UTC(2026, 7, 6, 12, 0, 0) / 1000
  assert.equal(run('pvfill_date', { topic: 'today' }, { now }).url,
    'https://pv.example.org/api/day?date=20260806&collector=foxess')
})

// --- los gaten op -------------------------------------------------------

const at = (ts, state) => ({ last_updated: new Date(ts * 1000).toISOString(), state: String(state) })
const resolve = (missing, states) =>
  run('pvfill_resolve', { pvDate: DAY, pvMissing: missing, payload: [states] })[0]

test('los gaten op: levende sensor geeft de laatst bekende waarde', () => {
  const ts = START + 43200
  const out = resolve([ts], [at(ts - 20, 4100), at(ts - 5, 4231), at(ts + 10, 4180)])
  assert.equal(out.length, 1)
  assert.deepEqual(out[0].payload, { collector: 'foxess', ts, power_w: 4231 })
  assert.equal(out[0].method, 'POST')
})

test('los gaten op: gat tussen twee nullen wordt 0 W', () => {
  const ts = START + 3600
  const out = resolve([ts], [at(ts - 7200, 0), at(ts + 7200, 0)])
  assert.equal(out.length, 1)
  assert.equal(out[0].payload.power_w, 0)
})

test('los gaten op: bron plat overdag wordt overgeslagen', () => {
  const ts = START + 57600
  const out = resolve([ts], [at(ts - 7200, 0), at(ts + 7200, 2420)])
  assert.equal(out.length, 0)
})

test('los gaten op: zonder meting ervoor of erna wordt overgeslagen', () => {
  const ts = START + 3600
  assert.equal(resolve([ts], [at(ts + 60, 0)]).length, 0)
  assert.equal(resolve([ts], [at(ts - 60, 0)]).length, 0)
})

test('los gaten op: unavailable telt niet als meting', () => {
  const ts = START + 43200
  const out = resolve([ts], [at(ts - 20, 'unavailable'), at(ts + 10, 4180)])
  assert.equal(out.length, 0)
})

test('los gaten op: mengt opgeloste en overgeslagen slots', () => {
  const live = START + 43200
  const dark = START + 57600
  const out = resolve([live, dark], [
    at(live - 10, 4000), at(live + 10, 4050),
    at(dark - 7200, 0), at(dark + 7200, 2420),
  ])
  assert.equal(out.length, 1)
  assert.equal(out[0].payload.ts, live)
})
