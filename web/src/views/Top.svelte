<script>
  import { getTop } from '../lib/api.js'
  import { dateNL, kwh } from '../lib/format.js'

  let scope = $state('global')
  let key = $state('')
  let data = $state({ days: [], peak: [] })

  async function load() { data = await getTop(scope, key, 10) }
  $effect(() => { scope; key; load() })
</script>

<div class="nav-day">
  <label>Scope
    <select bind:value={scope} onchange={() => (key = '')}>
      <option value="global">Globaal</option>
      <option value="year">Jaar</option>
      <option value="month">Maand</option>
    </select>
  </label>
  {#if scope !== 'global'}
    <input bind:value={key} placeholder={scope === 'year' ? '2025' : '06'} />
  {/if}
</div>

<div class="top-cols">
  <div>
    <h3>Top dagen (kWh)</h3>
    <ol>{#each data.days as d}<li>{dateNL(d.Date)} — {kwh(d.Value)} kWh</li>{/each}</ol>
  </div>
  <div>
    <h3>Top piek (W)</h3>
    <ol>{#each data.peak as d}<li>{dateNL(d.Date)} — {d.Value} W</li>{/each}</ol>
  </div>
</div>
