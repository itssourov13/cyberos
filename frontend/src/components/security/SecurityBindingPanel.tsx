import type { ReactNode } from 'react'
import { DigestView, IDView } from '@/components/ui/DigestView'
import { StatusBadge } from '@/components/ui/StatusBadge'

export type BindingFields = {
  projectId?: string
  scopeId?: string
  capabilityId?: string
  target?: string
  grantId?: string
  decisionId?: string
  packDigest?: string
  workerId?: string
  hostId?: string
  taskId?: string
}

/** Read-only security binding display — no editable authority fields. */
export function SecurityBindingPanel({
  title = 'Security binding',
  fields,
}: {
  title?: string
  fields: BindingFields
}) {
  const rows: { label: string; node: ReactNode }[] = []
  if (fields.projectId) rows.push({ label: 'Project', node: <IDView value={fields.projectId} /> })
  if (fields.scopeId) rows.push({ label: 'Scope', node: <IDView value={fields.scopeId} /> })
  if (fields.taskId) rows.push({ label: 'Task', node: <IDView value={fields.taskId} /> })
  if (fields.capabilityId)
    rows.push({
      label: 'Capability',
      node: <span className="ops-mono text-sky-200/90">{fields.capabilityId}</span>,
    })
  if (fields.target)
    rows.push({
      label: 'Target',
      node: <span className="ops-mono">{fields.target}</span>,
    })
  if (fields.decisionId) rows.push({ label: 'Decision', node: <IDView value={fields.decisionId} /> })
  if (fields.grantId) rows.push({ label: 'Grant', node: <IDView value={fields.grantId} /> })
  if (fields.packDigest)
    rows.push({ label: 'Pack digest', node: <DigestView value={fields.packDigest} /> })
  if (fields.workerId) rows.push({ label: 'Worker', node: <IDView value={fields.workerId} /> })
  if (fields.hostId) rows.push({ label: 'Host', node: <IDView value={fields.hostId} /> })

  return (
    <div className="ops-panel p-4">
      <div className="mb-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
        {title}
      </div>
      <dl className="grid gap-2.5 sm:grid-cols-2">
        {rows.map((r) => (
          <div key={r.label}>
            <dt className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{r.label}</dt>
            <dd className="mt-0.5">{r.node}</dd>
          </div>
        ))}
      </dl>
      <p className="mt-3 text-[11px] text-[var(--color-muted)]">
        Bindings are server/mock-derived and read-only. The browser cannot alter them.
      </p>
    </div>
  )
}

export function SecurityContextStrip({
  scopeStatus = 'IN_SCOPE',
  integrity = 'VERIFIED',
  signature = 'VALID',
  trust = 'TRUSTED',
}: {
  scopeStatus?: string
  integrity?: string
  signature?: string
  trust?: string
}) {
  return (
    <div className="ops-panel-elevated grid gap-2 p-3 sm:grid-cols-2 lg:grid-cols-4">
      <Ctx label="Scope" value={scopeStatus} />
      <Ctx label="Integrity" value={integrity} />
      <Ctx label="Signature" value={signature} />
      <Ctx label="Trust" value={trust} />
    </div>
  )
}

function Ctx({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{label}</div>
      <div className="mt-1">
        <StatusBadge status={value} />
      </div>
    </div>
  )
}

export function GrantBrokerChain({ grantId, decisionId }: { grantId?: string; decisionId?: string }) {
  return (
    <div className="ops-panel flex flex-wrap items-center justify-center gap-2 p-4 text-[12px]">
      <ChainBox label="Decision" value={decisionId ?? '—'} />
      <span className="text-[var(--color-muted)]">→</span>
      <ChainBox label="Grant" value={grantId ?? '—'} accent />
      <span className="text-[var(--color-muted)]">→</span>
      <ChainBox label="Broker" value="sole gateway" />
    </div>
  )
}

function ChainBox({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div
      className={`min-w-[7rem] rounded-[var(--radius-sm)] border px-3 py-2 text-center ${
        accent
          ? 'border-[var(--color-accent)]/40 bg-[var(--color-accent)]/10'
          : 'border-[var(--color-border)] bg-[var(--color-surface-2)]'
      }`}
    >
      <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{label}</div>
      <div className="ops-mono mt-1 truncate text-[11px] text-[var(--color-fg-secondary)]" title={value}>
        {value.length > 18 ? `${value.slice(0, 10)}…` : value}
      </div>
    </div>
  )
}
