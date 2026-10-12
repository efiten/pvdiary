import { describe, it, expect } from 'vitest'
import * as astro from './astro.js'

describe('astro', () => {
  it('sunrise before sunset in summer in Brussels', () => {
    const { sunrise, sunset } = astro.sunTimes('20250621', 50.8467, 4.3525)
    expect(sunrise < sunset).toBe(true)
    expect(sunrise).toMatch(/^0[45]:/) // ~05:2x CEST
  })
  it('clear-sky peaks around midday and is zero at night', () => {
    const cs = astro.clearSky('20250621', 50.8467, 4.3525, 8960)
    const peak = Math.max(...cs.map((p) => p.w))
    expect(peak).toBeGreaterThan(0)
    expect(cs[0].w).toBe(0)
  })

  it('grid contract: 217 samples 04:00–22:00 at 5-min intervals', () => {
    const cs = astro.clearSky('20250621', 50.8467, 4.3525, 8960)
    expect(cs.length).toBe(217)
    expect(cs[0].t).toBe('04:00')
    expect(cs[216].t).toBe('22:00')
    // every t matches HH:MM format
    cs.forEach((sample) => {
      expect(sample.t).toMatch(/^\d{2}:\d{2}$/)
    })
    // last sample is zero (after sunset)
    expect(cs[216].w).toBe(0)
    // all watts >= 0
    cs.forEach((sample) => {
      expect(sample.w).toBeGreaterThanOrEqual(0)
    })
  })

  it('winter sunTimes: sunrise ~08:34 and sunset ~17:00 CET', () => {
    const { sunrise, sunset } = astro.sunTimes('20250115', 50.8467, 4.3525)
    expect(sunrise).toMatch(/^08:/)
    expect(sunset).toMatch(/^17:/)
    expect(sunrise < sunset).toBe(true)
  })

  it('winter clear-sky peak much lower than summer', () => {
    const summerCs = astro.clearSky('20250621', 50.8467, 4.3525, 8960)
    const winterCs = astro.clearSky('20250115', 50.8467, 4.3525, 8960)
    const summerPeak = Math.max(...summerCs.map((p) => p.w))
    const winterPeak = Math.max(...winterCs.map((p) => p.w))
    expect(winterPeak).toBeGreaterThan(0)
    expect(winterPeak).toBeLessThan(summerPeak * 0.6)
  })
})

describe('clearSkyStrings (full per-string model)', () => {
  // User's real installation in Brussels: 5760 Wp az130 + 3200 Wp az310, tilt 40, temp_coeff -0.38.
  const LAT = 50.8467
  const LON = 4.3525
  const strings = [
    { wp: 5760, az: 130, tilt: 40, temp_coeff: -0.38 },
    { wp: 3200, az: 310, tilt: 40, temp_coeff: -0.38 },
  ]
  const peakOf = (cs) => cs.reduce((best, s) => (s.w > best.w ? s : best), { w: -1, t: '' })
  const energyOf = (cs) => cs.reduce((a, s) => a + s.w, 0) / 12

  it('summer peak power ~5845 W between 12:30 and 14:00 (legacy validation target)', () => {
    const cs = astro.clearSkyStrings('20260705', LAT, LON, strings)
    const peak = peakOf(cs)
    expect(peak.w).toBeGreaterThan(5845 * 0.9)
    expect(peak.w).toBeLessThan(5845 * 1.1)
    expect(peak.t >= '12:30' && peak.t <= '14:00').toBe(true)
  })

  it('summer day energy ~52996 Wh (legacy validation target)', () => {
    const cs = astro.clearSkyStrings('20260705', LAT, LON, strings)
    const wh = energyOf(cs)
    expect(wh).toBeGreaterThan(52996 * 0.9)
    expect(wh).toBeLessThan(52996 * 1.1)
  })

  it('all-zero at night (first and last sample)', () => {
    const winter = astro.clearSkyStrings('20260115', LAT, LON, strings)
    expect(winter[0].w).toBe(0) // 04:00, before sunrise
    expect(winter[216].w).toBe(0) // 22:00, after sunset
    const summer = astro.clearSkyStrings('20260705', LAT, LON, strings)
    expect(summer[0].w).toBe(0) // 04:00 local == 03:00 std, still night
  })

  it('217-sample grid 04:00-22:00 preserved', () => {
    const cs = astro.clearSkyStrings('20260705', LAT, LON, strings)
    expect(cs.length).toBe(217)
    expect(cs[0].t).toBe('04:00')
    expect(cs[216].t).toBe('22:00')
    cs.forEach((s) => {
      expect(s.t).toMatch(/^\d{2}:\d{2}$/)
      expect(s.w).toBeGreaterThanOrEqual(0)
    })
  })

  it('winter peak far below summer peak', () => {
    const summerPeak = peakOf(astro.clearSkyStrings('20260705', LAT, LON, strings)).w
    const winterPeak = peakOf(astro.clearSkyStrings('20260115', LAT, LON, strings)).w
    expect(winterPeak).toBeGreaterThan(0)
    expect(winterPeak).toBeLessThan(summerPeak * 0.6)
  })
})
