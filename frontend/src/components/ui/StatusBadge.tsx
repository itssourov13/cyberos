import type { HealthState, TrustState, WorkerStatus, AssignmentStatus, DecisionOutcome } from '@/types/models'

type Status =
  | HealthState
  | TrustState
  | WorkerStatus
  | AssignmentStatus
  | DecisionOutcome
  | 'VALID'
  | 'INVALID'
  | 'VERIFIED'
  | 'UNVERIFIED'
  | string

const styles: Record<string, string> = {
  HEALTHY: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  DEGRADED: 'bg-amber-500/12 text-amber-300 border-amber-500/25',
  UNKNOWN: 'bg-slate-500/12 text-slate-300 border-slate-500/25',
  UNAVAILABLE: 'bg-slate-500/10 text-slate-400 border-slate-500/20',
  ALLOW: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  DENY: 'bg-orange-500/12 text-orange-300 border-orange-500/30',
  TRUSTED: 'bg-sky-500/12 text-sky-300 border-sky-500/25',
  UNTRUSTED: 'bg-slate-500/12 text-slate-300 border-slate-500/25',
  VERIFIED: 'bg-cyan-500/12 text-cyan-300 border-cyan-500/25',
  REVOKED: 'bg-red-500/12 text-red-300 border-red-500/30',
  ACTIVE: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  DISABLED: 'bg-amber-500/12 text-amber-300 border-amber-500/25',
  PENDING: 'bg-slate-500/12 text-slate-300 border-slate-500/25',
  RUNNING: 'bg-blue-500/12 text-blue-300 border-blue-500/25',
  COMPLETED: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  FAILED: 'bg-red-500/12 text-red-300 border-red-500/30',
  EXPIRED: 'bg-amber-500/12 text-amber-300 border-amber-500/25',
  VALID: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  INVALID: 'bg-red-500/12 text-red-300 border-red-500/30',
  UNVERIFIED: 'bg-slate-500/12 text-slate-300 border-slate-500/25',
  PASS: 'bg-emerald-500/12 text-emerald-400 border-emerald-500/25',
  WARN: 'bg-amber-500/12 text-amber-300 border-amber-500/25',
  FAIL: 'bg-red-500/12 text-red-300 border-red-500/30',
}

export function StatusBadge({ status, label }: { status: Status; label?: string }) {
  const key = String(status).toUpperCase()
  const cls = styles[key] ?? styles.UNKNOWN
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-[var(--radius-sm)] border px-2 py-0.5 text-[11px] font-medium tracking-wide ${cls}`}
      role="status"
      aria-label={label ?? key}
    >
      <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-current opacity-90" aria-hidden />
      {label ?? key}
    </span>
  )
}
