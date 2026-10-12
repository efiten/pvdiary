<script>
  import { getSummary } from '../lib/api.js'

  let rows = $state([])
  let cfg = $state({ ele_cost: 0.18, co2: 0.596 })

  async function load() {
    const s = await getSummary()
    cfg = s.config
    rows = Object.keys(s.kwh_year || {}).sort().map((yr) => {
      const kwh = s.kwh_year[yr].reduce((a, b) => a + b, 0)
      return { yr, kwh, euro: kwh * cfg.ele_cost, co2: kwh * cfg.co2 }
    })
  }
  load()
</script>

<h3>Opbrengst &amp; Milieu</h3>
<p>Tarief {cfg.ele_cost} €/kWh · {cfg.co2} kg CO₂/kWh vermeden</p>
<table>
  <thead><tr><th>Jaar</th><th>kWh</th><th>Bespaard €</th><th>CO₂ vermeden (kg)</th></tr></thead>
  <tbody>
    {#each rows as r}
      <tr><td>{r.yr}</td><td>{r.kwh.toFixed(0)}</td><td>{r.euro.toFixed(0)}</td><td>{r.co2.toFixed(0)}</td></tr>
    {/each}
  </tbody>
</table>
