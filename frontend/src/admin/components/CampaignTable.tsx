import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { NO_VALUE, campaignLabel, sourceLabel } from '../format'
import { rate, sum } from '../stats'
import type { CampaignStats } from '../types'

// Campaign evaluation as a table: per source (with subtotal) and campaign.
// Status counts use unique leads only (duplicates excluded).
export function CampaignTable({ rows }: { rows: CampaignStats[] }) {
  const bySource = groupBySource(rows)
  const total = sum(rows)

  return (
    <div className="space-y-2">
      <div className="overflow-x-auto rounded-lg bg-white ring-1 ring-neutral-200">
        <table className="w-full text-sm">
          <thead className="border-b border-neutral-200 bg-neutral-50 text-left text-neutral-700">
            <tr>
              <th className="px-4 py-3 font-semibold">Quelle / Kampagne</th>
              <Num head>Anfragen</Num>
              <Num head>Eindeutig</Num>
              <Num head>Neu</Num>
              <Num head>Kontaktiert</Num>
              <Num head>Qualifiziert</Num>
              <Num head>Nicht qual.</Num>
              <Num head>Quote</Num>
            </tr>
          </thead>
          <tbody>
            {bySource.map(([source, campaigns]) => (
              <SourceGroup key={source} source={source} campaigns={campaigns} />
            ))}
            {rows.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-10 text-center text-neutral-600">Noch keine Anfragen.</td></tr>
            )}
          </tbody>
          {rows.length > 0 && (
            <tfoot className="border-t-2 border-neutral-300 font-semibold">
              <StatsRow label="Gesamt" s={total} />
            </tfoot>
          )}
        </table>
      </div>
      <p className="max-w-3xl text-xs text-neutral-600">
        <strong>Quote</strong> = qualifiziert ÷ bereits bewertete Leads (qualifiziert + nicht qualifiziert). Neue und
        kontaktierte Leads zählen nicht in die Quote. Duplikate sind nur in „Anfragen“ enthalten.
      </p>
    </div>
  )
}

function SourceGroup({ source, campaigns }: { source: string; campaigns: CampaignStats[] }) {
  const sourceParam = source || NO_VALUE
  return (
    <>
      <StatsRow
        className="border-t border-neutral-200 bg-neutral-50 font-semibold"
        label={<Link to={`/admin?source=${encodeURIComponent(sourceParam)}`} className="hover:underline">{sourceLabel(source)}</Link>}
        s={sum(campaigns)}
      />
      {campaigns.map((c) => (
        <StatsRow
          key={c.campaign}
          className="border-t border-neutral-100"
          label={
            <Link className="pl-4 text-neutral-700 hover:underline"
              to={`/admin?source=${encodeURIComponent(sourceParam)}&campaign=${encodeURIComponent(c.campaign || NO_VALUE)}`}>
              {campaignLabel(c.campaign)}
            </Link>
          }
          s={c}
        />
      ))}
    </>
  )
}

function StatsRow({ label, s, className = '' }: { label: ReactNode; s: Omit<CampaignStats, 'source' | 'campaign'>; className?: string }) {
  return (
    <tr className={className}>
      <td className="px-4 py-2.5">{label}</td>
      <Num>{s.total}</Num>
      <Num>{s.unique}</Num>
      <Num>{s.new}</Num>
      <Num>{s.contacted}</Num>
      <Num>{s.qualified}</Num>
      <Num>{s.notQualified}</Num>
      <Num>{rate(s)}</Num>
    </tr>
  )
}

function Num({ children, head }: { children: ReactNode; head?: boolean }) {
  const cls = 'px-4 py-2.5 text-right tabular-nums'
  return head ? <th className={`${cls} font-semibold`}>{children}</th> : <td className={cls}>{children}</td>
}

function groupBySource(rows: CampaignStats[]): [string, CampaignStats[]][] {
  const map = new Map<string, CampaignStats[]>()
  for (const r of rows) map.set(r.source, [...(map.get(r.source) ?? []), r])
  // Most leads first, "direct / unknown" last.
  return [...map.entries()].sort(([a, ra], [b, rb]) =>
    a === '' ? 1 : b === '' ? -1 : sum(rb).unique - sum(ra).unique,
  )
}
