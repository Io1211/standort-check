// All requests go to the same origin under /api (Vite proxy locally, Vercel
// rewrite in production). The admin session is an HttpOnly cookie, so the
// browser sends it automatically and JavaScript never sees the token.
const adminCache = new Map<string, { value: unknown; expires: number }>()
let cacheGeneration = 0

function clearAdminCache() {
  adminCache.clear()
  cacheGeneration++
}

export function readAdminCache<T>(path: string): T | undefined {
  const entry = adminCache.get(path)
  if (!entry || entry.expires <= Date.now()) {
    adminCache.delete(path)
    return undefined
  }
  return entry.value as T
}

// Short-lived, tab-local cache: never write personal data to browser storage.
export async function cachedAdminFetch<T>(path: string, signal: AbortSignal): Promise<T> {
  const cached = readAdminCache<T>(path)
  if (cached !== undefined) return cached
  const generation = cacheGeneration
  const value = await apiFetch<T>(path, { signal })
  if (generation === cacheGeneration && !signal.aborted) {
    if (adminCache.size >= 30) adminCache.delete(adminCache.keys().next().value!)
    adminCache.set(path, { value, expires: Date.now() + 15_000 })
  }
  return value
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const mutation = (init?.method ?? 'GET').toUpperCase() !== 'GET'
  if (path === '/auth/logout' || path === '/auth/login') clearAdminCache()
  const res = await fetch(`/api${path}`, {
    ...init,
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    if (res.status === 401) clearAdminCache()
    const body = await res.json().catch(() => ({}))
    throw new ApiError(res.status, body.error ?? 'Unbekannter Fehler', body.fields)
  }
  if (mutation) clearAdminCache()
  return res.status === 204 ? (undefined as T) : res.json()
}

export class ApiError extends Error {
  status: number
  fields?: Record<string, string>

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

export function isUnauthorized(err: unknown): boolean {
  return err instanceof ApiError && err.status === 401
}
