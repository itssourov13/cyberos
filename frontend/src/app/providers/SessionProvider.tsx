import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useApi } from '@/api'
import { onUnauthorized } from '@/api/http'
import { fetchSession, signIn as apiSignIn, signOut as apiSignOut, type SessionInfo } from '@/api/session'
import { SessionContext, type SessionState, type SessionStatus } from './session-context'

interface Inner {
  status: SessionStatus
  principalId?: string
  tenantId?: string
  expiresAt?: string
  expired: boolean
  error?: string
}

const statusOf = (s: SessionInfo): SessionStatus => (!s.authRequired ? 'open' : s.authenticated ? 'authenticated' : 'anonymous')

/**
 * Holds the *display* state of the backend session. The session itself is an
 * HttpOnly cookie the browser cannot read; every protected request is
 * re-checked by the backend, so nothing here grants access to anything.
 */
export function SessionProvider({ children }: { children: ReactNode }) {
  const api = useApi()
  const preview = api.mode === 'preview'
  const [s, setS] = useState<Inner>({ status: preview ? 'preview' : 'loading', expired: false })
  const [tick, setTick] = useState(0)

  useEffect(() => {
    if (preview) return
    const ctrl = new AbortController()
    fetchSession(ctrl.signal).then(
      (info) => setS({ status: statusOf(info), principalId: info.principalId, tenantId: info.tenantId, expiresAt: info.expiresAt, expired: false }),
      (e: unknown) => {
        if (ctrl.signal.aborted) return
        const code = (e as { code?: string }).code
        if (code === 'unauthenticated') setS({ status: 'anonymous', expired: false })
        else setS({ status: 'unavailable', expired: false, error: e instanceof Error ? e.message : 'Cannot reach the CYBEROS API.' })
      },
    )
    return () => ctrl.abort()
  }, [preview, tick])

  // Any 401 from a protected call means the session is gone.
  useEffect(
    () =>
      onUnauthorized(() =>
        setS((cur) => (cur.status === 'authenticated' ? { status: 'anonymous', expired: true } : cur)),
      ),
    [],
  )

  // Drop to signed-out at the server-declared expiry.
  useEffect(() => {
    if (s.status !== 'authenticated' || !s.expiresAt) return
    const ms = new Date(s.expiresAt).getTime() - Date.now()
    if (!Number.isFinite(ms)) return
    const t = window.setTimeout(() => setS({ status: 'anonymous', expired: true }), Math.max(ms, 0))
    return () => window.clearTimeout(t)
  }, [s.status, s.expiresAt])

  const signIn = useCallback(async (apiKey: string) => {
    const info = await apiSignIn(apiKey)
    setS({ status: statusOf(info), principalId: info.principalId, tenantId: info.tenantId, expiresAt: info.expiresAt, expired: false })
  }, [])

  const signOut = useCallback(async () => {
    try {
      await apiSignOut()
    } finally {
      setS({ status: 'anonymous', expired: false })
    }
  }, [])

  const retry = useCallback(() => {
    setS({ status: 'loading', expired: false })
    setTick((t) => t + 1)
  }, [])

  const value = useMemo<SessionState>(
    () => ({ status: s.status, principalId: s.principalId, tenantId: s.tenantId, expired: s.expired, error: s.error, signIn, signOut, retry }),
    [s, signIn, signOut, retry],
  )
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}
