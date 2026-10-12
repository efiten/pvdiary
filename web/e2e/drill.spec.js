import { test, expect } from '@playwright/test'

// Alle staven even hoog: dan ligt een klik op 70% van de ploth zeker binnen de
// staaf, ongeacht hoe echarts de y-as schaalt.
const MONTH_DAYS = Array.from({ length: 31 }, (_, i) => ({
  Date: `202608${String(i + 1).padStart(2, '0')}`,
  EnergyWh: 40000,
  PeakW: 5000,
}))
const TWELVE = Array(12).fill(100)
const KWH_YEAR = { 2021: TWELVE, 2022: TWELVE, 2023: TWELVE }

async function stubApi(page) {
  await page.route('**/api/summary*', (r) => r.fulfill({
    json: {
      collectors: [{ Key: 'foxess', Name: 'foxess' }],
      kwh_year: KWH_YEAR,
      last_date: '20260803',
      last_update: 1785000000,
      config: { ele_cost: 0.3, co2: 0.4, lat: 50.85, lon: 4.35, strings: [] },
    },
  }))
  await page.route('**/api/month*', (r) => r.fulfill({ json: MONTH_DAYS }))
  await page.route('**/api/year*', (r) => r.fulfill({
    json: { year: 2026, months: Array(12).fill(100000), kwh_year: KWH_YEAR, years: [2021, 2022, 2023] },
  }))
  await page.route('**/api/day*', (r) => r.fulfill({
    json: { date: '20260803', energy_wh: 40000, peak_w: 5000, samples: [] },
  }))
  await page.route('**/api/daymax*', (r) => r.fulfill({ json: {} }))
}

// mountBars gebruikt grid { left: 48, right: 16, top: 24, bottom: 32 }.
async function clickChartBar(page, chart, index, count) {
  const canvas = chart.locator('canvas')
  await expect(canvas).toBeVisible()
  await chart.scrollIntoViewIfNeeded()
  const box = await chart.boundingBox()
  const band = (box.width - 48 - 16) / count
  const lx = 48 + (index + 0.5) * band
  const ly = 24 + (box.height - 24 - 32) * 0.7

  // echarts animeert de staven omhoog; wacht tot er op dit punt echt accentkleur
  // staat, anders klikken we op lege grafiekachtergrond.
  await expect.poll(() => canvas.evaluate((c, [x, y]) => {
    const dpr = c.width / c.getBoundingClientRect().width
    const d = c.getContext('2d').getImageData(Math.round(x * dpr), Math.round(y * dpr), 1, 1).data
    const accent = getComputedStyle(document.documentElement).getPropertyValue('--accent').trim()
    const hex = '#' + [d[0], d[1], d[2]].map((v) => v.toString(16).padStart(2, '0')).join('')
    return hex === accent
  }, [lx, ly])).toBe(true)

  await page.mouse.click(box.x + lx, box.y + ly)
}

const clickBar = (page, cardId, index, count) =>
  clickChartBar(page, page.locator(`#${cardId} .chart`), index, count)

test.beforeEach(async ({ page }) => {
  await stubApi(page)
  await page.goto('/')
  await expect(page.locator('#view-day strong')).toHaveText('3 aug 2026')
})

test('klik op een dagstaaf in de maandview opent die dag in de dagview', async ({ page }) => {
  await clickBar(page, 'view-month', 11, 31) // index 11 -> 12 augustus
  await expect(page.locator('#view-day strong')).toHaveText('12 aug 2026')
})

test('klik op een maandstaaf in de jaarview opent die maand in de maandview', async ({ page }) => {
  await expect(page.locator('#view-month strong')).toHaveText('08/2026')
  await clickBar(page, 'view-year', 4, 12) // index 4 -> mei
  await expect(page.locator('#view-month strong')).toHaveText('05/2026')
})

test('klik op een jaarstaaf in alle jaren opent dat jaar in de jaarview', async ({ page }) => {
  await expect(page.locator('#view-year strong')).toHaveText('2026')
  const allYears = page.locator('.row .half').nth(1).locator('.chart')
  await clickChartBar(page, allYears, 0, 3) // index 0 -> 2021
  await expect(page.locator('#view-year strong')).toHaveText('2021')
})
