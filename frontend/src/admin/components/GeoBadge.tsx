import type { ReactNode } from 'react'
import type { GeoInfo } from '../types'

// Service-area status at a glance. Never colour alone: every badge has a
// symbol and a text label.
export function AreaBadge({ geo }: { geo: GeoInfo }) {
  if (geo.status === 'ok' && geo.inServiceArea === true) {
    return <Badge className="bg-green-50 text-green-800 ring-green-200">✓ Einzugsgebiet</Badge>
  }
  if (geo.status === 'ok' && geo.inServiceArea === false) {
    return <Badge className="bg-neutral-100 text-neutral-700 ring-neutral-300">✕ Außerhalb</Badge>
  }
  if (geo.status === 'not_found') {
    return <Badge className="bg-amber-50 text-amber-800 ring-amber-200">? Adresse nicht gefunden</Badge>
  }
  if (geo.status === 'error' || geo.status === '') {
    return <Badge className="bg-neutral-50 text-neutral-500 ring-neutral-200">Geodaten fehlen</Badge>
  }
  return null // ok, but no service area configured
}

// Data-quality warnings from the geocoder.
export function GeoWarnings({ geo, postalCode }: { geo: GeoInfo; postalCode: string }) {
  if (geo.status !== 'ok') return null
  return (
    <>
      {geo.postcodeMismatch && (
        <Badge className="bg-amber-50 text-amber-800 ring-amber-200" title={`Eingegeben: ${postalCode}, gefunden: ${geo.postcode}`}>
          ⚠ PLZ passt nicht ({geo.postcode})
        </Badge>
      )}
      {geo.imprecise && <Badge className="bg-amber-50 text-amber-800 ring-amber-200">⚠ Lage ungenau</Badge>}
    </>
  )
}

function Badge({ className, title, children }: { className: string; title?: string; children: ReactNode }) {
  return (
    <span title={title} className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${className}`}>
      {children}
    </span>
  )
}
