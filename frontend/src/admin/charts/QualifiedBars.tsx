import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { TooltipContentProps } from 'recharts'
import { BLUE_LIGHT, BLUE_STRONG } from './palette'

export interface BarItem {
  key: string
  label: string
  sublabel: string
  unique: number
  qualified: number
}

const ROW_HEIGHT = 44

// Horizontal stacked bars (Recharts): total length = unique leads of the
// campaign, dark part = qualified. Answers "how many leads, and how many are
// good" at a glance. A 2px white edge separates the two segments.
export function QualifiedBars({ items }: { items: BarItem[] }) {
  const max = Math.max(1, ...items.map((i) => i.unique))
  const data = items.map((i) => ({ ...i, rest: i.unique - i.qualified, track: max }))
  const byKey = new Map(data.map((d) => [d.key, d]))

  return (
    <div className="space-y-3">
      <div className="flex gap-4 text-xs text-neutral-600">
        <Legend color={BLUE_STRONG} label="qualifiziert" />
        <Legend color={BLUE_LIGHT} label="übrige Leads" />
      </div>
      <ResponsiveContainer width="100%" height={items.length * ROW_HEIGHT + 8}>
        <BarChart data={data} layout="vertical" margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barSize={20}>
          <XAxis type="number" hide domain={[0, max]} />
          {/* Grey track behind every row (own hidden axis, so it overlaps the data bars). */}
          <YAxis yAxisId="track" type="category" dataKey="key" hide />
          <Bar yAxisId="track" dataKey="track" fill="#f0efec" radius={4} isAnimationActive={false} activeBar={false} />
          {/* Ticks look items up by key, never by position: Recharts skips
              zero-length bars, so indexes of bar labels can shift. */}
          <YAxis yAxisId="names" type="category" dataKey="key" width={170} tickLine={false} axisLine={false}
            tick={(p: { x?: number | string; y?: number | string; payload?: { value: string } }) => {
              const item = byKey.get(p.payload?.value ?? '')
              if (!item) return <g />
              return (
                <g transform={`translate(${p.x},${p.y})`}>
                  <text x={-10} y={-3} textAnchor="end" fontSize={12} fontWeight={500} fill="#262626">
                    {truncate(item.label, 22)}
                  </text>
                  <text x={-10} y={12} textAnchor="end" fontSize={11} fill="#737373">
                    {item.sublabel}
                  </text>
                </g>
              )
            }} />
          <YAxis yAxisId="values" orientation="right" type="category" dataKey="key" width={52} tickLine={false} axisLine={false}
            tick={(p: { x?: number | string; y?: number | string; payload?: { value: string } }) => {
              const item = byKey.get(p.payload?.value ?? '')
              if (!item) return <g />
              return (
                <text x={Number(p.x) + 10} y={p.y} dy="0.32em" fontSize={12} fill="#404040">
                  <tspan fontWeight={600} fill="#171717">{item.qualified}</tspan> / {item.unique}
                </text>
              )
            }} />
          <Tooltip content={TooltipBox} cursor={{ fill: '#f5f5f4' }} />
          <Bar yAxisId="names" dataKey="qualified" stackId="leads" fill={BLUE_STRONG} stroke="#fff" strokeWidth={2}
            isAnimationActive={false} />
          <Bar yAxisId="names" dataKey="rest" stackId="leads" fill={BLUE_LIGHT} stroke="#fff" strokeWidth={2}
            radius={[0, 4, 4, 0]} isAnimationActive={false} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}

function TooltipBox({ active, payload }: TooltipContentProps) {
  const item = payload?.[0]?.payload as BarItem | undefined
  if (!active || !item) return null
  return (
    <div className="rounded-md bg-white px-3 py-2 text-xs shadow-lg ring-1 ring-neutral-200">
      <div className="font-semibold text-neutral-700">{item.sublabel} · {item.label}</div>
      <div className="text-neutral-600">
        <strong className="text-neutral-900">{item.qualified}</strong> von {item.unique} Leads qualifiziert
      </div>
    </div>
  )
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className="inline-block size-2.5 rounded-sm" style={{ background: color }} />
      {label}
    </span>
  )
}

function truncate(s: string, max: number): string {
  return s.length > max ? `${s.slice(0, max - 1)}…` : s
}
