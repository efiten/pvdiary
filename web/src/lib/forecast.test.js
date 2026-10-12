import { describe, it, expect, vi, beforeEach } from 'vitest'
import { toMeteoAz, fetchForecast, forecastEnergyWh } from './forecast.js'

const STRINGS = [
  { wp: 5760, az: 130, tilt: 40 },
  { wp: 3200, az: 310, tilt: 40 },
]

function meteoResponse(gti) {
  return {
    ok: true,
    json: () =>
      Promise.resolve({
        minutely_15: {
          time: ['2026-07-26T12:00', '2026-07-26T12:15'],
          global_tilted_irradiance: gti,
        },
      }),
  }
}

beforeEach(() => {
  const store = new Map()
  globalThis.sessionStorage = {
    getItem: (k) => store.get(k) ?? null,
    setItem: (k, v) => store.set(k, v),
  }
})

describe('toMeteoAz', () => {
  it('converts compass to Open-Meteo convention', () => {
    expect(toMeteoAz(180)).toBe(0) // zuid
    expect(toMeteoAz(130)).toBe(-50) // zuidoost-achtig
    expect(toMeteoAz(310)).toBe(130) // noordwest-achtig
    expect(toMeteoAz(90)).toBe(-90) // oost
  })
})

describe('fetchForecast', () => {
  it('sums per-string power and builds correct urls', async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve(meteoResponse([1000, 500])))
    const slots = await fetchForecast('20260726', 50.85, 4.35, STRINGS)
    // 1000 W/m2: (5760+3200)*0.85 = 7616 W op slot 12:00
    expect(slots.get('12:00')).toBe(7616)
    expect(slots.get('12:15')).toBe(3808)
    const urls = fetch.mock.calls.map((c) => c[0])
    expect(urls[0]).toContain('azimuth=-50')
    expect(urls[1]).toContain('azimuth=130')
    expect(urls[0]).toContain('tilt=40')
    expect(urls[0]).toContain('start_date=2026-07-26')
  })

  it('returns null on http error', async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve({ ok: false }))
    expect(await fetchForecast('20260726', 50.85, 4.35, STRINGS)).toBeNull()
  })

  it('caches in sessionStorage', async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve(meteoResponse([1000, 500])))
    await fetchForecast('20260726', 50.85, 4.35, STRINGS)
    await fetchForecast('20260726', 50.85, 4.35, STRINGS)
    expect(fetch).toHaveBeenCalledTimes(2) // 2 strings, 1x — tweede call uit cache
  })
})

describe('forecastEnergyWh', () => {
  it('integrates quarter-hour slots to Wh', () => {
    const slots = new Map([
      ['12:00', 4000],
      ['12:15', 2000],
    ])
    expect(forecastEnergyWh(slots)).toBe(1500) // (4000+2000)/4
    expect(forecastEnergyWh(null)).toBe(0)
  })
})
