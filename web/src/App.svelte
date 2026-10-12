<script>
  import './app.css'
  import Day from './views/Day.svelte'
  import Month from './views/Month.svelte'
  import Year from './views/Year.svelte'
  import AllYears from './views/AllYears.svelte'
  import Top from './views/Top.svelte'
  import Yield from './views/Yield.svelte'
  import { getSummary } from './lib/api.js'

  const VERSION = '3.0'
  let panel = $state(null) // null | 'top' | 'yield'
  let updated = $state('')

  getSummary().then((s) => {
    if (s.last_update) {
      updated = new Date(s.last_update * 1000).toLocaleString('nl-BE', {
        timeZone: 'Europe/Brussels',
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit',
      })
    }
  })

  function toggle(name) {
    panel = panel === name ? null : name
  }
</script>

<div class="container">
  <header>
    <h1>PVdiary</h1>
    <nav>
      <button class:active={panel === 'top'} onclick={() => toggle('top')}>Top</button>
      <button class:active={panel === 'yield'} onclick={() => toggle('yield')}>Opbrengst &amp; Milieu</button>
    </nav>
  </header>

  {#if panel === 'top'}
    <section class="card"><Top /></section>
  {:else if panel === 'yield'}
    <section class="card"><Yield /></section>
  {/if}

  <section class="card" id="view-day"><Day /></section>
  <section class="card" id="view-month"><Month /></section>
  <div class="row">
    <section class="card half" id="view-year"><Year /></section>
    <section class="card half"><AllYears /></section>
  </div>

  <footer>
    <span>PVdiary {VERSION}</span>
    {#if updated}<span>Updated: {updated}</span>{/if}
  </footer>
</div>
