import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link } from 'react-router'
import { ApiError, apiFetch } from '../../api'
import { PRESETS, slugify, trackingLink, type Campaign, type CampaignInput } from '../campaigns'

const EMPTY: CampaignInput = { name: '', utmSource: '', utmMedium: '', utmCampaign: '', utmContent: '', notes: '', archived: false }

// Manage campaigns and their tracking links. A campaign is only the link
// configuration: leads keep their UTM values, so editing or deleting a
// campaign never changes or loses a lead.
export function CampaignsPage() {
  const [campaigns, setCampaigns] = useState<Campaign[] | null>(null)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState<Campaign | 'new' | null>(null)
  const [showArchived, setShowArchived] = useState(false)

  function load() {
    apiFetch<{ campaigns: Campaign[] }>('/campaigns')
      .then((d) => setCampaigns(d.campaigns))
      .catch((err) => setError(err.message))
  }
  useEffect(load, [])

  async function save(c: Campaign, patch: Partial<CampaignInput>) {
    try {
      const updated = await apiFetch<Campaign>(`/campaigns/${c.id}`, { method: 'PUT', body: JSON.stringify({ ...toInput(c), ...patch }) })
      setCampaigns((list) => list?.map((x) => (x.id === c.id ? updated : x)) ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Speichern fehlgeschlagen.')
    }
  }

  async function remove(c: Campaign) {
    const msg = c.leads > 0
      ? `Kampagne „${c.name}“ löschen? Der Link funktioniert weiter, und die ${c.leads} Leads bleiben unverändert erhalten.`
      : `Kampagne „${c.name}“ löschen?`
    if (!window.confirm(msg)) return
    try {
      await apiFetch(`/campaigns/${c.id}`, { method: 'DELETE' })
      setCampaigns((list) => list?.filter((x) => x.id !== c.id) ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Löschen fehlgeschlagen.')
    }
  }

  const visible = campaigns?.filter((c) => showArchived || !c.archived) ?? []
  const archivedCount = campaigns?.filter((c) => c.archived).length ?? 0

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">Kampagnen</h1>
          <p className="text-sm text-neutral-600">Tracking-Links für Google, Meta & Co. anlegen und Ergebnisse je Link sehen.</p>
        </div>
        {editing === null && (
          <button onClick={() => setEditing('new')} className="rounded-md bg-ink px-4 py-2 text-sm font-medium text-white hover:bg-ink/90">
            + Neue Kampagne
          </button>
        )}
      </div>

      {error && <p role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>}

      {editing !== null && (
        <CampaignForm
          initial={editing === 'new' ? EMPTY : toInput(editing)}
          id={editing === 'new' ? null : editing.id}
          onCancel={() => setEditing(null)}
          onSaved={(c) => {
            setCampaigns((list) => (editing === 'new' ? [c, ...(list ?? [])] : list?.map((x) => (x.id === c.id ? c : x)) ?? null))
            setEditing(null)
          }}
        />
      )}

      {!campaigns && !error && <p className="text-sm text-neutral-500">Lade …</p>}
      {campaigns && visible.length === 0 && (
        <p className="rounded-lg bg-white px-4 py-10 text-center text-neutral-600 ring-1 ring-neutral-200">
          {archivedCount > 0
            ? 'Alle Kampagnen sind archiviert.'
            : 'Noch keine Kampagnen. Lege die erste an, um einen Tracking-Link zu bekommen.'}
        </p>
      )}

      <ul className="space-y-3">
        {visible.map((c) => (
          <CampaignCard key={c.id} c={c} onEdit={() => setEditing(c)} onArchive={() => save(c, { archived: !c.archived })}
            onDelete={() => remove(c)} />
        ))}
      </ul>

      {archivedCount > 0 && (
        <label className="flex items-center gap-2 text-sm text-neutral-600">
          <input type="checkbox" checked={showArchived} onChange={(e) => setShowArchived(e.target.checked)} className="size-4 accent-ink" />
          Archivierte anzeigen ({archivedCount})
        </label>
      )}
    </div>
  )
}

function CampaignCard({ c, onEdit, onArchive, onDelete }: { c: Campaign; onEdit: () => void; onArchive: () => void; onDelete: () => void }) {
  const link = trackingLink(c)
  const leadsHref = `/admin?source=${encodeURIComponent(c.utmSource)}&campaign=${encodeURIComponent(c.utmCampaign)}`
  const rate = c.unique > 0 ? Math.round((c.qualified / c.unique) * 100) : null

  return (
    <li className={`rounded-lg bg-white p-4 ring-1 ring-neutral-200 ${c.archived ? 'opacity-60' : ''}`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-semibold">{c.name}</h2>
            <Tag>{c.utmSource} / {c.utmMedium}</Tag>
            {c.archived && <Tag>archiviert</Tag>}
          </div>
          <div className="mt-0.5 text-sm text-neutral-600">
            Kampagne <code className="text-neutral-800">{c.utmCampaign}</code>
            {c.utmContent && <> · Anzeige <code className="text-neutral-800">{c.utmContent}</code></>}
          </div>
          {c.notes && <p className="mt-1 text-sm whitespace-pre-line text-neutral-600">{c.notes}</p>}
        </div>
        <Link to={leadsHref} className="text-right text-sm hover:underline">
          <div><strong className="text-lg">{c.unique}</strong> Leads</div>
          <div className="text-neutral-600">
            {c.qualified} qualifiziert{rate !== null && ` (${rate} %)`}
          </div>
        </Link>
      </div>

      <LinkBox link={link} />

      <div className="mt-3 flex flex-wrap gap-2 text-sm">
        <button onClick={onEdit} className="rounded-md px-3 py-1.5 font-medium ring-1 ring-neutral-300 hover:bg-neutral-50">Bearbeiten</button>
        <button onClick={onArchive} className="rounded-md px-3 py-1.5 font-medium ring-1 ring-neutral-300 hover:bg-neutral-50">
          {c.archived ? 'Reaktivieren' : 'Archivieren'}
        </button>
        <button onClick={onDelete} className="rounded-md px-3 py-1.5 font-medium text-red-700 ring-1 ring-red-200 hover:bg-red-50">Löschen</button>
      </div>
    </li>
  )
}

function LinkBox({ link }: { link: string }) {
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      // Clipboard can be blocked (e.g. no HTTPS); the link is selectable anyway.
    }
  }
  return (
    <div className="mt-3 flex items-stretch overflow-hidden rounded-md ring-1 ring-neutral-200">
      <input readOnly value={link} aria-label="Tracking-Link" onFocus={(e) => e.target.select()}
        className="min-w-0 flex-1 bg-neutral-50 px-3 py-2 font-mono text-xs text-neutral-700 outline-none" />
      <button onClick={copy} className="shrink-0 border-l border-neutral-200 bg-white px-3 text-sm font-medium hover:bg-neutral-50">
        {copied ? 'Kopiert ✓' : 'Kopieren'}
      </button>
    </div>
  )
}

function CampaignForm({ initial, id, onCancel, onSaved }: {
  initial: CampaignInput; id: string | null; onCancel: () => void; onSaved: (c: Campaign) => void
}) {
  const [v, setV] = useState(initial)
  // While creating, the campaign slug follows the name until edited by hand.
  const [slugTouched, setSlugTouched] = useState(id !== null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  const set = (patch: Partial<CampaignInput>) => setV((prev) => ({ ...prev, ...patch }))

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setErrors({})
    setFormError('')
    try {
      const saved = await apiFetch<Campaign>(id ? `/campaigns/${id}` : '/campaigns', {
        method: id ? 'PUT' : 'POST',
        body: JSON.stringify(v),
      })
      onSaved(saved)
    } catch (err) {
      if (err instanceof ApiError && err.fields) setErrors(err.fields)
      setFormError(err instanceof Error ? err.message : 'Speichern fehlgeschlagen.')
      setSaving(false)
    }
  }

  const preview = v.utmSource && v.utmMedium && v.utmCampaign ? trackingLink(v) : ''

  return (
    <form onSubmit={onSubmit} className="space-y-4 rounded-lg bg-white p-5 ring-1 ring-neutral-300">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-lg font-semibold">{id ? 'Kampagne bearbeiten' : 'Neue Kampagne'}</h2>
        <div className="flex gap-2 text-sm">
          {PRESETS.map((p) => (
            <button type="button" key={p.label} onClick={() => set({ utmSource: p.utmSource, utmMedium: p.utmMedium })}
              className="rounded-md px-3 py-1.5 font-medium ring-1 ring-neutral-300 hover:bg-neutral-50">
              {p.label}
            </button>
          ))}
        </div>
      </div>

      <Input label="Name" hint="Für euch intern, z. B. „Dresden Herbst 2026“" error={errors.name}>
        <input value={v.name} required onChange={(e) => set({ name: e.target.value, ...(slugTouched ? {} : { utmCampaign: slugify(e.target.value) }) })}
          className={inputCls} />
      </Input>

      <div className="grid gap-4 sm:grid-cols-3">
        <Input label="Quelle (utm_source)" hint="z. B. google, meta" error={errors.utmSource}>
          <input value={v.utmSource} required onChange={(e) => set({ utmSource: typing(e.target.value) })}
            onBlur={() => set({ utmSource: slugify(v.utmSource) })} className={inputCls} />
        </Input>
        <Input label="Medium (utm_medium)" hint="z. B. cpc, paid_social, email" error={errors.utmMedium}>
          <input value={v.utmMedium} required onChange={(e) => set({ utmMedium: typing(e.target.value) })}
            onBlur={() => set({ utmMedium: slugify(v.utmMedium) })} className={inputCls} />
        </Input>
        <Input label="Kampagne (utm_campaign)" hint="wird aus dem Namen erzeugt" error={errors.utmCampaign}>
          <input value={v.utmCampaign} required
            onChange={(e) => { setSlugTouched(true); set({ utmCampaign: typing(e.target.value) }) }}
            onBlur={() => set({ utmCampaign: slugify(v.utmCampaign) })} className={inputCls} />
        </Input>
      </div>

      <Input label="Anzeige / Variante (utm_content, optional)" error={errors.utmContent}
        hint="Unterscheidet Anzeigen derselben Kampagne. Platzhalter wie {creative} (Google) oder {{ad.name}} (Meta) ersetzt die Plattform beim Klick; feste Teile davor oder danach müssen übereinstimmen.">
        <input value={v.utmContent} onChange={(e) => set({ utmContent: e.target.value.trim() })} className={inputCls} />
      </Input>

      <Input label="Notiz (optional)" error={errors.notes}>
        <textarea value={v.notes} rows={2} onChange={(e) => set({ notes: e.target.value })} className={inputCls} />
      </Input>

      {preview && (
        <div>
          <div className="mb-1 text-sm font-medium">Tracking-Link</div>
          <LinkBox link={preview} />
          <p className="mt-1 text-xs text-neutral-500">
            Google und Meta hängen ihre Klick-IDs (gclid / fbclid) bei aktiviertem Auto-Tagging selbst an – die werden ebenfalls gespeichert.
          </p>
        </div>
      )}

      {formError && <p role="alert" className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{formError}</p>}

      <div className="flex gap-2">
        <button type="submit" disabled={saving} className="rounded-md bg-ink px-4 py-2 text-sm font-medium text-white hover:bg-ink/90 disabled:opacity-60">
          {saving ? 'Speichern …' : 'Speichern'}
        </button>
        <button type="button" onClick={onCancel} className="rounded-md px-4 py-2 text-sm font-medium ring-1 ring-neutral-300 hover:bg-neutral-50">
          Abbrechen
        </button>
      </div>
    </form>
  )
}

const typing = (s: string) => slugify(s, { keepTrailing: true })

const inputCls = 'w-full rounded-md border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-ink focus:ring-1 focus:ring-ink'

function Input({ label, hint, error, children }: { label: string; hint?: string; error?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium">{label}</span>
      {children}
      {error ? <span className="mt-1 block text-xs text-red-700">{error}</span> : hint && <span className="mt-1 block text-xs text-neutral-500">{hint}</span>}
    </label>
  )
}

function Tag({ children }: { children: ReactNode }) {
  return <span className="rounded bg-neutral-100 px-2 py-0.5 text-xs text-neutral-700 ring-1 ring-neutral-200">{children}</span>
}

function toInput(c: Campaign): CampaignInput {
  return { name: c.name, utmSource: c.utmSource, utmMedium: c.utmMedium, utmCampaign: c.utmCampaign, utmContent: c.utmContent, notes: c.notes, archived: c.archived }
}
