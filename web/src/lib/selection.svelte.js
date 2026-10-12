import { todayBrussels } from './format.js'

// Gedeelde periodeselectie. Een klik in een grafiek zet hier de doelperiode van
// de view erboven; die view reageert via zijn eigen $effect.
export const selection = $state({
  date: '', // wordt door Day gezet zodra de laatste dag met data bekend is
  ym: todayBrussels().slice(0, 6),
  year: +todayBrussels().slice(0, 4),
})

const CARD = { date: 'view-day', ym: 'view-month', year: 'view-year' }

// Selecteer een periode en breng de bijbehorende kaart in beeld.
export function show(key, value) {
  if (value === null) return
  selection[key] = value
  document.getElementById(CARD[key])?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}
