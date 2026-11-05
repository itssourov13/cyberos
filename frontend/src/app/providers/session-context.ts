import { createContext, useContext } from 'react'

export type SessionStatus =
  | 'loading'
  /** Preview adapter selected explicitly: there is no backend session. */
  | 'preview'
  /** Backend reports authentication disabled (explicit development mode). */
  | 'open'
  | 'authenticated'
  | 'anonymous'
  /** The backend could not be reached while restoring the session. */
  | 'unavailable'

export interface SessionState {
  status: SessionStatus
  principalId?: string
  tenantId?: string
  /** Set when a previously valid session was rejected or expired. */
  expired: boolean
  error?: string
  signIn: (apiKey: string) => Promise<void>
  signOut: () => Promise<void>
  retry: () => void
}

export const SessionContext = createContext<SessionState | null>(null)

export function useSession(): SessionState {
  const s = useContext(SessionContext)
  if (!s) throw new Error('useSession must be used inside <SessionProvider>')
  return s
}
