import { CartesianGrid, Line, LineChart as RLineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { TooltipContentProps } from 'recharts'

export interface Series {
  key: string
  label: string
  color: string
  values: number[]
}

const HEIGHT = 240
const MARGIN = { top: 12, right: 130, bottom: 4, left: 0 }
const X_AXIS_HEIGHT = 24

// Multi-series line chart (Recharts): 2px lines, an end dot with a white ring,
// the series name at the line end, and a tooltip listing every series.
export function LineChart({ xLabels, series }: { xLabels: string[]; series: Series[] }) {
  const max = niceMax(Math.max(1, ...series.flatMap((s) => s.values)))
  const data = xLabels.map((label, i) => ({
    label,
    ...Object.fromEntries(series.map((s) => [s.key, s.values[i] ?? 0])),
  }))
  const labelOffsets = endLabelOffsets(series, max)
  const last = xLabels.length - 1

  return (
    <ResponsiveContainer width="100%" height={HEIGHT}>
      <RLineChart data={data} margin={MARGIN}>
        <CartesianGrid vertical={false} stroke="#e7e6e2" />
        <XAxis dataKey="label" tickLine={false} axisLine={false} height={X_AXIS_HEIGHT}
          tick={{ fontSize: 11, fill: '#737373' }} interval="preserveStartEnd" />
        <YAxis domain={[0, max]} ticks={[0, max / 2, max]} allowDecimals={false} tickLine={false} axisLine={false}
          width={36} tick={{ fontSize: 11, fill: '#737373' }} />
        <Tooltip content={TooltipBox} cursor={{ stroke: '#b5b4ae', strokeWidth: 1 }} />
        {series.map((s) => (
          <Line key={s.key} dataKey={s.key} name={s.label} stroke={s.color} strokeWidth={2} isAnimationActive={false}
            strokeLinecap="round" strokeLinejoin="round"
            activeDot={{ r: 4, fill: s.color, stroke: '#fff', strokeWidth: 2 }}
            dot={(p: { cx?: number; cy?: number; index?: number }) =>
              p.index === last ? (
                <circle key={`dot-${s.key}`} cx={p.cx} cy={p.cy} r={4} fill={s.color} stroke="#fff" strokeWidth={2} />
              ) : (
                <g key={`dot-${s.key}-${p.index}`} />
              )
            }
            label={(p: { x?: number | string; y?: number | string; index?: number }) =>
              p.index === last ? (
                <text key={`label-${s.key}`} x={Number(p.x) + 10} y={Number(p.y) + (labelOffsets.get(s.key) ?? 0)}
                  dy="0.32em" fontSize={12} fill="#404040">
                  {truncate(s.label, 18)}
                </text>
              ) : (
                <g key={`label-${s.key}-${p.index}`} />
              )
            }
          />
        ))}
      </RLineChart>
    </ResponsiveContainer>
  )
}

function TooltipBox({ active, label, payload }: TooltipContentProps) {
  if (!active || !payload?.length) return null
  const rows = [...payload].sort((a, b) => Number(b.value) - Number(a.value))
  return (
    <div className="min-w-36 rounded-md bg-white px-3 py-2 text-xs shadow-lg ring-1 ring-neutral-200">
      <div className="mb-1 font-semibold text-neutral-700">{label}</div>
      {rows.map((r) => (
        <div key={String(r.dataKey)} className="flex items-center justify-between gap-4">
          <span className="flex items-center gap-2 text-neutral-600">
            <span className="inline-block h-0.5 w-3 rounded" style={{ background: r.color }} />
            {r.name}
          </span>
          <strong className="tabular-nums text-neutral-900">{r.value}</strong>
        </div>
      ))}
    </div>
  )
}

// End labels of series with (nearly) the same last value would sit on top of
// each other. Compute their pixel positions and push them apart by ≥ 14px.
function endLabelOffsets(series: Series[], max: number): Map<string, number> {
  const plotH = HEIGHT - MARGIN.top - MARGIN.bottom - X_AXIS_HEIGHT
  const items = series
    .map((s) => {
      const y = plotH * (1 - (s.values[s.values.length - 1] ?? 0) / max)
      return { key: s.key, y, placed: y }
    })
    .sort((a, b) => a.y - b.y)
  for (let i = 1; i < items.length; i++) {
    items[i].placed = Math.max(items[i].y, items[i - 1].placed + 14)
  }
  return new Map(items.map((it) => [it.key, it.placed - it.y]))
}

// Rounds the axis maximum up to a clean value (5 → 6, 13 → 20, 170 → 200).
function niceMax(v: number): number {
  if (v <= 4) return Math.max(2, Math.ceil(v / 2) * 2)
  const magnitude = 10 ** Math.floor(Math.log10(v))
  for (const step of [1, 2, 2.5, 5, 10]) {
    if (step * magnitude >= v) return step * magnitude
  }
  return 10 * magnitude
}

function truncate(s: string, max: number): string {
  return s.length > max ? `${s.slice(0, max - 1)}…` : s
}
