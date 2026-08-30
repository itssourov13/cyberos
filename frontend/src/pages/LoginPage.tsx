import { Link } from 'react-router-dom'

export function LoginPage() {
  return (
    <div className="flex min-h-full items-center justify-center p-6">
      <div className="w-full max-w-md rounded-[var(--radius-lg)] border border-[var(--color-border)] bg-[var(--color-surface-1)]/95 p-8 shadow-2xl backdrop-blur-sm">
        <div className="text-center">
          <div className="text-base font-bold tracking-[0.22em] text-[var(--color-fg)]">CYBEROS</div>
          <div className="mt-1.5 text-[10px] uppercase tracking-[0.18em] text-[var(--color-muted)]">
            Security Operations Platform
          </div>
        </div>
        <form
          className="mt-8 space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
          }}
        >
          <label className="block text-[11px] uppercase tracking-wider text-[var(--color-muted)]">
            Principal
            <input
              className="mt-1.5 w-full rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-sm outline-none transition-colors focus:border-[var(--color-border-focus)]"
              placeholder="operator@example"
              autoComplete="username"
            />
          </label>
          <label className="block text-[11px] uppercase tracking-wider text-[var(--color-muted)]">
            Credential
            <input
              type="password"
              className="mt-1.5 w-full rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 text-sm outline-none transition-colors focus:border-[var(--color-border-focus)]"
              placeholder="••••••••"
              autoComplete="current-password"
            />
          </label>
          <Link to="/dashboard" className="ops-btn ops-btn-primary w-full py-2.5">
            Continue to console
          </Link>
        </form>
        <p className="mt-6 text-center text-[10px] uppercase tracking-[0.14em] text-amber-200/75">
          Environment: Development · Mock authentication
        </p>
        <p className="mt-2 text-center text-[12px] text-[var(--color-muted)]">
          Production authentication is provided by the CYBEROS backend. No secrets are stored in the
          frontend.
        </p>
      </div>
    </div>
  )
}
