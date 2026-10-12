<script>
  import { getMonth } from '../lib/api.js'
  import { mountBars } from '../lib/chart.js'
  import { kwh } from '../lib/format.js'
  import { dayFromBar, shiftYm } from '../lib/drill.js'
  import { selection, show } from '../lib/selection.svelte.js'

  let compare = $state(false)
  let el, chart
  let total = $state(0)
  let prevTotal = $state(0)

  const prevYm = (v) => shiftYm(v, -1)
  // dagtotalen -> vaste array per dagnummer (1..31), zodat beide maanden uitlijnen
  function byDay(days) {
    const out = Array(31).fill(null)
    for (const d of days) out[+d.Date.slice(6, 8) - 1] = +(d.EnergyWh / 1000).toFixed(2)
    return out
  }
  async function load() {
    const ym = selection.ym
    const days = await getMonth(ym)
    total = days.reduce((a, d) => a + d.EnergyWh, 0)
    const xs = Array.from({ length: 31 }, (_, i) => i + 1)
    let cmp = null
    if (compare) {
      const prev = await getMonth(prevYm(ym))
      prevTotal = prev.reduce((a, d) => a + d.EnergyWh, 0)
      cmp = byDay(prev)
    }
    if (chart) chart.dispose()
    chart = mountBars(el, {
      xLabels: xs,
      values: byDay(days),
      name: `${ym.slice(4, 6)}/${ym.slice(0, 4)}`,
      compareValues: cmp,
      compareName: `${prevYm(ym).slice(4, 6)}/${prevYm(ym).slice(0, 4)}`,
      onClick: (i, s) => show('date', dayFromBar(ym, i, s)),
    })
  }
  $effect(() => { selection.ym; compare; if (el) load() })
</script>

<div class="nav-day">
  <button onclick={() => (selection.ym = shiftYm(selection.ym, -1))}>‹</button>
  <strong>{selection.ym.slice(4, 6)}/{selection.ym.slice(0, 4)}</strong>
  <button onclick={() => (selection.ym = shiftYm(selection.ym, 1))}>›</button>
  <span>Maandtotaal {kwh(total)} kWh{#if compare} · vorige {kwh(prevTotal)} kWh{/if}</span>
  <label class="cmp"><input type="checkbox" bind:checked={compare} /> vorige maand</label>
</div>
<div class="chart" bind:this={el}></div>
