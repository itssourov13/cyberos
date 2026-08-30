import type { FormEvent } from 'react'
import { useState } from 'react'
import { api } from '@/api/client'
import type { Decision } from '@/types/models'
import { PageHeader } from '@/components/ui/PageHeader'
import { DecisionPanel } from '@/components/security/DecisionPanel'
import { SecurityContextStrip, SecurityBindingPanel } from '@/components/security/SecurityBindingPanel'
import { SecurityPipeline } from '@/components/security/SecurityPipeline'

export function EvaluatePage() {
  const [projectId, setProjectId] = useState('proj-demo-1')
  const [scopeId, setScopeId] = useState('scope-demo-1')
  const [taskId, setTaskId] = useState('task-demo-1')
  const [capabilityId, setCapabilityId] = useState('cap.scan.network')
  const [decision, setDecision] = useState<Decision | null>(null)
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    try {
      const d = await api.mockEvaluate({ projectId, scopeId, taskId, capabilityId })
      setDecision(d)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title="Evaluate"
        description="Request a Control Plane decision. Server evaluates — browser does not click ALLOW."
      />

      <div className="rounded-[var(--radius-md)] border border-orange-500/30 bg-orange-500/5 px-4 py-3 text-[12px] text-orange-100/90">
        Sensitive surface. MOCK path only — live Control Plane is not contacted. No client trust or
        authorize toggles.
      </div>

      <SecurityPipeline activeIndex={2} />

      <div className="grid gap-4 lg:grid-cols-2">
        <form onSubmit={onSubmit} className="ops-panel space-y-3 p-4">
          <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
            Request parameters
          </div>
          {(
            [
              ['Project', projectId, setProjectId],
              ['Scope', scopeId, setScopeId],
              ['Task', taskId, setTaskId],
              ['Capability', capabilityId, setCapabilityId],
            ] as const
          ).map(([label, value, set]) => (
            <label key={label} className="block text-[11px] uppercase tracking-wider text-[var(--color-muted)]">
              {label}
              <input
                value={value}
                onChange={(e) => set(e.target.value)}
                className="mt-1 w-full rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-2 ops-mono text-[13px] outline-none focus:border-[var(--color-border-focus)]"
              />
            </label>
          ))}
          <button type="submit" disabled={loading} className="ops-btn ops-btn-primary w-full py-2 disabled:opacity-50">
            {loading ? 'Evaluating…' : 'Submit evaluation request (MOCK)'}
          </button>
          <p className="text-[11px] text-[var(--color-muted)]">
            Understand context → request evaluation → review server Decision. No local ALLOW control.
          </p>
        </form>

        <div className="space-y-4">
          <SecurityContextStrip />
          <SecurityBindingPanel
            title="Request context"
            fields={{ projectId, scopeId, taskId, capabilityId }}
          />
        </div>
      </div>

      {decision && <DecisionPanel decision={decision} />}
    </div>
  )
}
