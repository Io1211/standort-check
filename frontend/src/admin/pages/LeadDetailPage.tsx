import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { ApiError, apiFetch } from '../../api'
import { AreaBadge, GeoWarnings } from '../components/GeoBadge'
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
            <a href={mapLink(l.geo.lat, l.geo.lon, address)} target="_blank"
              rel="noreferrer" className="underline">In OpenStreetMap öffnen ↗</a>
          </Row>
        </Section>

        <GeoSection lead={l} onUpdated={(d) => setResult({ id, detail: d })} />

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

// Geo data from Geoapify: where the plot is and whether we serve that area.
function GeoSection({ lead: l, onUpdated }: { lead: LeadDetail['lead']; onUpdated: (d: LeadDetail) => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const g = l.geo

  async function refresh(automatic = false) {
    setBusy(true)
    setError('')
    try {
      onUpdated(await apiFetch<LeadDetail>(`/leads/${l.id}/geocode`, { method: 'POST' }))
    } catch (err) {
      // An automatic attempt stays quiet if geo is simply not configured.
      if (!(automatic && err instanceof ApiError && err.status === 503)) {
        setError(err instanceof Error ? err.message : 'Abruf fehlgeschlagen.')
      }
    } finally {
      setBusy(false)
    }
  }

  // Geoapify can take longer than the form submission waits. When sales
  // opens a lead without geo data, fetch it now (once per lead).
  const attempted = useRef<string | null>(null)
  useEffect(() => {
    if ((g.status === '' || g.status === 'error') && attempted.current !== l.id) {
      attempted.current = l.id
      void refresh(true)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- run once per lead id
  }, [l.id])

  return (
    <Section title="Lage & Einzugsgebiet">
      <Row label="Einzugsgebiet">
        <span className="flex flex-wrap gap-1">
          <AreaBadge geo={g} />
          <GeoWarnings geo={g} postalCode={l.postalCode} />
          {g.status === 'ok' && g.inServiceArea === null && <span className="text-neutral-500">nicht konfiguriert</span>}
        </span>
      </Row>
      {g.status === 'ok' && (
        <>
          <Row label="Gemeinde">{g.municipality || '–'}</Row>
          <Row label="Landkreis">{g.county || '–'}</Row>
          <Row label="Bundesland">{g.state || '–'}</Row>
          <Row label="Gefunden als">{g.formatted || '–'}</Row>
          <Row label="Koordinaten">
            {g.lat !== null && g.lon !== null ? `${g.lat.toFixed(5)}, ${g.lon.toFixed(5)}` : '–'}
            {g.imprecise && <span className="text-neutral-500"> (ungefähr: {resultTypeLabel(g.resultType)})</span>}
          </Row>
        </>
      )}
      {g.status === 'not_found' && (
        <p className="text-neutral-600">Zu dieser Adresse wurde in Deutschland nichts gefunden – evtl. Tippfehler oder ausgedachte Adresse.</p>
      )}
      <Row label="Geprüft">{g.checkedAt ? formatDateTime(g.checkedAt) : 'noch nicht'}</Row>
      <div className="pt-1">
        <button onClick={() => refresh()} disabled={busy}
          className="rounded-md px-3 py-1.5 text-sm font-medium ring-1 ring-neutral-300 hover:bg-neutral-50 disabled:opacity-60">
          {busy ? 'Rufe Geodaten ab … (kann bis zu 15 s dauern)' : 'Geodaten neu abrufen'}
        </button>
        {error && <p role="alert" className="mt-2 rounded-md bg-red-50 px-3 py-2 text-red-700">{error}</p>}
      </div>
    </Section>
  )
}

function mapLink(lat: number | null, lon: number | null, address: string): string {
  return lat !== null && lon !== null
    ? `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=17/${lat}/${lon}`
    : `https://www.openstreetmap.org/search?query=${encodeURIComponent(address)}`
}

function resultTypeLabel(t: string): string {
  const labels: Record<string, string> = { street: 'Straße', postcode: 'PLZ-Gebiet', city: 'Ort', suburb: 'Ortsteil', county: 'Landkreis' }
  return labels[t] ?? t
}
