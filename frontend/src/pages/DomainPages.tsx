import type { ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { api } from '@/api/client'
import type {
  MockAssessment,
  MockAsset,
  MockCase,
  MockFinding,
  MockReport,
  MockRisk,
} from '@/api/mock/data'
import { PageHeader, LoadingState, EmptyState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable, SearchInput, SeverityBadge } from '@/components/ui/DataTable'
import { IDView } from '@/components/ui/DigestView'

function useList<T>(loader: () => Promise<T[]>) {
  const [rows, setRows] = useState<T[] | null>(null)
  useEffect(() => {
    let cancelled = false
    loader().then((data) => {
      if (!cancelled) setRows(data)
    })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- stable page-level loaders
  }, [])
  return rows
}

export function AssessmentsPage() {
  const rows = useList(() => api.listAssessments())
  return (
    <ListPage
      title="Assessments"
      description="Assessment orchestration overview (MOCK). Security assessment data, not a project board."
      rows={rows}
      headers={['ID', 'Project', 'Target', 'Status', 'Progress', 'Findings']}
      render={(r: MockAssessment) => (
        <tr key={r.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
          <td className="px-3 py-2.5">
            <IDView value={r.id} />
          </td>
          <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{r.projectId}</td>
          <td className="px-3 py-2.5 ops-mono">{r.target}</td>
          <td className="px-3 py-2.5">
            <StatusBadge status={r.status} />
          </td>
          <td className="px-3 py-2.5 tabular-nums">{r.progress}%</td>
          <td className="px-3 py-2.5 tabular-nums">{r.findings}</td>
        </tr>
      )}
    />
  )
}

export function FindingsPage() {
  const rows = useList(() => api.listFindings())
  const [q, setQ] = useState('')
  const [sev, setSev] = useState('ALL')
  const filtered = useMemo(() => {
    if (!rows) return []
    return rows.filter((f) => {
      if (sev !== 'ALL' && f.severity !== sev) return false
      const s = q.toLowerCase()
      return !s || f.title.toLowerCase().includes(s) || f.id.includes(s)
    })
  }, [rows, q, sev])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Findings"
        description="Finding inventory prioritized by severity. Evidence is text-only; no unsafe HTML."
      />
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput value={q} onChange={setQ} placeholder="Search findings…" />
        <select
          value={sev}
          onChange={(e) => setSev(e.target.value)}
          className="h-[34px] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2 text-[13px] outline-none focus:border-[var(--color-border-focus)]"
        >
          {['ALL', 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO'].map((s) => (
            <option key={s}>{s}</option>
          ))}
        </select>
      </div>
      {!rows && <LoadingState />}
      {rows && filtered.length === 0 && <EmptyState title="No findings" description="No findings match the current filters." />}
      {filtered.length > 0 && (
        <DataTable headers={['Title', 'ID', 'Severity', 'Status', 'Asset', 'Project']}>
          {filtered.map((f: MockFinding) => (
            <tr key={f.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5 font-medium">{f.title}</td>
              <td className="px-3 py-2.5">
                <IDView value={f.id} />
              </td>
              <td className="px-3 py-2.5">
                <SeverityBadge severity={f.severity} />
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={f.status} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{f.assetId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{f.projectId}</td>
            </tr>
          ))}
        </DataTable>
      )}
    </div>
  )
}

export function AssetsPage() {
  const rows = useList(() => api.listAssets())
  return (
    <ListPage
      title="Assets"
      description="Asset inventory (MOCK). Technical identifiers in monospace."
      rows={rows}
      headers={['Hostname', 'ID', 'Address', 'Type', 'Status', 'Project']}
      render={(a: MockAsset) => (
        <tr key={a.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
          <td className="px-3 py-2.5 font-medium">{a.hostname}</td>
          <td className="px-3 py-2.5">
            <IDView value={a.id} />
          </td>
          <td className="px-3 py-2.5 ops-mono">{a.address}</td>
          <td className="px-3 py-2.5 text-[12px] text-[var(--color-muted)]">{a.type}</td>
          <td className="px-3 py-2.5">
            <StatusBadge status={a.status} />
          </td>
          <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{a.projectId}</td>
        </tr>
      )}
    />
  )
}

export function RisksPage() {
  const rows = useList(() => api.listRisks())
  return (
    <ListPage
      title="Risks"
      description="Risk overview. Scores are MOCK/demo — not live calculations."
      rows={rows}
      headers={['Title', 'ID', 'Severity', 'Likelihood', 'Impact', 'Status']}
      render={(r: MockRisk) => (
        <tr key={r.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
          <td className="px-3 py-2.5 font-medium">{r.title}</td>
          <td className="px-3 py-2.5">
            <IDView value={r.id} />
          </td>
          <td className="px-3 py-2.5">
            <SeverityBadge severity={r.severity} />
          </td>
          <td className="px-3 py-2.5 text-[12px]">{r.likelihood}</td>
          <td className="px-3 py-2.5 text-[12px]">{r.impact}</td>
          <td className="px-3 py-2.5">
            <StatusBadge status={r.status} />
          </td>
        </tr>
      )}
    />
  )
}

export function CasesPage() {
  const rows = useList(() => api.listCases())
  return (
    <ListPage
      title="Cases"
      description="Investigation cases (MOCK). Security investigation objects, not generic tickets."
      rows={rows}
      headers={['Title', 'ID', 'Status', 'Priority', 'Owner', 'Project']}
      render={(c: MockCase) => (
        <tr key={c.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
          <td className="px-3 py-2.5 font-medium">{c.title}</td>
          <td className="px-3 py-2.5">
            <IDView value={c.id} />
          </td>
          <td className="px-3 py-2.5">
            <StatusBadge status={c.status} />
          </td>
          <td className="px-3 py-2.5">
            <SeverityBadge severity={c.priority} />
          </td>
          <td className="px-3 py-2.5 text-[12px]">{c.owner}</td>
          <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{c.projectId}</td>
        </tr>
      )}
    />
  )
}

export function ReportsPage() {
  const rows = useList(() => api.listReports())
  return (
    <ListPage
      title="Reports"
      description="Report listing. Preview only — not cryptographically verified unless the server says so."
      rows={rows}
      headers={['Title', 'ID', 'Type', 'Project', 'Generated', 'Status']}
      render={(r: MockReport) => (
        <tr key={r.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
          <td className="px-3 py-2.5 font-medium">{r.title}</td>
          <td className="px-3 py-2.5">
            <IDView value={r.id} />
          </td>
          <td className="px-3 py-2.5 text-[12px] text-[var(--color-muted)]">{r.type}</td>
          <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{r.projectId}</td>
          <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{r.generatedAt}</td>
          <td className="px-3 py-2.5">
            <StatusBadge status={r.status} />
          </td>
        </tr>
      )}
    />
  )
}

function ListPage<T>({
  title,
  description,
  rows,
  headers,
  render,
}: {
  title: string
  description: string
  rows: T[] | null
  headers: string[]
  render: (row: T) => ReactNode
}) {
  return (
    <div className="space-y-4">
      <PageHeader title={title} description={description} />
      {!rows && <LoadingState />}
      {rows && rows.length === 0 && <EmptyState title={`No ${title.toLowerCase()}`} />}
      {rows && rows.length > 0 && <DataTable headers={headers}>{rows.map(render)}</DataTable>}
    </div>
  )
}
