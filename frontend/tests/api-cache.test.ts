import assert from 'node:assert/strict'
import { test } from 'node:test'
import { apiFetch, cachedAdminFetch, readAdminCache, getAdminRevision, subscribeAdminUpdates } from '../src/api.ts'

test('admin cache reuses reads, expires, and invalidates on writes and logout', async () => {
  const originalFetch = globalThis.fetch
  const originalNow = Date.now
  let now = originalNow()
  let reads = 0
  Date.now = () => now
  globalThis.fetch = async () => new Response(JSON.stringify({ reads: ++reads }), { status: 200 })
  const signal = new AbortController().signal
  try {
    const first = await cachedAdminFetch('/leads?cache-test', signal)
    assert.deepEqual(await cachedAdminFetch('/leads?cache-test', signal), first)
    assert.equal(reads, 1)
    now += 15_001
    await cachedAdminFetch('/leads?cache-test', signal)
    assert.equal(reads, 2)
    await apiFetch('/leads/example/status', { method: 'PATCH' })
    assert.equal(readAdminCache('/leads?cache-test'), undefined)
    await cachedAdminFetch('/leads?cache-test', signal)
    await apiFetch('/auth/logout', { method: 'POST' })
    assert.equal(readAdminCache('/leads?cache-test'), undefined)
  } finally {
    globalThis.fetch = originalFetch
    Date.now = originalNow
  }
})

test('a dashboard read started before a status change returns fresh data', async () => {
  const originalFetch = globalThis.fetch
  let resolveRead!: (response: Response) => void
  let reads = 0
  globalThis.fetch = async (_url, init) => {
    if (init?.method === 'PATCH') return new Response('{}')
    if (++reads > 1) return new Response('{"qualified":1}')
    return new Promise<Response>((resolve) => { resolveRead = resolve })
  }
  try {
    const pending = cachedAdminFetch('/leads/stats?days=30', new AbortController().signal)
    await apiFetch('/leads/example/status', { method: 'PATCH' })
    resolveRead(new Response('{"qualified":0}'))
    assert.deepEqual(await pending, { qualified: 1 })
    assert.deepEqual(readAdminCache('/leads/stats?days=30'), { qualified: 1 })
    assert.equal(reads, 2)
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('successful status changes notify dashboard subscribers; failed writes do not', async () => {
  const originalFetch = globalThis.fetch
  const revision = getAdminRevision()
  let notifications = 0
  const unsubscribe = subscribeAdminUpdates(() => { notifications++ })
  try {
    globalThis.fetch = async () => new Response('{}')
    await apiFetch('/leads/example/status', { method: 'PATCH' })
    assert.equal(notifications, 1)
    assert.equal(getAdminRevision(), revision + 1)
    globalThis.fetch = async () => new Response('{}', { status: 500 })
    await assert.rejects(apiFetch('/leads/example/status', { method: 'PATCH' }))
    assert.equal(notifications, 1)
    unsubscribe()
    globalThis.fetch = async () => new Response('{}')
    await apiFetch('/leads/example/status', { method: 'PATCH' })
    assert.equal(notifications, 1)
  } finally {
    unsubscribe()
    globalThis.fetch = originalFetch
  }
})

test('an expired session invalidates once without triggering endless dashboard reloads', async () => {
  const originalFetch = globalThis.fetch
  let notifications = 0
  const unsubscribe = subscribeAdminUpdates(() => { notifications++ })
  try {
    globalThis.fetch = async () => new Response('{}', { status: 401 })
    await assert.rejects(apiFetch('/leads/stats?days=30'))
    await assert.rejects(apiFetch('/leads/stats?days=30'))
    assert.equal(notifications, 1)
  } finally {
    unsubscribe()
    globalThis.fetch = originalFetch
  }
})
