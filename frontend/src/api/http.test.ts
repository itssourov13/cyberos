import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from './client'
import { onUnauthorized, request, setCsrfToken } from './http'

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

/** Builds a fresh Response per call (a Response body can only be read once). */
function stubFetch(make: (() => Response) | Error) {
  const fn = vi.fn(async (_url: string, _init?: RequestInit) => {
    if (make instanceof Error) throw make
    return make()
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => {
  vi.unstubAllGlobals()
  setCsrfToken(undefined)
})

describe('request', () => {
  it('sends credentials, builds the query without empty values, and no CSRF on GET', async () => {
    setCsrfToken('tok')
    const fetchFn = stubFetch(() => json(200, { ok: true }))
    await request('/scopes', { query: { project_id: 'p1', empty: '', skip: undefined, limit: 5 } })
    const [url, init] = fetchFn.mock.calls[0]!
    expect(url).toBe('/api/v1/scopes?project_id=p1&limit=5')
    expect(init!.credentials).toBe('include')
    expect(init!.headers).not.toHaveProperty('X-CSRF-Token')
  })

  it('sends the CSRF token on writes only when one is held', async () => {
    const fetchFn = stubFetch(() => json(200, {}))
    await request('/grants', { method: 'POST', body: { a: 1 } })
    expect(fetchFn.mock.calls[0]![1]!.headers).not.toHaveProperty('X-CSRF-Token')
    setCsrfToken('tok')
    await request('/grants', { method: 'POST', body: { a: 1 } })
    const headers = fetchFn.mock.calls[1]![1]!.headers as Record<string, string>
    expect(headers['X-CSRF-Token']).toBe('tok')
    expect(headers['Content-Type']).toBe('application/json')
  })

  it.each([
    [401, 'unauthenticated'],
    [403, 'forbidden'],
    [404, 'not-found'],
    [409, 'conflict'],
    [400, 'invalid'],
    [429, 'rate-limited'],
    [501, 'unsupported'],
    [503, 'server'],
  ])('maps HTTP %i to %s and keeps the server message', async (status, code) => {
    stubFetch(() => json(status, { error: 'nope' }))
    await expect(request('/x')).rejects.toMatchObject({ name: 'ApiError', code, status, message: 'nope' })
  })

  it('fires the session-expired handler on 401 unless suppressed', async () => {
    const handler = vi.fn()
    const off = onUnauthorized(handler)
    stubFetch(() => json(401, { error: 'expired' }))
    await expect(request('/x')).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledTimes(1)
    await expect(request('/auth/login', { suppressUnauthorized: true })).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledTimes(1)
    off()
  })

  it('reports an unreachable backend as unavailable, never as data', async () => {
    stubFetch(new TypeError('network down'))
    await expect(request('/x')).rejects.toMatchObject({ code: 'unavailable' })
  })

  it('returns undefined for 204', async () => {
    stubFetch(() => new Response(null, { status: 204 }))
    await expect(request('/x', { method: 'POST' })).resolves.toBeUndefined()
  })
})
