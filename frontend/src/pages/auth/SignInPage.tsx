import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { ArrowRight, Lock } from 'lucide-react'
import { useApi } from '@/api'
import { Banner } from '@/components/ui'
import { Wordmark } from '@/components/brand/Logo'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { SPINE_ALL_PASS } from '@/lib/spine'
import { useSession } from '@/app/providers/session-context'

/** Only same-app paths are valid post-login targets (no open redirects). */
function safeTarget(from: unknown): string {
  return typeof from === 'string' && /^\/app(\/|\?|$)/.test(from) ? from : '/app'
}

function friendly(e: unknown): string {
  const code = (e as { code?: string }).code
  if (code === 'unauthenticated') return 'That API key was not accepted.'
  if (code === 'rate-limited') return 'Too many attempts. Wait a minute and try again.'
  if (code === 'invalid') return 'Enter your API key.'
  if (code === 'unavailable' || code === 'timeout') return 'Cannot reach the CYBEROS API.'
  return e instanceof Error ? e.message : 'Sign-in failed.'
}

/**
 * Sign-in screen. The key is exchanged once for an HttpOnly session cookie by
 * the backend; it is never stored in the browser and never shown again.
 */
export function SignInPage() {
  const api = useApi()
  const session = useSession()
  const navigate = useNavigate()
  const location = useLocation()
  const state = (location.state ?? {}) as { from?: string; expired?: boolean }
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  const preview = api.mode === 'preview'
  if (!preview && (session.status === 'authenticated' || session.status === 'open')) {
    return <Navigate to={safeTarget(state.from)} replace />
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!key.trim() || busy) return
    setBusy(true)
    setError(undefined)
    try {
      await session.signIn(key.trim())
      setKey('')
      navigate(safeTarget(state.from), { replace: true })
    } catch (err) {
      setError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-wrap">
      <aside className="auth-aside">
        <Link to="/" aria-label="CYBEROS home">
          <Wordmark />
        </Link>
        <div className="col" style={{ gap: 'var(--s-5)' }}>
          <h2>Every action, on the record.</h2>
          <SpineFlow stages={SPINE_ALL_PASS} compact />
        </div>
        <span className="muted mono" style={{ fontSize: 12 }}>
          Scope → Decision → Grant → Broker → Execution → Evidence
        </span>
      </aside>
      <main className="auth-main" id="main">
        <form className="auth-card" onSubmit={submit} noValidate>
          <h1>Sign in</h1>
          <p className="dim">Use your CYBEROS API key to access the operator console.</p>
          {preview && (
            <Banner tone="warn" title="Preview mode">
              This build uses sample data and has no backend session. Nothing here authenticates you.
            </Banner>
          )}
          {state.expired && <Banner tone="warn" title="Your session ended">Sign in again to continue.</Banner>}
          {session.status === 'unavailable' && <Banner tone="danger" title="Cannot reach the CYBEROS API">{session.error}</Banner>}
          {error && <Banner tone="danger" title="Sign-in failed">{error}</Banner>}
          {!preview && (
            <>
              <div className="field">
                <label htmlFor="api-key">API key</label>
                <input
                  id="api-key"
                  className="input"
                  type="password"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  autoComplete="current-password"
                  spellCheck={false}
                  autoFocus
                />
                <span className="hint">Exchanged once for a secure session cookie. It is not stored in this browser.</span>
              </div>
              <button type="submit" className="btn btn-primary btn-lg" disabled={busy || !key.trim()}>
                <Lock size={15} /> {busy ? 'Signing in…' : 'Sign in'}
              </button>
            </>
          )}
          {preview && (
            <Link to="/app" className="btn btn-primary btn-lg">
              Continue to preview console <ArrowRight size={16} />
            </Link>
          )}
          <hr className="hr" />
          <Link to="/" className="link" style={{ textAlign: 'center' }}>
            ← Back to site
          </Link>
        </form>
      </main>
    </div>
  )
}
