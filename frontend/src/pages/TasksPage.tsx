import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { MockTask } from '@/api/mock/data'
import { PageHeader, LoadingState, EmptyState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable, SearchInput, SeverityBadge } from '@/components/ui/DataTable'
import { IDView } from '@/components/ui/DigestView'

export function TasksPage() {
  const [rows, setRows] = useState<MockTask[] | null>(null)
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('ALL')
  useEffect(() => {
    api.listTasks().then(setRows)
  }, [])
  const filtered = useMemo(() => {
    if (!rows) return []
    return rows.filter((t) => {
      if (status !== 'ALL' && t.status !== status) return false
      const s = q.toLowerCase()
      return !s || t.name.toLowerCase().includes(s) || t.id.includes(s)
    })
  }, [rows, q, status])
  return (
    <div className="space-y-4">
      <PageHeader title="Tasks" description="Task catalog. Status is display-only MOCK data — not live execution state." />
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput value={q} onChange={setQ} placeholder="Search tasks…" />
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="h-[34px] rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-2 text-[13px] outline-none focus:border-[var(--color-border-focus)]"
          aria-label="Status filter"
        >
          {['ALL', 'PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'DENIED', 'CANCELLED'].map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <span className="text-[11px] text-[var(--color-muted)]">{filtered.length} shown</span>
      </div>
      {!rows && <LoadingState />}
      {rows && filtered.length === 0 && <EmptyState title="No tasks" description="No tasks match the current filters." />}
      {filtered.length > 0 && (
        <DataTable headers={['Task', 'ID', 'Project', 'Scope', 'Priority', 'Status', '']}>
          {filtered.map((t) => (
            <tr key={t.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5 font-medium">{t.name}</td>
              <td className="px-3 py-2.5">
                <IDView value={t.id} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{t.projectId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{t.scopeId}</td>
              <td className="px-3 py-2.5">
                <SeverityBadge severity={t.priority} />
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={t.status} />
              </td>
              <td className="px-3 py-2.5">
                <Link to={`/tasks/${t.id}`} className="text-[12px] text-sky-300/90 hover:underline">
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
