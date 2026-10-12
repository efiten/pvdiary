<script>
  import { getSummary } from '../lib/api.js'
  import { mountBars } from '../lib/chart.js'
  import { show } from '../lib/selection.svelte.js'

  let el, chart
  let totalMWh = $state(0)

  async function load() {
    const s = await getSummary()
    const ky = s.kwh_year || {}
    const years = Object.keys(ky).sort()
    const totals = years.map((yr) => +ky[yr].reduce((a, b) => a + b, 0).toFixed(0))
    totalMWh = totals.reduce((a, b) => a + b, 0) / 1000
    if (chart) chart.dispose()
    chart = mountBars(el, {
      xLabels: years,
      values: totals,
      name: 'kWh/jaar',
      onClick: (i) => show('year', years[i] ? +years[i] : null),
    })
  }
  $effect(() => { if (el) load() })
</script>

<div class="nav-day">
  <strong>Alle jaren</strong>
  <span>Totaal {totalMWh.toFixed(1)} MWh</span>
</div>
<div class="chart" bind:this={el}></div>
