import { useEffect, useState, type ReactNode } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { ApiError, apiFetch } from '../../api'
import { StatusBadge, StatusSelect } from '../components/StatusBadge'
import { formatAddress, formatDateTime, sourceLabel } from '../format'
import type { LeadDetail, LeadSummary } from '../types'
import { useStatusUpdate } from '../useStatusUpdate'

export function LeadDetailPage() {
  const { id } = useParams()
  const location = useLocation()
  const backSearch = (location.state as { from?: string } | null)?.from ?? ''
  // The result remembers which id it belongs to, so navigating from one lead
  // to another (duplicate links) never shows the previous lead's data.
  const [result, setResult] = useState<{ id?: string; detail?: LeadDetail; error?: string }>({})
  const current = result.id === id ? result : {}
  const detail = current.detail ?? null
  const error = current.error ?? ''
  const setDetail = (fn: (d: LeadDetail | null) => LeadDetail | null) =>
    setResult((r) => ({ ...r, detail: fn(r.detail ?? null) ?? undefined }))

  useEffect(() => {
    apiFetch<LeadDetail>(`/leads/${id}`)
      .then((d) => setResult({ id, detail: d }))
      .catch((err) =>
        setResult({ id, error: err instanceof ApiError && err.status === 404 ? 'Dieser Lead existiert nicht.' : err.message }),
      )
  }, [id])

  const { updateStatus, saving, error: statusError } = useStatusUpdate((_, status) =>
    setDetail((d) => d && { ...d, lead: { ...d.lead, status } }),
  )

  const back = (
    <Link to={`/admin${backSearch}`} className="text-sm text-neutral-600 hover:underline">← Zurück zur Liste</Link>
  )

  if (error) return <div className="space-y-4">{back}<p className="rounded-md bg-red-50 px-4 py-3 text-red-700">{error}</p></div>
  if (!detail) return <p className="text-sm text-neutral-500">Lade …</p>

  const l = detail.lead
  const address = `${formatAddress(l)}, ${l.postalCode} ${l.city}`

  return (
    <div className="space-y-5">
      {back}

      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-bold">{l.firstName} {l.lastName}</h1>
        <StatusSelect status={l.status} disabled={saving === l.id} label="Status ändern"
          onChange={(s) => updateStatus(l.id, l.status, s)} />
      </div>
      {statusError && <p role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{statusError}</p>}

      {detail.original && (
        <div className="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-800 ring-1 ring-red-200">
          <strong>⚠ Mögliches Duplikat</strong> von{' '}
          <Link to={`/admin/leads/${detail.original.id}`} state={{ from: backSearch }} className="font-semibold underline">
            {detail.original.firstName} {detail.original.lastName}
          </Link>{' '}
          (eingegangen {formatDateTime(detail.original.createdAt)}, Status{' '}
          <StatusBadge status={detail.original.status} />). Gleiche Grundstücksadresse und gleiche E-Mail oder Telefonnummer.
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Section title="Kontakt">
          <Row label="Name">{l.firstName} {l.lastName}</Row>
          <Row label="E-Mail"><a href={`mailto:${l.email}`} className="underline">{l.email}</a></Row>
          <Row label="Telefon"><a href={`tel:${l.phone.replace(/\s/g, '')}`} className="underline">{l.phone}</a></Row>
        </Section>

        <Section title="Grundstück">
          <Row label="Straße">{l.street}</Row>
          <Row label="Hausnummer">{l.houseNumber || '–'}</Row>
          <Row label="PLZ / Ort">{l.postalCode} {l.city}</Row>
          <Row label="Hinweis">{l.parcelNote ? <span className="whitespace-pre-line">{l.parcelNote}</span> : '–'}</Row>
          <Row label="Karte">
            <a href={`https://www.openstreetmap.org/search?query=${encodeURIComponent(address)}`} target="_blank"
              rel="noreferrer" className="underline">In OpenStreetMap öffnen ↗</a>
          </Row>
        </Section>

        <Section title="Herkunft">
          <Row label="Quelle">{sourceLabel(l.utmSource)}</Row>
          <Row label="Medium">{l.utmMedium || '–'}</Row>
          <Row label="Kampagne">{l.utmCampaign || '–'}</Row>
          <Row label="Anzeige (content)">{l.utmContent || '–'}</Row>
          <Row label="Keyword (term)">{l.utmTerm || '–'}</Row>
          <Row label="Klick-ID">{l.gclid ? `gclid: ${l.gclid}` : l.fbclid ? `fbclid: ${l.fbclid}` : '–'}</Row>
          <Row label="Referrer">{l.referrer ? <span className="break-all">{l.referrer}</span> : '–'}</Row>
        </Section>

        <Section title="Lead">
          <Row label="Status"><StatusBadge status={l.status} /></Row>
          <Row label="Eingegangen">{formatDateTime(l.createdAt)}</Row>
          <Row label="Duplikat">{detail.original ? 'Ja, siehe oben' : 'Nein'}</Row>
          <Row label="ID"><span className="font-mono text-xs break-all">{l.id}</span></Row>
        </Section>
      </div>

      {detail.duplicates.length > 0 && (
        <Section title={`Spätere Anfragen zum selben Grundstück (${detail.duplicates.length})`}>
          <ul className="divide-y divide-neutral-100">
            {detail.duplicates.map((d) => <SummaryRow key={d.id} lead={d} backSearch={backSearch} />)}
          </ul>
        </Section>
      )}

      <DeleteLead id={l.id} name={`${l.firstName} ${l.lastName}`} duplicates={detail.duplicates.length} backSearch={backSearch} />
    </div>
  )
}

// Permanent deletion (e.g. GDPR request or test data) with an explicit
// second step, because it cannot be undone.
function DeleteLead({ id, name, duplicates, backSearch }: { id: string; name: string; duplicates: number; backSearch: string }) {
  const navigate = useNavigate()
  const [confirming, setConfirming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState('')

  async function onDelete() {
    setDeleting(true)
    setError('')
    try {
      await apiFetch(`/leads/${id}`, { method: 'DELETE' })
      navigate(`/admin${backSearch}`, { state: { flash: `Lead „${name}“ wurde gelöscht.` } })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Löschen fehlgeschlagen.')
      setDeleting(false)
    }
  }

  return (
    <section className="rounded-lg bg-white p-4 ring-1 ring-red-200">
      <h2 className="mb-2 text-sm font-semibold tracking-wide text-red-700 uppercase">Lead löschen</h2>
      {!confirming ? (
        <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
          <p className="text-neutral-600">Löscht diese Anfrage endgültig, z. B. bei einem Löschantrag nach DSGVO oder für Testdaten.</p>
          <button onClick={() => setConfirming(true)}
            className="rounded-md px-3 py-1.5 font-medium text-red-700 ring-1 ring-red-300 hover:bg-red-50">
            Lead löschen …
          </button>
        </div>
      ) : (
        <div role="alertdialog" aria-labelledby="delete-question" className="space-y-3 text-sm">
          <p id="delete-question">
            <strong>„{name}“ endgültig löschen?</strong> Das kann nicht rückgängig gemacht werden.
            {duplicates > 0 && (
              <> {duplicates === 1 ? 'Die spätere Anfrage wird' : `Die ${duplicates} späteren Anfragen werden`} neu
              zugeordnet und bleiben erhalten.</>
            )}
          </p>
          {error && <p role="alert" className="rounded-md bg-red-50 px-3 py-2 text-red-700">{error}</p>}
          <div className="flex gap-2">
            <button onClick={onDelete} disabled={deleting}
              className="rounded-md bg-red-700 px-3 py-1.5 font-medium text-white hover:bg-red-800 disabled:opacity-60">
              {deleting ? 'Lösche …' : 'Endgültig löschen'}
            </button>
            <button onClick={() => setConfirming(false)} disabled={deleting}
              className="rounded-md px-3 py-1.5 font-medium ring-1 ring-neutral-300 hover:bg-neutral-50">
              Abbrechen
            </button>
          </div>
        </div>
      )}
    </section>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg bg-white p-4 ring-1 ring-neutral-200">
      <h2 className="mb-3 text-sm font-semibold tracking-wide text-neutral-500 uppercase">{title}</h2>
      <dl className="space-y-2 text-sm">{children}</dl>
    </section>
  )
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[9rem_1fr] gap-2">
      <dt className="text-neutral-500">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  )
}

function SummaryRow({ lead, backSearch }: { lead: LeadSummary; backSearch: string }) {
  return (
    <li className="flex flex-wrap items-center gap-3 py-2 text-sm">
      <Link to={`/admin/leads/${lead.id}`} state={{ from: backSearch }} className="font-medium underline">
        {lead.firstName} {lead.lastName}
      </Link>
      <span className="text-neutral-600">{lead.email}</span>
      <span className="text-neutral-600">{formatDateTime(lead.createdAt)}</span>
      <StatusBadge status={lead.status} />
    </li>
  )
}
