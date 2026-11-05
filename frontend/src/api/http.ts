import { ApiError } from './client'

/**
 * Centralised HTTP transport for the live adapter.
 *
 * - Same-origin by default (`VITE_CYBEROS_API_BASE` overrides the origin only).
 * - Authentication is an opaque HttpOnly session cookie the browser attaches
 *   (`credentials: 'include'`). No credential is ever read, stored or logged
 *   here; the only secret held in memory is the per-session CSRF token the
 *   backend hands back, which is sent on state-changing requests.
 * - Every failure is mapped to a typed `ApiError`. Nothing falls back to
 *   fixtures.
 */

const BASE = ((import.meta.env.VITE_CYBEROS_API_BASE as string | undefined) ?? '').replace(/\/+$/, '')
const DEFAULT_TIMEOUT_MS = 15_000

let csrfToken: string | undefined
export const setCsrfToken = (t: string | undefined) => {
  csrfToken = t
}

type Listener = () => void
const unauthorized = new Set<Listener>()
/** Subscribe to "the backend says this session is no longer valid". */
export function onUnauthorized(fn: Listener): () => void {
  unauthorized.add(fn)
  return () => {
    unauthorized.delete(fn)
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  query?: Record<string, string | number | boolean | undefined>
  body?: unknown
  signal?: AbortSignal
  timeoutMs?: number
  /** Auth endpoints must not trigger the global session-expired handling. */
  suppressUnauthorized?: boolean
}

function url(path: string, query?: RequestOptions['query']): string {
  const u = `${BASE}/api/v1${path}`
  if (!query) return u
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(query)) if (v !== undefined && v !== '') qs.set(k, String(v))
  const s = qs.toString()
  return s ? `${u}?${s}` : u
}

function codeFor(status: number): ApiError['code'] {
  if (status === 401) return 'unauthenticated'
  if (status === 403) return 'forbidden'
  if (status === 404) return 'not-found'
  if (status === 409) return 'conflict'
  if (status === 400 || status === 422) return 'invalid'
  if (status === 429) return 'rate-limited'
  if (status === 501) return 'unsupported'
  if (status >= 500) return 'server'
  return 'unknown'
}

async function errorMessage(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { error?: unknown }
    if (typeof j.error === 'string' && j.error) return j.error
  } catch {
    /* non-JSON error body */
  }
  return res.statusText || `HTTP ${res.status}`
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const method = opts.method ?? 'GET'
  const ctrl = new AbortController()
  let timedOut = false
  const timer = setTimeout(() => {
    timedOut = true
    ctrl.abort()
  }, opts.timeoutMs ?? DEFAULT_TIMEOUT_MS)
  const onAbort = () => ctrl.abort()
  opts.signal?.addEventListener('abort', onAbort)

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'
  if (method !== 'GET' && csrfToken) headers['X-CSRF-Token'] = csrfToken

  try {
    const res = await fetch(url(path, opts.query), {
      method,
      headers,
      credentials: 'include',
      cache: 'no-store',
      signal: ctrl.signal,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    })
    if (!res.ok) {
      const msg = await errorMessage(res)
      if (res.status === 401 && !opts.suppressUnauthorized) unauthorized.forEach((fn) => fn())
      throw new ApiError(codeFor(res.status), msg, res.status)
    }
    if (res.status === 204) return undefined as T
    return (await res.json()) as T
  } catch (e) {
    if (e instanceof ApiError) throw e
    if (timedOut) throw new ApiError('timeout', 'The CYBEROS API did not respond in time.')
    if (opts.signal?.aborted) throw new DOMException('Aborted', 'AbortError')
    throw new ApiError('unavailable', 'Cannot reach the CYBEROS API.')
  } finally {
    clearTimeout(timer)
    opts.signal?.removeEventListener('abort', onAbort)
  }
}
