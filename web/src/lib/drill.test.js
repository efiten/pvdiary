import { describe, it, expect } from 'vitest'
import { shiftYm, daysInMonth, dayFromBar, ymFromBar } from './drill.js'

describe('shiftYm', () => {
  it('shifts within a year', () => {
    expect(shiftYm('202608', 1)).toBe('202609')
    expect(shiftYm('202608', -1)).toBe('202607')
  })
  it('crosses year boundaries', () => {
    expect(shiftYm('202601', -1)).toBe('202512')
    expect(shiftYm('202612', 1)).toBe('202701')
  })
})

describe('daysInMonth', () => {
  it('handles 30/31-day months', () => {
    expect(daysInMonth('202604')).toBe(30)
    expect(daysInMonth('202608')).toBe(31)
  })
  it('handles february in normal and leap years', () => {
    expect(daysInMonth('202602')).toBe(28)
    expect(daysInMonth('202402')).toBe(29)
  })
})

describe('dayFromBar', () => {
  it('maps a bar index to the day of the shown month', () => {
    expect(dayFromBar('202608', 0)).toBe('20260801')
    expect(dayFromBar('202608', 2)).toBe('20260803')
    expect(dayFromBar('202608', 30)).toBe('20260831')
  })
  it('returns null for a day the month does not have', () => {
    expect(dayFromBar('202604', 30)).toBeNull()
    expect(dayFromBar('202602', 28)).toBeNull()
  })
  it('maps the compare series to the previous month', () => {
    expect(dayFromBar('202608', 0, 1)).toBe('20260701')
    expect(dayFromBar('202601', 4, 1)).toBe('20251205')
  })
  it('validates the compare day against the previous month', () => {
    expect(dayFromBar('202603', 30, 1)).toBeNull() // feb 2026 heeft 28 dagen
  })
  it('rejects out-of-range indices', () => {
    expect(dayFromBar('202608', -1)).toBeNull()
    expect(dayFromBar('202608', 31)).toBeNull()
  })
})

describe('ymFromBar', () => {
  it('maps a bar index to the month of the shown year', () => {
    expect(ymFromBar(2026, 0)).toBe('202601')
    expect(ymFromBar(2026, 7)).toBe('202608')
    expect(ymFromBar(2026, 11)).toBe('202612')
  })
  it('maps the compare series to the previous year', () => {
    expect(ymFromBar(2026, 0, 1)).toBe('202501')
    expect(ymFromBar(2026, 11, 1)).toBe('202512')
  })
  it('rejects out-of-range indices', () => {
    expect(ymFromBar(2026, -1)).toBeNull()
    expect(ymFromBar(2026, 12)).toBeNull()
  })
})
