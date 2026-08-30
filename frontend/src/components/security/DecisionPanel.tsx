import type { Decision } from '@/types/models'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { IDView } from '@/components/ui/DigestView'
import { ConceptStrip } from '@/components/security/SecurityPipeline'

const outcomeShell: Record<string, string> = {
  ALLOW: 'border-emerald-500/35 bg-emerald-500/8',
  DENY: 'border-orange-500/40 bg-orange-500/10',
  UNKNOWN: 'border-slate-500/35 bg-slate-500/10',
}

export function DecisionPanel({ decision }: { decision: Decision }) {
  const outcome = decision.outcome.toUpperCase()
  const shell = outcomeShell[outcome] ?? outcomeShell.UNKNOWN

  return (
    <div className={`space-y-4 rounded-[var(--radius-md)] border p-4 ${shell}`}>
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
          Server decision
        </span>
        <StatusBadge status={decision.outcome} />
      </div>

      {outcome === 'ALLOW' && (
        <p className="text-[13px] text-emerald-100/90">
          Policy requirements satisfied in this MOCK evaluate path. Live CP was not contacted.
        </p>
      )}
      {outcome === 'DENY' && (
        <p className="text-[13px] text-orange-100/90">
          Request denied. No Grant can be issued from this decision. Execution remains unavailable.
        </p>
      )}
      {outcome === 'UNKNOWN' && (
        <p className="text-[13px] text-slate-200/90">
          Decision could not be established. UNKNOWN must never be treated as ALLOW. Execution remains
          unavailable.
        </p>
      )}

      <div className="grid gap-3 text-[13px] sm:grid-cols-2">
        <Field label="Decision ID" node={<IDView value={decision.id} />} />
        <Field label="Correlation" node={<IDView value={decision.correlationId ?? '—'} />} />
        <Field label="Project" node={<span className="ops-mono">{decision.projectId}</span>} />
        <Field label="Scope" node={<span className="ops-mono">{decision.scopeId}</span>} />
        <Field
          label="Capability"
          node={<span className="ops-mono text-sky-200/90">{decision.capabilityId}</span>}
        />
        <Field label="Reason" node={<span className="text-[var(--color-fg-secondary)]">{decision.reason}</span>} />
        <Field
          label="Timestamp"
          node={
            <span className="ops-mono text-[var(--color-muted)]">
              {new Date(decision.timestamp).toLocaleString()}
            </span>
          }
        />
      </div>

      <ConceptStrip />
      <p className="text-[11px] text-[var(--color-muted)]">
        Decision is server-derived (or MOCK). Browser cannot set TRUSTED, AUTHORIZED, or ALLOW.
      </p>
    </div>
  )
}

function Field({ label, node }: { label: string; node: React.ReactNode }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{label}</div>
      <div className="mt-0.5">{node}</div>
    </div>
  )
}
