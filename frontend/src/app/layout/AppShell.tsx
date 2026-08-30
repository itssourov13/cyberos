import { Outlet, Link } from 'react-router-dom'
import { Bell, Search, Activity } from 'lucide-react'
import { Sidebar } from '@/components/navigation/Sidebar'
import { isMockMode } from '@/api/client'

export function AppShell() {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex h-12 shrink-0 items-center justify-between border-b border-[var(--color-border)] bg-[var(--color-surface-1)]/95 px-4 backdrop-blur-sm">
        <div className="flex items-center gap-3">
          <Link
            to="/dashboard"
            className="text-[13px] font-semibold tracking-[0.14em] text-[var(--color-fg)] transition-opacity hover:opacity-90"
          >
            CYBEROS
          </Link>
          <span className="hidden text-[10px] uppercase tracking-[0.16em] text-[var(--color-muted)] sm:inline">
            Security Operations
          </span>
          {isMockMode() && (
            <span className="rounded-[var(--radius-sm)] border border-amber-500/35 bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider text-amber-200/90">
              Development · Mock API
            </span>
          )}
        </div>
        <div className="flex items-center gap-2 sm:gap-3">
          <div className="hidden items-center gap-1.5 rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2.5 py-1 md:flex">
            <Activity size={12} className="text-[var(--color-ok)]" aria-hidden />
            <span className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">System</span>
            <span className="text-[10px] font-medium text-[var(--color-ok)]">Nominal</span>
          </div>
          <div className="hidden items-center gap-2 rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2 py-1 lg:flex">
            <Search size={14} className="text-[var(--color-muted)]" />
            <input
              className="w-44 bg-transparent text-xs outline-none placeholder:text-[var(--color-muted)]"
              placeholder="Search IDs, digests…"
              aria-label="Search"
            />
          </div>
          <button
            type="button"
            className="ops-btn ops-btn-ghost rounded-[var(--radius-sm)] p-1.5"
            aria-label="Alerts"
          >
            <Bell size={16} />
          </button>
          <div
            className="flex h-7 w-7 items-center justify-center rounded-full border border-[var(--color-border)] bg-[var(--color-surface-3)] text-[10px] font-semibold tracking-wide"
            title="Operator"
          >
            OP
          </div>
        </div>
      </header>
      <div className="flex min-h-0 flex-1">
        <div className="hidden md:block">
          <Sidebar />
        </div>
        <main className="min-w-0 flex-1 overflow-y-auto p-4 md:p-6 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
