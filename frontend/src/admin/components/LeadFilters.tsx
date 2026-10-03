import { useEffect, useState } from 'react'
import { NO_VALUE, STATUSES, STATUS_LABELS, campaignLabel, sourceLabel } from '../format'
import type { CampaignStats } from '../types'

export interface Filters {
  search: string
  status: string
  source: string
  campaign: string
  hideDuplicates: boolean
}

export function LeadFilters({
  filters,
  campaigns,
  onChange,
}: {
  filters: Filters
  campaigns: CampaignStats[]
  onChange: (patch: Partial<Filters>) => void
}) {
  // Search is debounced so we don't query on every keystroke.
  const [search, setSearch] = useState(filters.search)
  // Follow external changes (e.g. back button) without an extra effect.
  const [urlSearch, setUrlSearch] = useState(filters.search)
  if (urlSearch !== filters.search) {
    setUrlSearch(filters.search)
    setSearch(filters.search)
  }
  useEffect(() => {
    if (search === filters.search) return
    const t = setTimeout(() => onChange({ search }), 300)
    return () => clearTimeout(t)
  }, [search, filters.search, onChange])

  const sources = unique(campaigns.map((c) => c.source))
  const campaignOptions = unique(
    campaigns.filter((c) => !filters.source || c.source === (filters.source === NO_VALUE ? '' : filters.source)).map((c) => c.campaign),
  )

  const select = 'rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-ink'

  return (
    <div className="flex flex-wrap items-center gap-3">
      <input type="search" value={search} onChange={(e) => setSearch(e.target.value)} aria-label="Suche"
        placeholder="Suche: Name, E-Mail, Telefon, Ort …"
        className="min-w-0 flex-1 basis-64 rounded-md border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-ink" />

      <select aria-label="Status" value={filters.status} onChange={(e) => onChange({ status: e.target.value })} className={select}>
        <option value="">Alle Status</option>
        {STATUSES.map((s) => <option key={s} value={s}>{STATUS_LABELS[s]}</option>)}
      </select>

      <select aria-label="Quelle" value={filters.source} onChange={(e) => onChange({ source: e.target.value, campaign: '' })} className={select}>
        <option value="">Alle Quellen</option>
        {sources.map((s) => <option key={s} value={s || NO_VALUE}>{sourceLabel(s)}</option>)}
      </select>

      <select aria-label="Kampagne" value={filters.campaign} onChange={(e) => onChange({ campaign: e.target.value })} className={select}>
        <option value="">Alle Kampagnen</option>
        {campaignOptions.map((c) => <option key={c} value={c || NO_VALUE}>{campaignLabel(c)}</option>)}
      </select>

      <label className="flex items-center gap-2 text-sm">
        <input type="checkbox" checked={filters.hideDuplicates} onChange={(e) => onChange({ hideDuplicates: e.target.checked })}
          className="size-4 accent-ink" />
        Duplikate ausblenden
      </label>
    </div>
  )
}

function unique(values: string[]): string[] {
  return [...new Set(values)].sort((a, b) => (a === '' ? 1 : b === '' ? -1 : a.localeCompare(b, 'de')))
}
