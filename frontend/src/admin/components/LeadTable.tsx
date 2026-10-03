import { Link, useLocation } from 'react-router'
import { campaignLabel, formatAddress, formatReceived, sourceLabel } from '../format'
import type { AdminLead, LeadStatus } from '../types'
import { DuplicateBadge } from './DuplicateBadge'
import { AreaBadge, GeoWarnings } from './GeoBadge'
import { StatusSelect } from './StatusBadge'

export type SortKey = 'created_at' | 'name' | 'city' | 'status'

interface Props {
  leads: AdminLead[]
  sort: SortKey
  desc: boolean
  onSort: (key: SortKey) => void
  onStatusChange: (lead: AdminLead, status: LeadStatus) => void
  savingId: string | null
}

// Desktop: table. Phone: one card per lead (a 6-column table does not fit).
export function LeadTable(props: Props) {
  return (
    <>
      <DesktopTable {...props} />
      <MobileCards {...props} />
    </>
  )
}

function DesktopTable({ leads, sort, desc, onSort, onStatusChange, savingId }: Props) {
  const location = useLocation()

  const header = (key: SortKey | null, label: string) => (
    <th scope="col" className="px-4 py-3 text-left font-semibold"
      aria-sort={key === sort ? (desc ? 'descending' : 'ascending') : undefined}>
      {key ? (
        <button onClick={() => onSort(key)} className="inline-flex items-center gap-1 hover:underline">
          {label}
          <span aria-hidden="true" className={key === sort ? '' : 'invisible'}>{desc ? '↓' : '↑'}</span>
        </button>
      ) : label}
    </th>
  )

  return (
    <div className="hidden overflow-x-auto rounded-lg bg-white ring-1 ring-neutral-200 md:block">
      <table className="w-full text-sm">
        <thead className="border-b border-neutral-200 bg-neutral-50 text-neutral-700">
          <tr>
            {header('name', 'Name / Kontakt')}
            {header('city', 'Grundstück')}
            {header(null, 'Herkunft')}
            {header('status', 'Status')}
            {header('created_at', 'Eingang')}
          </tr>
        </thead>
        <tbody className="divide-y divide-neutral-100">
          {leads.map((l) => (
            <tr key={l.id} className={l.duplicateOf ? 'bg-red-50/40' : 'hover:bg-neutral-50'}>
              <td className="px-4 py-3 align-top">
                <Link to={`/admin/leads/${l.id}`} state={{ from: location.search }} className="font-semibold hover:underline">
                  {l.firstName} {l.lastName}
                </Link>
                <div className="text-neutral-600">{l.email}</div>
                <div className="text-neutral-600">{l.phone}</div>
              </td>
              <td className="px-4 py-3 align-top">
                <div>{formatAddress(l)}</div>
                <div className="text-neutral-600">{l.postalCode} {l.city}</div>
                {l.geo.municipality && l.geo.municipality !== l.city && (
                  <div className="text-xs text-neutral-500">Gemeinde {l.geo.municipality}</div>
                )}
                <div className="mt-1 flex flex-wrap gap-1">
                  <AreaBadge geo={l.geo} />
                  <GeoWarnings geo={l.geo} postalCode={l.postalCode} />
                  <DuplicateBadge duplicateOf={l.duplicateOf} duplicateCount={l.duplicateCount} />
                </div>
              </td>
              <td className="px-4 py-3 align-top">
                <div>{sourceLabel(l.utmSource)}</div>
                <div className="text-neutral-600">{campaignLabel(l.utmCampaign)}</div>
              </td>
              <td className="px-4 py-3 align-top">
                <StatusSelect status={l.status} disabled={savingId === l.id} label={`Status von ${l.firstName} ${l.lastName}`}
                  onChange={(s) => onStatusChange(l, s)} />
              </td>
              <td className="px-4 py-3 align-top whitespace-nowrap text-neutral-700">{formatReceived(l.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function MobileCards({ leads, onStatusChange, savingId }: Props) {
  const location = useLocation()
  return (
    <ul className="space-y-3 md:hidden">
      {leads.map((l) => (
        <li key={l.id} className={`rounded-lg p-4 ring-1 ${l.duplicateOf ? 'bg-red-50/40 ring-red-200' : 'bg-white ring-neutral-200'}`}>
          <div className="flex items-start justify-between gap-3">
            <Link to={`/admin/leads/${l.id}`} state={{ from: location.search }} className="font-semibold hover:underline">
              {l.firstName} {l.lastName}
            </Link>
            <span className="shrink-0 text-xs text-neutral-600">{formatReceived(l.createdAt)}</span>
          </div>
          <div className="mt-1 text-sm text-neutral-700">
            {formatAddress(l)}, {l.postalCode} {l.city}
          </div>
          <div className="text-sm text-neutral-600">
            <a href={`tel:${l.phone.replace(/\s/g, '')}`} className="underline">{l.phone}</a>
            {' · '}{sourceLabel(l.utmSource)}
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <StatusSelect status={l.status} disabled={savingId === l.id} label={`Status von ${l.firstName} ${l.lastName}`}
              onChange={(s) => onStatusChange(l, s)} />
            <AreaBadge geo={l.geo} />
            <GeoWarnings geo={l.geo} postalCode={l.postalCode} />
            <DuplicateBadge duplicateOf={l.duplicateOf} duplicateCount={l.duplicateCount} />
          </div>
        </li>
      ))}
    </ul>
  )
}
