import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import type { Grant } from '@/types/models'
import { PageHeader, EmptyState } from '@/components/ui/PageHeader'
import { DigestView, IDView } from '@/components/ui/DigestView'
import { ConfirmDialog } from '@/components/ui/DataTable'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { SecurityPipeline } from '@/components/security/SecurityPipeline'
import { SecurityBindingPanel } from '@/components/security/SecurityBindingPanel'

type ExecPhase =
  | 'READY'
  | 'SUBMITTING'
  | 'BROKER_ACCEPTED'
  | 'DISPATCHED'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'

export function ExecutePage() {
  const [grants, setGrants] = useState<Grant[]>([])
  const [selected, setSelected] = useState('')
  const [confirm, setConfirm] = useState(false)
  const [phase, setPhase] = useState<ExecPhase>('READY')

  useEffect(() => {
    api.listGrants().then((g) => {
      setGrants(g)
      if (g[0]) setSelected(g[0].id)
    })
  }, [])

  const grant = grants.find((g) => g.id === selected)

  function runMockPipeline() {
    setPhase('SUBMITTING')
    const steps: ExecPhase[] = ['BROKER_ACCEPTED', 'DISPATCHED', 'RUNNING', 'COMPLETED']
    let i = 0
    const tick = () => {
      if (i >= steps.length) return
      setPhase(steps[i])
      i += 1
      if (i < steps.length) setTimeout(tick, 450)
    }
    setTimeout(tick, 400)
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title="Execute"
        description="Submit a server-authorized Grant to the Broker path. UI does not execute locally."
      />

      <div className="rounded-[var(--radius-md)] border border-orange-500/40 bg-orange-500/10 px-4 py-3 text-[12px] text-orange-100">
        Critical path. Confirmation required. <strong>MOCK execution only</strong> — Broker / Host /
        Executor are not contacted from the browser.
      </div>

      <SecurityPipeline activeIndex={5} />

      {!grant && (
        <EmptyState
          title="No grants available"
          description="Issue a Grant via server before execute. MOCK inventory may be empty."
        />
      )}

      {grant && (
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="space-y-4">
            <div className="ops-panel space-y-3 p-4">
              <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
                Execution request
              </div>
              <label className="block text-[11px] uppercase tracking-wider text-[var(--color-muted)]">
                Grant
                <select
                  value={selected}
                  onChange={(e) => {
                    setSelected(e.target.value)
                    setPhase('READY')
                  }}
                  className="mt-1 w-full rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 ops-mono text-[13px]"
                >
                  {grants.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.id}
                    </option>
                  ))}
                </select>
              </label>
              <div className="flex flex-wrap gap-2">
                <StatusBadge status={grant.status} />
                <IDView value={grant.id} label="Grant" />
              </div>
              {grant.packContentDigest && (
                <DigestView value={grant.packContentDigest} label="Pack" />
              )}
              <button
                type="button"
                onClick={() => setConfirm(true)}
                disabled={phase !== 'READY' && phase !== 'COMPLETED' && phase !== 'FAILED'}
                className="ops-btn ops-btn-danger w-full py-2.5 disabled:opacity-50"
              >
                Request execution…
              </button>
              <p className="text-[11px] text-[var(--color-muted)]">
                Understand → verify Grant bindings → confirm → submit (MOCK).
              </p>
            </div>

            <div className="ops-panel p-4">
              <div className="mb-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
                Execution state (MOCK)
              </div>
              <div className="flex flex-wrap gap-1.5">
                {(
                  ['READY', 'SUBMITTING', 'BROKER_ACCEPTED', 'DISPATCHED', 'RUNNING', 'COMPLETED'] as ExecPhase[]
                ).map((s) => (
                  <span
                    key={s}
                    className={`rounded-[var(--radius-sm)] border px-2 py-1 text-[10px] font-medium ${
                      phase === s
                        ? 'border-[var(--color-accent)]/50 bg-[var(--color-accent)]/15 text-sky-100'
                        : 'border-[var(--color-border)] text-[var(--color-muted)]'
                    }`}
                  >
                    {s}
                  </span>
                ))}
              </div>
              {phase === 'COMPLETED' && (
                <div className="mt-4 space-y-1 text-[12px] text-[var(--color-muted)]">
                  <div>
                    Result · <StatusBadge status="COMPLETED" />{' '}
                    <span className="text-amber-200/80">MOCK / DEMO RESULT</span>
                  </div>
                  <div>
                    Verification · <StatusBadge status="VERIFIED" label="SERVER-DERIVED (MOCK)" />
                  </div>
                  <p className="text-[11px]">Browser cannot mark results verified.</p>
                </div>
              )}
            </div>
          </div>

          <div className="space-y-4">
            <SecurityBindingPanel
              title="Security binding"
              fields={{
                grantId: grant.id,
                projectId: grant.projectId,
                scopeId: grant.scopeId,
                taskId: grant.taskId,
                capabilityId: grant.capabilityId,
                packDigest: grant.packContentDigest,
              }}
            />
            <div className="ops-panel p-4 text-[12px] text-[var(--color-muted)]">
              <div className="mb-2 text-[10px] font-semibold uppercase tracking-[0.14em]">Execution path</div>
              <p>Grant → Broker → Host → Executor</p>
              <p className="mt-1">Remote: Grant → Broker → RemoteHost → mTLS Worker → Assignment → Signed Result → Verify</p>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={confirm}
        title="CONFIRM EXECUTION"
        confirmLabel="Confirm execution"
        onCancel={() => setConfirm(false)}
        onConfirm={() => {
          setConfirm(false)
          runMockPipeline()
        }}
        body={
          <div className="space-y-2 text-[13px]">
            <p>
              You are about to submit a security-sensitive operation through the CYBEROS execution path.
            </p>
            <div className="ops-mono space-y-1 text-[11px] text-[var(--color-fg-secondary)]">
              <div>Grant: {grant?.id}</div>
              <div>Capability: {grant?.capabilityId}</div>
              <div>Pack: {grant?.packContentDigest?.slice(0, 28)}…</div>
              <div>Gateway: Broker</div>
              <div className="text-amber-200">Environment: MOCK / DEVELOPMENT</div>
            </div>
            <p className="text-[11px] text-orange-200/90">
              This frontend uses MOCK execution. Infrastructure is not contacted.
            </p>
          </div>
        }
      />
    </div>
  )
}
