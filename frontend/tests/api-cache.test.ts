import assert from 'node:assert/strict'
import { test } from 'node:test'
import { apiFetch, cachedAdminFetch, readAdminCache } from '../src/api.ts'

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

test('a read started before a mutation cannot repopulate the cache', async () => {
  const originalFetch = globalThis.fetch
  let resolveRead!: (response: Response) => void
  globalThis.fetch = async (_url, init) => {
    if (init?.method === 'PATCH') return new Response('{}')
    return new Promise<Response>((resolve) => { resolveRead = resolve })
  }
  try {
    const pending = cachedAdminFetch('/leads?race-test', new AbortController().signal)
    await apiFetch('/leads/example/status', { method: 'PATCH' })
    resolveRead(new Response('{"old":true}'))
    await pending
    assert.equal(readAdminCache('/leads?race-test'), undefined)
  } finally {
    globalThis.fetch = originalFetch
  }
})
