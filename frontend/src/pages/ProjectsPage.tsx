import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { Project } from '@/types/models'
import { PageHeader, LoadingState, EmptyState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable, SearchInput } from '@/components/ui/DataTable'
import { IDView } from '@/components/ui/DigestView'

export function ProjectsPage() {
  const [rows, setRows] = useState<Project[] | null>(null)
  const [q, setQ] = useState('')
  useEffect(() => {
    api.listProjects().then(setRows)
  }, [])
  const filtered = useMemo(() => {
    if (!rows) return []
    const s = q.toLowerCase()
    return rows.filter((p) => p.name.toLowerCase().includes(s) || p.id.toLowerCase().includes(s))
  }, [rows, q])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Projects"
        description="Workspace projects. Dense catalog view. MOCK data — not live catalog."
      />
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput value={q} onChange={setQ} placeholder="Search projects…" />
        <span className="text-[11px] text-[var(--color-muted)]">
          {filtered.length} project{filtered.length === 1 ? '' : 's'}
        </span>
      </div>
      {!rows && <LoadingState />}
      {rows && filtered.length === 0 && (
        <EmptyState title="No projects" description="No matching projects for the current filter." />
      )}
      {filtered.length > 0 && (
        <DataTable headers={['Project', 'ID', 'Status', '']}>
          {filtered.map((p) => (
            <tr key={p.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5 font-medium text-[var(--color-fg)]">{p.name}</td>
              <td className="px-3 py-2.5">
                <IDView value={p.id} />
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={p.status} />
              </td>
              <td className="px-3 py-2.5">
                <Link className="text-[12px] text-sky-300/90 hover:underline" to={`/projects/${p.id}`}>
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
