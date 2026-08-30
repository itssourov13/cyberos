import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { DashboardSnapshot } from '@/types/models'
import { PageHeader, MetricCard, LoadingState, ErrorState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { SecurityPipeline, ConceptStrip } from '@/components/security/SecurityPipeline'
import { IDView } from '@/components/ui/DigestView'

export function DashboardPage() {
  const [data, setData] = useState<DashboardSnapshot | null>(null)
  const [err, setErr] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .getDashboard()
      .then((d) => {
        if (!cancelled) setData(d)
      })
      .catch(() => {
        if (!cancelled) setErr('BACKEND_UNAVAILABLE')
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (err) return <ErrorState title="Dashboard unavailable" reason={err} />
  if (!data) return <LoadingState label="Loading operations overview…" />

  return (
    <div className="space-y-6">
      <PageHeader
        title="Operations"
        description="Security operations overview. Server-derived status only. MOCK data in Development mode."
        actions={
          <div className="flex flex-wrap gap-2 text-[11px]">
            <Link className="ops-btn ops-btn-secondary" to="/projects">
              Projects
            </Link>
            <Link className="ops-btn ops-btn-secondary" to="/tasks">
              Tasks
            </Link>
            <Link className="ops-btn ops-btn-secondary" to="/workers">
              Workers
            </Link>
            <Link className="ops-btn ops-btn-secondary" to="/findings">
              Findings
            </Link>
          </div>
        }
      />

      <section>
        <SectionLabel>Security concepts</SectionLabel>
        <ConceptStrip />
      </section>

      <section>
        <SectionLabel>Execution pipeline</SectionLabel>
        <SecurityPipeline activeIndex={4} />
      </section>

      <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Active workers" value={data.activeWorkers} hint="MOCK sample" />
        <MetricCard label="Running assignments" value={data.runningAssignments} />
        <MetricCard label="Pending assignments" value={data.pendingAssignments} />
        <MetricCard label="Findings (recent)" value={data.recentFindings} />
      </section>

      <section className="grid gap-3 lg:grid-cols-2">
        <div>
          <SectionLabel>System health</SectionLabel>
          <div className="ops-panel divide-y divide-[var(--color-border)]">
            {data.health.map((h) => (
              <div key={h.name} className="flex items-center justify-between gap-3 px-4 py-2.5">
                <div className="min-w-0">
                  <div className="text-[13px] font-medium text-[var(--color-fg)]">{h.name}</div>
                  {h.detail && (
                    <div className="truncate text-[11px] text-[var(--color-muted)]">{h.detail}</div>
                  )}
                </div>
                <StatusBadge status={h.state} />
              </div>
            ))}
          </div>
        </div>
        <div>
          <SectionLabel>Fleet & trust</SectionLabel>
          <div className="grid gap-3 sm:grid-cols-2">
            <MetricCard label="Trusted packs" value={data.trustedPacks} hint="Trust ≠ Authorization" />
            <MetricCard label="Denied decisions" value={data.deniedDecisions} hint="MOCK" />
          </div>
        </div>
      </section>

      <section>
        <SectionLabel>Recent activity</SectionLabel>
        <ul className="ops-panel divide-y divide-[var(--color-border)]">
          {data.recentActivity.map((e) => (
            <li key={e.id} className="flex flex-col gap-1 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-[var(--color-muted)]">
                    {e.type}
                  </span>
                  <span className="text-[13px] text-[var(--color-fg-secondary)]">{e.message}</span>
                </div>
                <div className="mt-1 text-[11px] text-[var(--color-muted)]">
                  {new Date(e.timestamp).toLocaleString()}
                  {e.projectId && <span className="ml-2 ops-mono">{e.projectId}</span>}
                </div>
              </div>
              {e.correlationId && (
                <div className="shrink-0">
                  <IDView value={e.correlationId} label="corr" />
                </div>
              )}
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="mb-2.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
      {children}
    </h2>
  )
}
