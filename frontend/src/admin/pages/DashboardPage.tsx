import { useEffect, useState, type ReactNode } from 'react'
import { apiFetch } from '../../api'
import { LineChart, type Series } from '../charts/LineChart'
import { OTHER_COLOR, sourceColors } from '../charts/palette'
import { QualifiedBars, type BarItem } from '../charts/QualifiedBars'
import { CampaignTable } from '../components/CampaignTable'
import { rate, sum } from '../stats'
import { campaignLabel, sourceLabel } from '../format'
import type { CampaignStats } from '../types'

interface WeeklyRow {
  week: string
  source: string
  leads: number
}

const PERIODS = [
  { days: 30, label: '30 Tage' },
  { days: 90, label: '90 Tage' },
  { days: 0, label: 'Gesamt' },
]

// Answers "which campaigns bring leads, and which of them are any good?".
export function DashboardPage() {
  const [days, setDays] = useState(30)
  const [stats, setStats] = useState<{ days: number; rows: CampaignStats[] } | null>(null)
  const [weekly, setWeekly] = useState<{ weeks: string[]; rows: WeeklyRow[] } | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ campaigns: CampaignStats[] }>(`/leads/stats?days=${days}`)
      .then((d) => setStats({ days, rows: d.campaigns }))
      .catch((err) => setError(err.message))
  }, [days])

  useEffect(() => {
    apiFetch<{ weeks: string[]; rows: WeeklyRow[] }>('/leads/timeseries?weeks=8')
      .then(setWeekly)
      .catch((err) => setError(err.message))
  }, [])

  const rows = stats?.days === days ? stats.rows : null
  const periodLabel = PERIODS.find((p) => p.days === days)?.label ?? ''

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <div role="group" aria-label="Zeitraum" className="inline-flex rounded-md bg-white p-0.5 ring-1 ring-neutral-300">
          {PERIODS.map((p) => (
            <button key={p.days} onClick={() => setDays(p.days)} aria-pressed={p.days === days}
              className={`rounded px-3 py-1.5 text-sm font-medium ${p.days === days ? 'bg-ink text-white' : 'text-ink hover:bg-neutral-100'}`}>
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {error && <p role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>}

      <KpiTiles rows={rows} periodLabel={periodLabel} />

      <div className="grid gap-4 lg:grid-cols-[3fr_2fr]">
        <Card eyebrow="Letzte 8 Wochen" title="Leads je Quelle" note="Eindeutige Leads (ohne Duplikate) pro Kalenderwoche.">
          {weekly ? <WeeklyChart weekly={weekly} /> : <Loading />}
        </Card>

        <Card eyebrow={periodLabel === 'Gesamt' ? 'Gesamter Zeitraum' : `Letzte ${periodLabel}`} title="Qualifizierte Leads je Kampagne"
          note="Sortiert nach qualifizierten Leads. Balkenlänge = eindeutige Leads.">
          {rows ? <CampaignBars rows={rows} /> : <Loading />}
        </Card>
      </div>

      <section className="space-y-2">
        <h2 className="text-lg font-semibold">Alle Kampagnen · {periodLabel}</h2>
        {rows ? <CampaignTable rows={rows} /> : <Loading />}
      </section>
    </div>
  )
}

function KpiTiles({ rows, periodLabel }: { rows: CampaignStats[] | null; periodLabel: string }) {
  const t = rows ? sum(rows) : null
  const tiles = [
    { label: 'Anfragen', value: t?.total, hint: 'inkl. Duplikate' },
    { label: 'Eindeutige Leads', value: t?.unique, hint: 'ohne Duplikate' },
    { label: 'Qualifiziert', value: t?.qualified, hint: t ? `${t.notQualified} nicht qualifiziert` : '' },
    { label: 'Quote', value: t ? rate(t) : undefined, hint: 'der bewerteten Leads' },
  ]
  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      {tiles.map((tile) => (
        <div key={tile.label} className="rounded-lg bg-white p-4 ring-1 ring-neutral-200">
          <div className="text-sm text-neutral-600">{tile.label} · {periodLabel}</div>
          <div className="mt-1 text-3xl font-semibold">{tile.value ?? '…'}</div>
          <div className="text-xs text-neutral-500">{tile.hint}</div>
        </div>
      ))}
    </div>
  )
}

function Card({ eyebrow, title, note, children }: { eyebrow: string; title: string; note: string; children: ReactNode }) {
  return (
    <section className="flex min-w-0 flex-col rounded-lg bg-white p-5 ring-1 ring-neutral-200">
      <div className="mb-4">
        <div className="text-xs font-medium tracking-widest text-neutral-500 uppercase">{eyebrow}</div>
        <h2 className="text-xl font-bold">{title}</h2>
      </div>
      <div className="flex-1">{children}</div>
      <p className="mt-4 text-xs text-neutral-500">{note}</p>
    </section>
  )
}

function buildSeries(weekly: { weeks: string[]; rows: WeeklyRow[] }): Series[] {
  const sources = [...new Set(weekly.rows.map((r) => r.source))]
  const colors = sourceColors(sources)
  const index = new Map(weekly.weeks.map((w, i) => [w, i]))

  const byColor = new Map<string, Series>()
  for (const r of weekly.rows) {
    const color = colors.get(r.source) ?? OTHER_COLOR
    const isOther = color === OTHER_COLOR
    const key = isOther ? '__other__' : r.source
    if (!byColor.has(key)) {
      byColor.set(key, { key, label: isOther ? 'Sonstige' : sourceLabel(r.source), color, values: weekly.weeks.map(() => 0) })
    }
    const i = index.get(r.week)
    if (i !== undefined) byColor.get(key)!.values[i] += r.leads
  }
  // Legend order = colour order (Google, Meta, others), "Sonstige" last.
  const order = [...colors.keys()]
  return [...byColor.values()].sort((a, b) =>
    a.key === '__other__' ? 1 : b.key === '__other__' ? -1 : order.indexOf(a.key) - order.indexOf(b.key),
  )
}

function WeeklyChart({ weekly }: { weekly: { weeks: string[]; rows: WeeklyRow[] } }) {
  const series = buildSeries(weekly)
  if (series.length === 0) return <Empty text="Noch keine Leads in den letzten 8 Wochen." />
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-neutral-700">
        {series.map((s) => (
          <span key={s.key} className="flex items-center gap-1.5">
            <span className="inline-block size-2.5 rounded-sm" style={{ background: s.color }} />
            {s.label}
          </span>
        ))}
      </div>
      <LineChart xLabels={weekly.weeks.map(weekLabel)} series={series} />
    </div>
  )
}

function CampaignBars({ rows }: { rows: CampaignStats[] }) {
  const items: BarItem[] = rows
    .filter((r) => r.unique > 0)
    .map((r) => ({
      key: `${r.source}|${r.campaign}`,
      label: campaignLabel(r.campaign),
      sublabel: sourceLabel(r.source),
      unique: r.unique,
      qualified: r.qualified,
    }))
    .sort((a, b) => b.qualified - a.qualified || b.unique - a.unique)
    .slice(0, 8)
  if (items.length === 0) return <Empty text="Keine Leads in diesem Zeitraum." />
  return <QualifiedBars items={items} />
}

// "2026-09-28" (Monday) → "KW 40"
function weekLabel(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number)
  const date = new Date(Date.UTC(y, m - 1, d))
  const thursday = new Date(date)
  thursday.setUTCDate(date.getUTCDate() + 3)
  const firstThursday = new Date(Date.UTC(thursday.getUTCFullYear(), 0, 4))
  const week = 1 + Math.round(((thursday.getTime() - firstThursday.getTime()) / 86_400_000 - 3 + ((firstThursday.getUTCDay() + 6) % 7)) / 7)
  return `KW ${week}`
}

function Loading() {
  return <p className="text-sm text-neutral-500">Lade …</p>
}

function Empty({ text }: { text: string }) {
  return <p className="py-10 text-center text-sm text-neutral-500">{text}</p>
}
