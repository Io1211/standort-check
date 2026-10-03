import { STATUSES, STATUS_LABELS } from '../format'
import type { LeadStatus } from '../types'

const COLORS: Record<LeadStatus, string> = {
  new: 'bg-blue-50 text-blue-800 ring-blue-200',
  contacted: 'bg-amber-50 text-amber-800 ring-amber-200',
  qualified: 'bg-green-50 text-green-800 ring-green-200',
  not_qualified: 'bg-neutral-100 text-neutral-600 ring-neutral-200',
}

export function StatusBadge({ status }: { status: LeadStatus }) {
  return (
    <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${COLORS[status]}`}>
      {STATUS_LABELS[status]}
    </span>
  )
}

// A select styled like the badge, so status can be changed directly in the list.
export function StatusSelect({
  status,
  onChange,
  disabled,
  label,
}: {
  status: LeadStatus
  onChange: (s: LeadStatus) => void
  disabled?: boolean
  label: string
}) {
  return (
    <select
      aria-label={label}
      value={status}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as LeadStatus)}
      className={`cursor-pointer rounded-full py-1 pr-7 pl-2.5 text-xs font-medium ring-1 ring-inset disabled:cursor-wait disabled:opacity-60 ${COLORS[status]}`}
    >
      {STATUSES.map((s) => (
        <option key={s} value={s}>
          {STATUS_LABELS[s]}
        </option>
      ))}
    </select>
  )
}
