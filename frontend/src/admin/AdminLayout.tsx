import { useEffect, useState } from 'react'
import { NavLink, Navigate, Outlet, useLocation, useNavigate } from 'react-router'
import { apiFetch, isUnauthorized } from '../api'

// Guards all /admin pages: asks the API who is logged in. The real protection
// is on the server (every admin endpoint checks the session cookie); this
// only decides whether to show the login page.
export function AdminLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const [state, setState] = useState<'loading' | 'ok' | 'login' | 'error'>('loading')
  const [email, setEmail] = useState('')

  useEffect(() => {
    apiFetch<{ email: string }>('/auth/me')
      .then((me) => {
        setEmail(me.email)
        setState('ok')
      })
      .catch((err) => setState(isUnauthorized(err) ? 'login' : 'error'))
  }, [])

  async function logout() {
    await apiFetch('/auth/logout', { method: 'POST' }).catch(() => {})
    navigate('/admin/login', { replace: true })
  }

  if (state === 'loading') return <p className="p-8 text-sm text-neutral-500">Lade …</p>
  if (state === 'login') {
    const next = location.pathname + location.search
    return <Navigate to={`/admin/login?next=${encodeURIComponent(next)}`} replace />
  }
  if (state === 'error') return <p className="p-8 text-sm text-red-700">Server nicht erreichbar. Bitte Seite neu laden.</p>

  const navClass = ({ isActive }: { isActive: boolean }) =>
    `rounded-md px-3 py-1.5 text-sm font-medium ${isActive ? 'bg-ink text-white' : 'text-ink hover:bg-neutral-100'}`

  return (
    <div className="min-h-screen bg-neutral-50">
      <header className="border-b border-neutral-200 bg-white">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3">
          <span className="font-bold">Standort-Check · Sales</span>
          <nav className="flex gap-1">
            <NavLink to="/admin" end className={navClass}>Leads</NavLink>
            <NavLink to="/admin/kampagnen" className={navClass}>Kampagnen</NavLink>
            <NavLink to="/admin/dashboard" className={navClass}>Dashboard</NavLink>
          </nav>
          <div className="ml-auto flex items-center gap-3 text-sm text-neutral-600">
            <span className="hidden sm:inline">{email}</span>
            <button onClick={logout} className="rounded-md px-3 py-1.5 font-medium text-ink hover:bg-neutral-100">
              Abmelden
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
