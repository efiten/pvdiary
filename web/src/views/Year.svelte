<script>
  import { getYear } from '../lib/api.js'
  import { mountBars } from '../lib/chart.js'
  import { ymFromBar } from '../lib/drill.js'
  import { selection, show } from '../lib/selection.svelte.js'

  const MONTHS = ['jan', 'feb', 'mrt', 'apr', 'mei', 'jun', 'jul', 'aug', 'sep', 'okt', 'nov', 'dec']
  let compare = $state(false)
  let el, chart
  let total = $state(0)
  let prevTotal = $state(0)

  async function load() {
    const y = selection.year
    const res = await getYear(y)
    const ky = res.kwh_year || {}
    const ys = (res.months || []).map((wh) => +(wh / 1000).toFixed(1))
    total = ys.reduce((a, b) => a + b, 0)
    let cmp = null
    if (compare) {
      const prev = ky[String(y - 1)]
      cmp = prev ? prev.map((v) => +v.toFixed(1)) : Array(12).fill(null)
      prevTotal = prev ? prev.reduce((a, b) => a + b, 0) : 0
    }
    if (chart) chart.dispose()
    chart = mountBars(el, {
      xLabels: MONTHS,
      values: ys,
      name: String(y),
      compareValues: cmp,
      compareName: String(y - 1),
      onClick: (i, s) => show('ym', ymFromBar(y, i, s)),
    })
  }
  $effect(() => { selection.year; compare; if (el) load() })
</script>

<div class="nav-day">
  <button onclick={() => (selection.year -= 1)}>‹</button>
  <strong>{selection.year}</strong>
  <button onclick={() => (selection.year += 1)}>›</button>
  <span>{total.toFixed(0)} kWh{#if compare} · {selection.year - 1}: {prevTotal.toFixed(0)} kWh{/if}</span>
  <label class="cmp"><input type="checkbox" bind:checked={compare} /> vorig jaar</label>
</div>
<div class="chart" bind:this={el}></div>
