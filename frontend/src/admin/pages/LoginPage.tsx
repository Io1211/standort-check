import { useState, type FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { ApiError, apiFetch } from '../../api'

export function LoginPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      await apiFetch('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) })
      navigate(safeNext(params.get('next')), { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Server nicht erreichbar.')
      setSubmitting(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-neutral-50 px-4">
      <form onSubmit={onSubmit} className="w-full max-w-sm space-y-4 rounded-lg bg-white p-6 shadow-sm ring-1 ring-neutral-200">
        <h1 className="text-xl font-bold">Standort-Check · Sales</h1>
        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium">E-Mail</label>
          <input id="email" type="email" autoComplete="username" required value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-md border border-neutral-300 px-3 py-2 outline-none focus:border-ink focus:ring-1 focus:ring-ink" />
        </div>
        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium">Passwort</label>
          <input id="password" type="password" autoComplete="current-password" required value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded-md border border-neutral-300 px-3 py-2 outline-none focus:border-ink focus:ring-1 focus:ring-ink" />
        </div>
        {error && <p role="alert" className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
        <button type="submit" disabled={submitting}
          className="w-full rounded-md bg-ink px-4 py-2 font-medium text-white hover:bg-ink/90 disabled:opacity-60">
          {submitting ? 'Anmelden …' : 'Anmelden'}
        </button>
      </form>
    </main>
  )
}

// Only allow redirects inside the admin area (no open redirect via ?next=).
function safeNext(next: string | null): string {
  return next && next.startsWith('/admin') && !next.startsWith('//') ? next : '/admin'
}
