import { useCallback, useEffect, useState } from 'react'
import { useLocation, useSearchParams } from 'react-router'
import { apiFetch } from '../../api'
import { LeadFilters, type Filters } from '../components/LeadFilters'
import { LeadTable, type SortKey } from '../components/LeadTable'
import type { AdminLead, CampaignStats, LeadListResponse, LeadStatus } from '../types'
import { useStatusUpdate } from '../useStatusUpdate'

// All filters live in the URL: reload, back button and shared links keep them,
// and the CSV export uses exactly the same query.
export function LeadsPage() {
  const [params, setParams] = useSearchParams()
  const flash = (useLocation().state as { flash?: string } | null)?.flash
  const [data, setData] = useState<LeadListResponse | null>(null)
  const [campaigns, setCampaigns] = useState<CampaignStats[]>([])
  const [loadError, setLoadError] = useState('')

  const filters: Filters = {
    search: params.get('search') ?? '',
    status: params.get('status') ?? '',
    source: params.get('source') ?? '',
    campaign: params.get('campaign') ?? '',
    hideDuplicates: params.get('hideDuplicates') === 'true',
  }
  const sort = (params.get('sort') ?? 'created_at') as SortKey
  const desc = params.get('order') !== 'asc'
  const page = Number(params.get('page') ?? '1')

  const query = params.toString()

  useEffect(() => {
    let cancelled = false
    apiFetch<LeadListResponse>(`/leads?${query}`)
      .then((d) => !cancelled && (setData(d), setLoadError('')))
      .catch((err) => !cancelled && setLoadError(err.message))
    return () => {
      cancelled = true
    }
  }, [query])

  useEffect(() => {
    apiFetch<{ campaigns: CampaignStats[] }>('/leads/stats')
      .then((d) => setCampaigns(d.campaigns))
      .catch(() => {}) // only used for the filter dropdowns
  }, [])

  const update = useCallback(
    (patch: Record<string, string | boolean | number>) => {
      setParams((prev) => {
        const next = new URLSearchParams(prev)
        for (const [k, v] of Object.entries(patch)) {
          if (v === '' || v === false || (k === 'page' && v === 1)) next.delete(k)
          else next.set(k, String(v))
        }
        if (!('page' in patch)) next.delete('page') // new filter → back to page 1
        return next
      })
    },
    [setParams],
  )

  const onFilterChange = useCallback((patch: Partial<Filters>) => update(patch), [update])

  function onSort(key: SortKey) {
    // Same column toggles direction; a new column starts with a sensible default.
    if (key === sort) update({ order: desc ? 'asc' : '' })
    else update({ sort: key === 'created_at' ? '' : key, order: key === 'created_at' ? '' : 'asc' })
  }

  const { updateStatus, saving, error: statusError } = useStatusUpdate((id, status) =>
    setData((d) => d && { ...d, leads: d.leads.map((l) => (l.id === id ? { ...l, status } : l)) }),
  )
  const onStatusChange = (lead: AdminLead, status: LeadStatus) => updateStatus(lead.id, lead.status, status)

  const exportParams = new URLSearchParams(params)
  exportParams.delete('page')

  const from = data ? (data.page - 1) * data.pageSize + 1 : 0
  const to = data ? Math.min(data.page * data.pageSize, data.total) : 0

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Leads</h1>
          <p className="text-sm text-neutral-600">{data ? `${data.total} Anfragen` : ' '}</p>
        </div>
        <a href={`/api/leads/export?${exportParams}`} download
          className="rounded-md bg-ink px-4 py-2 text-sm font-medium text-white hover:bg-ink/90">
          CSV exportieren
        </a>
      </div>

      <LeadFilters filters={filters} campaigns={campaigns} onChange={onFilterChange} />

      {flash && <p role="status" className="rounded-md bg-green-50 px-4 py-3 text-sm text-green-800">{flash}</p>}
      {loadError && <p role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{loadError}</p>}
      {statusError && <p role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{statusError}</p>}

      {data && data.leads.length === 0 && (
        <p className="rounded-lg bg-white px-4 py-10 text-center text-neutral-600 ring-1 ring-neutral-200">
          Keine Anfragen gefunden.
        </p>
      )}
      {data && data.leads.length > 0 && (
        <>
          <LeadTable leads={data.leads} sort={sort} desc={desc} onSort={onSort} onStatusChange={onStatusChange} savingId={saving} />
          <div className="flex items-center justify-between text-sm text-neutral-600">
            <span>{from}–{to} von {data.total}</span>
            <div className="flex gap-2">
              <button disabled={page <= 1} onClick={() => update({ page: page - 1 })}
                className="rounded-md px-3 py-1.5 ring-1 ring-neutral-300 hover:bg-white disabled:opacity-40">Zurück</button>
              <button disabled={to >= data.total} onClick={() => update({ page: page + 1 })}
                className="rounded-md px-3 py-1.5 ring-1 ring-neutral-300 hover:bg-white disabled:opacity-40">Weiter</button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
