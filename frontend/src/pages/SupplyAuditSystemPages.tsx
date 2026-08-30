import { useEffect, useMemo, useState } from 'react'
import { api } from '@/api/client'
import type { MockAuditRow, MockPack, MockSigner } from '@/api/mock/data'
import { PageHeader, LoadingState, EmptyState, MetricCard } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable, SearchInput, SeverityBadge } from '@/components/ui/DataTable'
import { DigestView, IDView } from '@/components/ui/DigestView'
import { ConceptStrip, SecurityPipeline } from '@/components/security/SecurityPipeline'

export function PacksPage() {
  const [rows, setRows] = useState<MockPack[] | null>(null)
  const [selected, setSelected] = useState<MockPack | null>(null)
  useEffect(() => {
    api.listPacks().then((p) => {
      setRows(p)
      if (p[0]) setSelected(p[0])
    })
  }, [])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Packs"
        description="Supply-chain inventory. Integrity ≠ Signature ≠ Trust ≠ Authorization ≠ Execution."
      />
      <ConceptStrip />
      {!rows && <LoadingState />}
      {rows && rows.length === 0 && <EmptyState title="No packs" />}
      {rows && rows.length > 0 && (
        <DataTable headers={['Name', 'Version', 'Digest', 'Signature', 'Trust', 'Signer', 'Project']}>
          {rows.map((p) => (
            <tr
              key={p.id}
              className="cursor-pointer transition-colors hover:bg-[var(--color-surface-2)]/60"
              onClick={() => setSelected(p)}
            >
              <td className="px-3 py-2.5 font-medium">{p.name}</td>
              <td className="px-3 py-2.5 ops-mono text-[12px]">{p.version}</td>
              <td className="px-3 py-2.5">
                <DigestView value={p.contentDigest} truncate={12} />
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={p.signature} />
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={p.trustState} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{p.signerKeyId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{p.projectId}</td>
            </tr>
          ))}
        </DataTable>
      )}
      {selected && (
        <div className="ops-panel grid gap-4 p-4 sm:grid-cols-2">
          <div>
            <div className="text-[13px] font-semibold">{selected.name}</div>
            <div className="ops-mono text-[12px] text-[var(--color-muted)]">v{selected.version}</div>
            <div className="mt-3">
              <DigestView value={selected.contentDigest} label="Content" />
            </div>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">Integrity</div>
              <StatusBadge status="VERIFIED" />
            </div>
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">Signature</div>
              <StatusBadge status={selected.signature} />
            </div>
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">Trust</div>
              <StatusBadge status={selected.trustState} />
            </div>
            <div>
              <div className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">Signer</div>
              <span className="ops-mono text-[12px]">{selected.signerKeyId}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export function TrustPage() {
  const [packs, setPacks] = useState<MockPack[] | null>(null)
  const [signers, setSigners] = useState<MockSigner[] | null>(null)
  useEffect(() => {
    api.listPacks().then(setPacks)
    api.listSigners().then(setSigners)
  }, [])
  const trusted = packs?.filter((p) => p.trustState === 'TRUSTED').length ?? 0
  const revoked = packs?.filter((p) => p.trustState === 'REVOKED').length ?? 0
  return (
    <div className="space-y-5">
      <PageHeader
        title="Trust"
        description="Server-derived trust visibility. TRUSTED ≠ AUTHORIZED. Mutations require TrustAdmin SERVICE."
      />
      <div className="rounded-[var(--radius-md)] border border-orange-500/30 bg-orange-500/5 px-4 py-3 text-[12px] text-orange-100/90">
        Trust administration is SERVICE-authorized. This interface is read-only in frontend mock mode.
        <button
          type="button"
          disabled
          className="ml-3 cursor-not-allowed rounded border border-[var(--color-border)] px-2 py-1 text-[10px] uppercase opacity-60"
        >
          Revoke pack · Service API required
        </button>
      </div>
      <div className="grid gap-3 sm:grid-cols-4">
        <MetricCard label="Trusted packs" value={trusted} hint="MOCK" />
        <MetricCard label="Revoked packs" value={revoked} />
        <MetricCard label="Signers" value={signers?.length ?? '—'} />
        <MetricCard label="Mode" value="READ-ONLY" />
      </div>
      <SecurityPipeline activeIndex={2} />
      <div>
        <h2 className="mb-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
          Signer keys (public metadata only)
        </h2>
        {!signers && <LoadingState />}
        {signers && (
          <DataTable headers={['Key ID', 'Algorithm', 'Status', 'Public fingerprint']}>
            {signers.map((s) => (
              <tr key={s.keyId} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
                <td className="px-3 py-2.5">
                  <IDView value={s.keyId} />
                </td>
                <td className="px-3 py-2.5 text-[12px]">{s.algorithm}</td>
                <td className="px-3 py-2.5">
                  <StatusBadge status={s.status} />
                </td>
                <td className="px-3 py-2.5">
                  <DigestView value={s.publicFingerprint} />
                </td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>
      <p className="text-[11px] text-[var(--color-muted)]">
        Private keys are never displayed. TRUSTED means server trust policy — not execution authorization.
      </p>
    </div>
  )
}

export function AuditPage() {
  const [rows, setRows] = useState<MockAuditRow[] | null>(null)
  const [q, setQ] = useState('')
  const [type, setType] = useState('ALL')
  const [selected, setSelected] = useState<MockAuditRow | null>(null)
  useEffect(() => {
    api.listAudit({ q, type }).then(setRows)
  }, [q, type])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Audit"
        description="Observational forensics timeline. MOCK / DEMO events labeled. Non-authoritative."
      />
      <div className="ops-panel px-4 py-3 text-[12px] text-[var(--color-muted)]">
        Correlation: Task → Decision → Grant → Execution → Assignment → Worker → Result → Audit
      </div>
      <div className="flex flex-wrap gap-2">
        <SearchInput value={q} onChange={setQ} placeholder="Correlation, grant, worker…" />
        <select
          value={type}
          onChange={(e) => setType(e.target.value)}
          className="h-[34px] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2 text-[13px]"
        >
          {[
            'ALL',
            'DECISION_ALLOWED',
            'DECISION_DENIED',
            'GRANT_ISSUED',
            'ASSIGNMENT_CLAIMED',
            'RESULT_VERIFIED',
          ].map((t) => (
            <option key={t}>{t}</option>
          ))}
        </select>
      </div>
      {!rows && <LoadingState label="Loading audit timeline…" />}
      {rows && rows.length === 0 && <EmptyState title="No audit events" />}
      {rows && rows.length > 0 && (
        <DataTable headers={['Time', 'Type', 'Severity', 'Project', 'Correlation', 'Message', '']}>
          {rows.map((r) => (
            <tr key={r.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5 ops-mono text-[11px]">
                {new Date(r.timestamp).toLocaleString()}
              </td>
              <td className="px-3 py-2.5 text-[12px]">{r.type}</td>
              <td className="px-3 py-2.5">
                <SeverityBadge
                  severity={r.severity === 'SUCCESS' ? 'LOW' : r.severity === 'WARN' ? 'MEDIUM' : 'INFO'}
                />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{r.projectId}</td>
              <td className="px-3 py-2.5">
                <IDView value={r.correlationId} />
              </td>
              <td className="px-3 py-2.5 text-[12px]">
                {r.message}
                {r.mock && <span className="ml-2 text-[10px] text-amber-300">MOCK</span>}
              </td>
              <td className="px-3 py-2.5">
                <button
                  type="button"
                  className="text-[12px] text-sky-300/90 hover:underline"
                  onClick={() => setSelected(r)}
                >
                  Detail
                </button>
              </td>
            </tr>
          ))}
        </DataTable>
      )}
      {selected && (
        <div className="fixed inset-y-0 right-0 z-40 w-full max-w-md border-l border-[var(--color-border)] bg-[var(--color-surface-1)] p-5 shadow-2xl">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="text-sm font-semibold">Event detail</h2>
              <p className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">
                Observational · Non-authoritative
              </p>
            </div>
            <button type="button" className="text-[12px] text-[var(--color-muted)]" onClick={() => setSelected(null)}>
              Close
            </button>
          </div>
          <dl className="space-y-2.5 text-[12px]">
            <Row k="Type" v={selected.type} />
            <Row k="Correlation" v={selected.correlationId} mono />
            {selected.grantId && <Row k="Grant" v={selected.grantId} mono />}
            {selected.assignmentId && <Row k="Assignment" v={selected.assignmentId} mono />}
            {selected.workerId && <Row k="Worker" v={selected.workerId} mono />}
            <Row k="Actor" v={selected.actor} />
            <Row k="Message" v={selected.message} />
            {selected.mock && <Row k="Source" v="MOCK / DEMO" />}
          </dl>
        </div>
      )}
    </div>
  )
}

function Row({ k, v, mono }: { k: string; v: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{k}</dt>
      <dd className={mono ? 'ops-mono mt-0.5' : 'mt-0.5'}>{v}</dd>
    </div>
  )
}

export function SystemPage() {
  const [dash, setDash] = useState<Awaited<ReturnType<typeof api.getDashboard>> | null>(null)
  useEffect(() => {
    api.getDashboard().then(setDash)
  }, [])
  return (
    <div className="space-y-5">
      <PageHeader title="System" description="Operational health console. DEVELOPMENT · MOCK API." />
      {!dash && <LoadingState />}
      {dash && (
        <>
          <div className="ops-panel divide-y divide-[var(--color-border)]">
            {dash.health.map((h) => (
              <div key={h.name} className="flex items-center justify-between gap-3 px-4 py-2.5">
                <div>
                  <div className="text-[13px] font-medium">{h.name}</div>
                  {h.detail && <div className="text-[11px] text-[var(--color-muted)]">{h.detail}</div>}
                </div>
                <StatusBadge status={h.state} />
              </div>
            ))}
          </div>
          <div className="ops-panel p-4 ops-mono text-[11px] leading-relaxed text-[var(--color-muted)]">
            <pre className="whitespace-pre-wrap">{`CYBEROS
 ├── Control Plane → Decision
 ├── Broker → Grant → Host
 │     ├── Local Executor
 │     └── Remote mTLS Worker → Assignment → Signed Result → Verify
 └── Audit`}</pre>
          </div>
        </>
      )}
    </div>
  )
}

export function DoctorPage() {
  const checks = useMemo(
    () => [
      { name: 'API', status: 'PASS' as const, note: 'MOCK health ok' },
      { name: 'Control Plane', status: 'PASS' as const, note: 'UI wiring only — not live CP' },
      { name: 'Broker', status: 'PASS' as const, note: 'UI wiring only — not live Broker' },
      { name: 'PostgreSQL', status: 'UNKNOWN' as const, note: 'DSN not verified from browser' },
      { name: 'Remote Worker', status: 'UNKNOWN' as const, note: 'Live fleet not contacted' },
      { name: 'mTLS Fleet', status: 'UNKNOWN' as const, note: 'No live CA in UI' },
      { name: 'Trust store', status: 'WARN' as const, note: 'MOCK read-only' },
    ],
    [],
  )
  return (
    <div className="space-y-4">
      <PageHeader
        title="Doctor"
        description="Diagnostics. UNKNOWN is never treated as PASS when infrastructure is unavailable."
      />
      <ul className="ops-panel divide-y divide-[var(--color-border)]">
        {checks.map((c) => (
          <li key={c.name} className="flex items-center justify-between gap-3 px-4 py-3">
            <div>
              <div className="text-[13px] font-medium">{c.name}</div>
              <div className="text-[11px] text-[var(--color-muted)]">{c.note}</div>
            </div>
            <StatusBadge
              status={c.status === 'PASS' ? 'HEALTHY' : c.status === 'WARN' ? 'DEGRADED' : 'UNKNOWN'}
              label={c.status}
            />
          </li>
        ))}
      </ul>
    </div>
  )
}
