import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from './api.js'

beforeEach(() => {
  globalThis.fetch = vi.fn(() =>
    Promise.resolve({ ok: true, json: () => Promise.resolve({ ok: 1 }) })
  )
})

describe('api', () => {
  it('builds day url with date and collector', async () => {
    await api.getDay('20250601', 'foxess')
    expect(fetch).toHaveBeenCalledWith('/api/day?date=20250601&collector=foxess')
  })
  it('omits empty collector', async () => {
    await api.getMonth('202506')
    expect(fetch).toHaveBeenCalledWith('/api/month?ym=202506')
  })
})
