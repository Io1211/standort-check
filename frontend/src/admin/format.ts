import type { LeadStatus } from './types'

export const STATUSES: LeadStatus[] = ['new', 'contacted', 'qualified', 'not_qualified']

export const STATUS_LABELS: Record<LeadStatus, string> = {
  new: 'Neu',
  contacted: 'Kontaktiert',
  qualified: 'Qualifiziert',
  not_qualified: 'Nicht qualifiziert',
}

// Same value as the backend's NoValue: filter for leads without source/campaign.
export const NO_VALUE = '__none__'

export function sourceLabel(source: string): string {
  return source || 'Direkt / unbekannt'
}

export function campaignLabel(campaign: string): string {
  return campaign || 'ohne Kampagne'
}

const TZ = 'Europe/Berlin'
const dayFormat = new Intl.DateTimeFormat('de-DE', { timeZone: TZ, year: 'numeric', month: '2-digit', day: '2-digit' })
const timeFormat = new Intl.DateTimeFormat('de-DE', { timeZone: TZ, hour: '2-digit', minute: '2-digit' })

// "Heute, 09:14" / "Gestern, 17:02" / "01.10.2026, 08:30" – in German time,
// regardless of the viewer's time zone.
export function formatReceived(iso: string, now = new Date()): string {
  const date = new Date(iso)
  const day = dayFormat.format(date)
  const time = timeFormat.format(date)
  if (day === dayFormat.format(now)) return `Heute, ${time}`
  if (day === dayFormat.format(new Date(now.getTime() - 86_400_000))) return `Gestern, ${time}`
  return `${day}, ${time}`
}

export function formatDateTime(iso: string): string {
  return `${dayFormat.format(new Date(iso))}, ${timeFormat.format(new Date(iso))} Uhr`
}

export function formatAddress(l: { street: string; houseNumber: string }): string {
  return l.houseNumber ? `${l.street} ${l.houseNumber}` : l.street
}
