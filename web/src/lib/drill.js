// Vertaalt een klik op een staaf naar de periode die de bovenliggende view moet
// tonen. seriesIndex 1 is telkens de vergelijkingsreeks (vorige maand / vorig jaar).

const pad = (n) => String(n).padStart(2, '0')

export function shiftYm(ym, n) {
  const d = new Date(+ym.slice(0, 4), +ym.slice(4, 6) - 1 + n, 1)
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}`
}

export function daysInMonth(ym) {
  return new Date(+ym.slice(0, 4), +ym.slice(4, 6), 0).getDate()
}

// Maandview: staafindex (0-based dagnummer) -> 'YYYYMMDD', of null als die dag
// niet in de maand bestaat (bv. index 30 in een maand van 30 dagen).
export function dayFromBar(ym, dataIndex, seriesIndex = 0) {
  const target = seriesIndex === 1 ? shiftYm(ym, -1) : ym
  if (dataIndex < 0 || dataIndex >= daysInMonth(target)) return null
  return `${target}${pad(dataIndex + 1)}`
}

// Jaarview: staafindex (0-based maandnummer) -> 'YYYYMM'.
export function ymFromBar(year, dataIndex, seriesIndex = 0) {
  if (dataIndex < 0 || dataIndex > 11) return null
  return `${seriesIndex === 1 ? year - 1 : year}${pad(dataIndex + 1)}`
}
