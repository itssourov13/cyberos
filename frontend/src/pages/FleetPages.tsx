import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '@/api/client'
import type { Assignment, Worker } from '@/types/models'
import { PageHeader, LoadingState, EmptyState, ErrorState, MetricCard } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable, SearchInput } from '@/components/ui/DataTable'
import { DigestView, IDView } from '@/components/ui/DigestView'
import { SecurityPipeline } from '@/components/security/SecurityPipeline'
import { SecurityBindingPanel } from '@/components/security/SecurityBindingPanel'

export function WorkersPage() {
  const [rows, setRows] = useState<Worker[] | null>(null)
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('ALL')
  useEffect(() => {
    api.listWorkers().then(setRows)
  }, [])
  const filtered = useMemo(() => {
    if (!rows) return []
    return rows.filter((w) => {
      if (status !== 'ALL' && w.status !== status) return false
      const s = q.toLowerCase()
      return !s || w.id.toLowerCase().includes(s) || w.projectId.includes(s) || w.hostId.includes(s)
    })
  }, [rows, q, status])
  const counts = useMemo(() => {
    if (!rows) return { active: 0, disabled: 0, revoked: 0, total: 0 }
    return {
      active: rows.filter((w) => w.status === 'ACTIVE').length,
      disabled: rows.filter((w) => w.status === 'DISABLED').length,
      revoked: rows.filter((w) => w.status === 'REVOKED').length,
      total: rows.length,
    }
  }, [rows])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Worker fleet"
        description="Remote execution workers. mTLS authenticates identity — it does not authorize execution."
      />
      <div className="grid gap-3 sm:grid-cols-4">
        <MetricCard label="Active" value={counts.active} hint="MOCK" />
        <MetricCard label="Disabled" value={counts.disabled} />
        <MetricCard label="Revoked" value={counts.revoked} />
        <MetricCard label="Total" value={counts.total} />
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput value={q} onChange={setQ} placeholder="Search workers…" />
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="h-[34px] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2 text-[13px]"
        >
          {['ALL', 'ACTIVE', 'DISABLED', 'REVOKED', 'UNKNOWN'].map((s) => (
            <option key={s}>{s}</option>
          ))}
        </select>
      </div>
      {!rows && <LoadingState label="Loading worker fleet…" />}
      {rows && filtered.length === 0 && (
        <EmptyState title="No workers found" description="No workers match the current filter." />
      )}
      {filtered.length > 0 && (
        <DataTable headers={['Worker', 'Project', 'Host', 'Status', 'Identity', 'Fingerprint', 'Last seen', '']}>
          {filtered.map((w) => (
            <tr key={w.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5">
                <IDView value={w.id} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{w.projectId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{w.hostId}</td>
              <td className="px-3 py-2.5">
                <StatusBadge status={w.status} />
              </td>
              <td className="px-3 py-2.5 text-[11px] text-[var(--color-muted)]">
                {w.certFingerprint ? 'mTLS ✓' : 'UNKNOWN'}
              </td>
              <td className="px-3 py-2.5">
                {w.certFingerprint ? <DigestView value={w.certFingerprint} truncate={10} /> : '—'}
              </td>
              <td className="px-3 py-2.5 text-[11px] text-[var(--color-muted)]">
                {w.lastSeenAt ? new Date(w.lastSeenAt).toLocaleString() : '—'}
              </td>
              <td className="px-3 py-2.5">
                <Link to={`/workers/${w.id}`} className="text-[12px] text-sky-300/90 hover:underline">
                  Open
                </Link>
              </td>
            </tr>
          ))}
        </DataTable>
      )}
    </div>
  )
}

export function WorkerDetailPage() {
  const { id } = useParams()
  const [w, setW] = useState<Worker | null | undefined>(undefined)
  useEffect(() => {
    if (!id) return
    api.getWorker(id).then(setW)
  }, [id])
  if (w === undefined) return <LoadingState />
  if (!w) return <ErrorState title="Worker not found" reason="UNKNOWN_WORKER" />
  return (
    <div className="space-y-5">
      <PageHeader title={w.id} description="Worker identity and security context (display only)." />
      <div className="flex flex-wrap gap-3">
        <StatusBadge status={w.status} />
        <IDView value={w.projectId} label="Project" />
        <IDView value={w.hostId} label="Host" />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <div className="ops-panel p-4 text-[13px]">
          <div className="mb-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
            mTLS identity chain
          </div>
          <ol className="list-decimal space-y-1.5 pl-5 text-[12px] text-[var(--color-muted)]">
            <li>Controller ↔ Worker TLS 1.3</li>
            <li>Verified client certificate</li>
            <li>SHA-256 fingerprint</li>
            <li>Worker registry → WorkerID + ProjectID + HostID</li>
          </ol>
          <p className="mt-3 text-[12px] text-sky-200/90">
            mTLS authenticates the worker. It does not authorize execution.
          </p>
          {w.certFingerprint && (
            <div className="mt-3">
              <DigestView value={w.certFingerprint} label="Fingerprint" />
            </div>
          )}
          <p className="mt-2 text-[11px] text-[var(--color-muted)]">Credential: CRED-**** (reference only)</p>
        </div>
        <SecurityBindingPanel
          title="Registry binding"
          fields={{ projectId: w.projectId, workerId: w.id, hostId: w.hostId }}
        />
      </div>
    </div>
  )
}

export function AssignmentsPage() {
  const [rows, setRows] = useState<Assignment[] | null>(null)
  useEffect(() => {
    api.listAssignments().then(setRows)
  }, [])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Assignments"
        description="Execution evidence surface. Assignment binds GrantID + PackContentDigest + Worker/Host."
      />
      {!rows && <LoadingState />}
      {rows && rows.length === 0 && <EmptyState title="No assignments" />}
      {rows && rows.length > 0 && (
        <DataTable headers={['Assignment', 'Grant', 'Worker', 'Host', 'Pack', 'Status', 'Lease', '']}>
          {rows.map((a) => (
            <tr key={a.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5">
                <IDView value={a.id} />
              </td>
              <td className="px-3 py-2.5">
                <IDView value={a.grantId} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{a.workerId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{a.hostId}</td>
              <td className="px-3 py-2.5">
                {a.packContentDigest ? <DigestView value={a.packContentDigest} truncate={10} /> : '—'}
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={a.status} />
              </td>
              <td className="px-3 py-2.5 text-[11px] text-[var(--color-muted)]">
                {a.leaseUntil ? new Date(a.leaseUntil).toLocaleTimeString() : '—'}
              </td>
              <td className="px-3 py-2.5">
                <Link to={`/assignments/${a.id}`} className="text-[12px] text-sky-300/90 hover:underline">
                  Open
                </Link>
              </td>
            </tr>
          ))}
        </DataTable>
      )}
    </div>
  )
}

export function AssignmentDetailPage() {
  const { id } = useParams()
  const [a, setA] = useState<Assignment | null | undefined>(undefined)
  useEffect(() => {
    if (!id) return
    api.getAssignment(id).then(setA)
  }, [id])
  if (a === undefined) return <LoadingState />
  if (!a) return <ErrorState title="Assignment not found" reason="UNKNOWN_ASSIGNMENT" />
  const pipeline = ['PENDING', 'CLAIMED', 'RUNNING', 'COMPLETED'] as const
  return (
    <div className="space-y-5">
      <PageHeader title={a.id} description="Assignment lifecycle and security bindings (MOCK, read-only)." />
      <div className="flex flex-wrap gap-2">
        <StatusBadge status={a.status} />
        <IDView value={a.grantId} label="Grant" />
        <IDView value={a.workerId} label="Worker" />
        <IDView value={a.hostId} label="Host" />
      </div>
      <div className="ops-panel p-4">
        <div className="mb-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
          Lifecycle (display)
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          {pipeline.map((s, i) => (
            <div key={s} className="flex items-center gap-1.5">
              <span
                className={`rounded-[var(--radius-sm)] border px-2 py-1 text-[10px] font-medium ${
                  a.status === s || (a.status === 'RUNNING' && s === 'RUNNING')
                    ? 'border-[var(--color-accent)]/50 bg-[var(--color-accent)]/12 text-sky-100'
                    : 'border-[var(--color-border)] text-[var(--color-muted)]'
                }`}
              >
                {s}
              </span>
              {i < pipeline.length - 1 && <span className="text-[var(--color-muted)]">→</span>}
            </div>
          ))}
        </div>
      </div>
      {a.leaseUntil && (
        <div className="ops-panel-elevated px-4 py-3 text-[12px]">
          <span className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">Lease</span>
          <div className="mt-1 ops-mono text-[var(--color-fg-secondary)]">
            Until {new Date(a.leaseUntil).toLocaleString()} · display only
          </div>
        </div>
      )}
      <SecurityBindingPanel
        title="Assignment bindings"
        fields={{
          grantId: a.grantId,
          projectId: a.projectId,
          workerId: a.workerId,
          hostId: a.hostId,
          packDigest: a.packContentDigest,
        }}
      />
      <SecurityPipeline activeIndex={6} />
      <div className="ops-panel p-4 text-[12px]">
        <div className="mb-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
          Result security (MOCK display)
        </div>
        <div className="grid gap-1 sm:grid-cols-2 text-[var(--color-muted)]">
          <div>
            Signature · <StatusBadge status="VERIFIED" />
          </div>
          <div>
            Replay · <StatusBadge status="VALID" label="NOT DETECTED" />
          </div>
          <div>
            Worker · <StatusBadge status="VALID" label="MATCHED" />
          </div>
          <div>
            Assignment · <StatusBadge status="VALID" label="MATCHED" />
          </div>
        </div>
        <p className="mt-3 text-[11px] text-[var(--color-muted)]">
          Browser cannot mark results verified. Server VerifyCompletion is authoritative.
        </p>
      </div>
    </div>
  )
}
