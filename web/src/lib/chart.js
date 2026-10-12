import * as echarts from 'echarts'

function cssVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

export function mountLine(el, { xLabels, series, legend }) {
  const c = echarts.init(el)
  c.setOption({
    grid: { left: 48, right: 16, top: legend ? 32 : 24, bottom: 32 },
    xAxis: { type: 'category', data: xLabels, axisLine: { lineStyle: { color: cssVar('--grid') } }, axisLabel: { color: cssVar('--muted') } },
    yAxis: { type: 'value', name: 'W', splitLine: { lineStyle: { color: cssVar('--grid') } }, axisLabel: { color: cssVar('--muted') } },
    tooltip: { trigger: 'axis' },
    legend: legend ? { top: 0, textStyle: { color: cssVar('--fg') } } : undefined,
    series: series.map((s) => ({
      type: 'line', name: s.name, data: s.data, smooth: true, showSymbol: false,
      connectNulls: s.connectNulls ?? false,
      areaStyle: s.area ? { opacity: s.areaOpacity ?? 0.7 } : undefined,
      lineStyle: { color: s.color, type: s.dashed ? 'dashed' : 'solid', width: s.width ?? 2 },
      itemStyle: { color: s.color },
      z: s.z ?? 2,
    })),
  })
  return c
}

export function mountBars(el, { xLabels, values, name, compareValues, compareName, onClick }) {
  const c = echarts.init(el)
  const series = [{ type: 'bar', name: name || 'kWh', data: values, itemStyle: { color: cssVar('--accent') } }]
  if (compareValues) {
    series.push({
      type: 'bar', name: compareName || 'vorige', data: compareValues,
      itemStyle: { color: cssVar('--muted') },
    })
  }
  c.setOption({
    grid: { left: 48, right: 16, top: compareValues ? 32 : 24, bottom: 32 },
    xAxis: { type: 'category', data: xLabels, axisLine: { lineStyle: { color: cssVar('--grid') } }, axisLabel: { color: cssVar('--muted') } },
    yAxis: { type: 'value', name: 'kWh', splitLine: { lineStyle: { color: cssVar('--grid') } }, axisLabel: { color: cssVar('--muted') } },
    tooltip: { trigger: 'axis' },
    legend: compareValues ? { top: 0, textStyle: { color: cssVar('--fg') } } : undefined,
    series,
  })
  if (onClick) c.on('click', (p) => onClick(p.dataIndex, p.seriesIndex))
  return c
}
