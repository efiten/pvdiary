function qs(params) {
  const p = Object.entries(params).filter(([, v]) => v !== undefined && v !== '' && v !== null)
  return p.length ? '?' + p.map(([k, v]) => `${k}=${v}`).join('&') : ''
}
async function get(path, params = {}) {
  const res = await fetch(path + qs(params))
  if (!res.ok) throw new Error(`${path} -> ${res.status}`)
  return res.json()
}
export const getSummary = (collector) => get('/api/summary', { collector })
export const getDay = (date, collector) => get('/api/day', { date, collector })
export const getMonth = (ym, collector) => get('/api/month', { ym, collector })
export const getYear = (y, collector) => get('/api/year', { y, collector })
export const getTop = (scope, key, n) => get('/api/top', { scope, key, n })
export const getDayMax = (date, days, collector) => get('/api/daymax', { date, days, collector })
