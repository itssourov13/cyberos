import { request, setCsrfToken } from './http'

export interface SessionInfo {
  authRequired: boolean
  authenticated: boolean
  principalId?: string
  tenantId?: string
  expiresAt?: string
}

interface Wire {
  auth_required: boolean
  authenticated: boolean
  principal_id?: string
  tenant_id?: string
  csrf_token?: string
  expires_at?: string
}

function adopt(w: Wire): SessionInfo {
  setCsrfToken(w.csrf_token)
  return {
    authRequired: w.auth_required,
    authenticated: w.authenticated,
    principalId: w.principal_id,
    tenantId: w.tenant_id,
    expiresAt: w.expires_at,
  }
}

/** Ask the backend who we are. A 401 is an ordinary "not signed in" answer here. */
export async function fetchSession(signal?: AbortSignal): Promise<SessionInfo> {
  return adopt(await request<Wire>('/auth/me', { signal, suppressUnauthorized: true }))
}

/** Exchange an API key for an HttpOnly session cookie. The key is never stored. */
export async function signIn(apiKey: string): Promise<SessionInfo> {
  return adopt(await request<Wire>('/auth/login', { method: 'POST', body: { api_key: apiKey }, suppressUnauthorized: true }))
}

export async function signOut(): Promise<void> {
  try {
    await request<void>('/auth/logout', { method: 'POST', suppressUnauthorized: true })
  } finally {
    setCsrfToken(undefined)
  }
}
