const MONTHS = ['jan', 'feb', 'mrt', 'apr', 'mei', 'jun', 'jul', 'aug', 'sep', 'okt', 'nov', 'dec']
export function dateNL(d) {
  return `${+d.slice(6, 8)} ${MONTHS[+d.slice(4, 6) - 1]} ${d.slice(0, 4)}`
}
export const kwh = (wh) => (wh / 1000).toFixed(1)
export const euro = (wh, eleCost) => ((wh / 1000) * eleCost).toFixed(2)
export const co2 = (wh, factor) => ((wh / 1000) * factor).toFixed(1)

// todayBrussels() -> 'YYYYMMDD' for "today" in Europe/Brussels, regardless of
// the visitor's own clock/timezone (e.g. an evening visitor west of UTC must
// not see yesterday's date).
export function todayBrussels() {
  return new Intl.DateTimeFormat('sv-SE', { timeZone: 'Europe/Brussels' }).format(new Date()).replaceAll('-', '')
}
