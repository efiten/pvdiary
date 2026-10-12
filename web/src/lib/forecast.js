// Weerdata-verwachting via Open-Meteo: instraling op gekanteld vlak (GTI) per string.
// Gratis, geen key; alleen voor vandaag/morgen relevant.

const PR = 0.85 // performance ratio (bekabeling, vervuiling, omvormer, temperatuur)

// Open-Meteo azimut: 0 = zuid, -90 = oost, +90 = west; wij voeren kompasgraden.
export function toMeteoAz(compass) {
  let az = compass - 180
  if (az < -180) az += 360
  if (az > 180) az -= 360
  return az
}

function isoDate(d) {
  return `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`
}

// Haalt per string de GTI-reeks op en sommeert naar AC-vermogen per kwartier.
// Retourneert Map van 'HH:MM' -> W voor de gevraagde dag, of null bij fout/geen data.
export async function fetchForecast(date, lat, lon, strings) {
  const day = isoDate(date)
  const key = `pvfc:${date}:${strings.map((s) => s.az).join(',')}`
  const cached = sessionStorage.getItem(key)
  if (cached) {
    const { at, data } = JSON.parse(cached)
    if (Date.now() - at < 3600_000) return new Map(data)
  }
  const slots = new Map()
  try {
    for (const s of strings) {
      const url =
        `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}` +
        `&minutely_15=global_tilted_irradiance&tilt=${s.tilt}&azimuth=${toMeteoAz(s.az)}` +
        `&start_date=${day}&end_date=${day}&timezone=Europe%2FBrussels`
      const res = await fetch(url)
      if (!res.ok) return null
      const j = await res.json()
      const times = j.minutely_15?.time || []
      const gti = j.minutely_15?.global_tilted_irradiance || []
      for (let i = 0; i < times.length; i++) {
        const t = times[i].slice(11, 16) // 'HH:MM'
        const w = ((gti[i] || 0) / 1000) * s.wp * PR
        slots.set(t, (slots.get(t) || 0) + w)
      }
    }
  } catch {
    return null
  }
  const rounded = new Map([...slots].map(([t, w]) => [t, Math.round(w)]))
  sessionStorage.setItem(key, JSON.stringify({ at: Date.now(), data: [...rounded] }))
  return rounded
}

// Verwachte dagopbrengst in Wh uit de kwartierwaarden.
export function forecastEnergyWh(slots) {
  if (!slots) return 0
  let sum = 0
  for (const w of slots.values()) sum += w
  return Math.round(sum / 4) // kwartieren -> Wh
}
