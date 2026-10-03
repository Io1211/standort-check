import { useState } from 'react'
import { apiFetch } from '../api'
import type { LeadStatus } from './types'

// Changes a lead's status optimistically: the UI updates immediately and is
// rolled back if the request fails.
export function useStatusUpdate(apply: (id: string, status: LeadStatus) => void) {
  const [saving, setSaving] = useState<string | null>(null)
  const [error, setError] = useState('')

  async function updateStatus(id: string, from: LeadStatus, to: LeadStatus) {
    if (from === to) return
    setError('')
    setSaving(id)
    apply(id, to)
    try {
      await apiFetch(`/leads/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status: to }) })
    } catch (err) {
      apply(id, from)
      setError(err instanceof Error ? err.message : 'Status konnte nicht gespeichert werden.')
    } finally {
      setSaving(null)
    }
  }

  return { updateStatus, saving, error }
}
