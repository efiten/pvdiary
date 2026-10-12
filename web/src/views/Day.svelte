<script>
  import { getDay, getDayMax, getSummary } from '../lib/api.js'
  import { mountLine } from '../lib/chart.js'
  import { dateNL, kwh, todayBrussels } from '../lib/format.js'
  import { sunTimes, clearSkyStrings } from '../lib/astro.js'
  import { fetchForecast, forecastEnergyWh } from '../lib/forecast.js'
  import { selection } from '../lib/selection.svelte.js'

  // locatie + strings komen uit de server-config (PVDIARY_LAT/LON/STRINGS)
  let site = $state(null)
  let el
  let info = $state({ energy_wh: 0, peak_w: 0 })
  let chart
  let sun = $state({ sunrise: '', sunset: '' })
  let peakTime = $state('')
  let csWh = $state(0)
  let fcWh = $state(0)

  // start op vandaag, maar val terug op de laatste dag met data
  getSummary().then((s) => {
    site = { lat: s.config.lat, lon: s.config.lon, strings: s.config.strings || [] }
    const today = todayBrussels()
    if (!selection.date) selection.date = s.last_date && s.last_date < today ? s.last_date : today
  })

  function shift(days) {
    const d0 = selection.date
    const d = new Date(+d0.slice(0, 4), +d0.slice(4, 6) - 1, +d0.slice(6, 8) + days)
    selection.date = `${d.getFullYear()}${String(d.getMonth() + 1).padStart(2, '0')}${String(d.getDate()).padStart(2, '0')}`
  }

  function fmtSlot(ts) {
    return new Date(ts * 1000).toLocaleTimeString('nl-BE', { hour: '2-digit', minute: '2-digit', timeZone: 'Europe/Brussels' })
  }

  async function load() {
    const date = selection.date
    const { lat: LAT, lon: LON, strings } = site
    const [res, maxMap] = await Promise.all([getDay(date), getDayMax(date, 7)])
    info = res
    sun = sunTimes(date, LAT, LON)

    // clear-sky bepaalt het master-grid (04:00-22:00, 5 min)
    const cs = strings.length ? clearSkyStrings(date, LAT, LON, strings) : []
    const labels = cs.length ? cs.map((p) => p.t) : res.samples.map((s) => fmtSlot(s[0]))
    const idx = new Map(labels.map((t, i) => [t, i]))
    const alignTo = (fill) => Array(labels.length).fill(fill)

    const actual = alignTo(null)
    for (const s of res.samples) {
      const i = idx.get(fmtSlot(s[0]))
      if (i !== undefined) actual[i] = s[1]
    }
    const hist = alignTo(null)
    for (const [t, w] of Object.entries(maxMap || {})) {
      const i = idx.get(t)
      if (i !== undefined) hist[i] = w
    }

    csWh = cs.length ? Math.round(cs.reduce((a, p) => a + p.w, 0) / 12) : 0

    // weerdata-verwachting alleen voor vandaag en later
    fcWh = 0
    let fcSeries = null
    if (strings.length && date >= todayBrussels()) {
      const slots = await fetchForecast(date, LAT, LON, strings)
      if (slots) {
        fcWh = forecastEnergyWh(slots)
        fcSeries = alignTo(null)
        for (const [t, w] of slots) {
          const i = idx.get(t)
          if (i !== undefined) fcSeries[i] = w // kwartierpunten; lijn verbindt ze
        }
      }
    }

    peakTime = ''
    if (res.samples.length) {
      const peakSample = res.samples.reduce((a, b) => (b[1] > a[1] ? b : a))
      peakTime = fmtSlot(peakSample[0])
    }

    const cssv = (n) => getComputedStyle(document.documentElement).getPropertyValue(n)
    const series = [
      { name: '±7d max', data: hist, area: true, areaOpacity: 0.35, color: cssv('--histmax'), width: 0, z: 1 },
      { name: 'Clear-sky', data: cs.map((p) => p.w), color: cssv('--cs'), z: 3 },
    ]
    if (fcSeries) series.push({ name: 'Verwacht (weer)', data: fcSeries, dashed: true, connectNulls: true, color: cssv('--fc'), z: 4 })
    series.push({ name: 'Werkelijk', data: actual, area: true, color: cssv('--accent'), z: 5 })

    if (chart) { chart.dispose(); chart = null }
    chart = mountLine(el, { xLabels: labels, series, legend: true })
  }
  $effect(() => { if (el && selection.date && site) load() })
</script>

<div class="nav-day">
  <button onclick={() => shift(-1)}>‹</button>
  <strong>{selection.date ? dateNL(selection.date) : '…'}</strong>
  <button onclick={() => shift(1)}>›</button>
  <span>Opbrengst {kwh(info.energy_wh)} kWh{#if csWh} · ({Math.round((100 * info.energy_wh) / csWh)}% van clear-sky {kwh(csWh)} kWh){/if}</span>
</div>
<div class="nav-day">
  <span class="sun">
    Zon: {sun.sunrise}–{sun.sunset}
    {#if peakTime}· piek {info.peak_w} W om {peakTime}{/if}
    {#if fcWh}· verwacht (weerdata): {kwh(fcWh)} kWh{/if}
  </span>
</div>
{#if selection.date && !peakTime}<p class="nodata">Geen metingen voor deze dag</p>{/if}
<div class="chart" bind:this={el}></div>
