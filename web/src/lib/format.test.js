import { describe, it, expect } from 'vitest'
import { todayBrussels } from './format.js'

describe('todayBrussels', () => {
  it('returns an 8-digit YYYYMMDD string', () => {
    expect(todayBrussels()).toMatch(/^\d{8}$/)
  })
})
