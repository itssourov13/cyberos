import { Link } from 'react-router-dom'
import { ArrowRight, Lock } from 'lucide-react'
import { Banner } from '@/components/ui'
import { Wordmark } from '@/components/brand/Logo'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { SPINE_ALL_PASS } from '@/lib/spine'

/**
 * Sign-in screen. Authentication is a backend concern (Phase 24D): the fields
 * are intentionally disabled and nothing here creates a session.
 */
export function SignInPage() {
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
        <div className="auth-card">
          <h1>Sign in</h1>
          <p className="dim">Use your organisation’s identity to access the operator console.</p>
          <Banner tone="info" title="Authentication is not connected yet">
            Sign-in is delivered with backend integration in the next phase. For now you can walk through the console using sample data.
          </Banner>
          <div className="field">
            <label htmlFor="email">Email</label>
            <input id="email" className="input" type="email" placeholder="you@company.com" disabled autoComplete="username" />
          </div>
          <div className="field">
            <label htmlFor="password">Password</label>
            <input id="password" className="input" type="password" placeholder="••••••••" disabled autoComplete="current-password" />
          </div>
          <button type="button" className="btn btn-lg" disabled>
            <Lock size={15} /> Sign in
          </button>
          <hr className="hr" />
          <Link to="/app" className="btn btn-primary btn-lg">
            Continue to preview console <ArrowRight size={16} />
          </Link>
          <Link to="/" className="link" style={{ textAlign: 'center' }}>
            ← Back to site
          </Link>
        </div>
      </main>
    </div>
  )
}
