import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import type { MockScope } from '@/api/mock/data'
import { PageHeader, LoadingState, EmptyState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable } from '@/components/ui/DataTable'
import { IDView } from '@/components/ui/DigestView'

export function ScopesPage() {
  const [rows, setRows] = useState<MockScope[] | null>(null)
  useEffect(() => {
    api.listScopes().then(setRows)
  }, [])
  return (
    <div className="space-y-4">
      <PageHeader
        title="Scopes"
        description="Authorization boundaries. UI does not decide authorization — server Scope remains authoritative."
      />
      <p className="text-[12px] text-[var(--color-muted)]">
        Expired or out-of-scope entries are not executable. Status is display-only.
      </p>
      {!rows && <LoadingState />}
      {rows && rows.length === 0 && <EmptyState title="No scopes" />}
      {rows && rows.length > 0 && (
        <DataTable headers={['Name', 'ID', 'Project', 'Target', 'Type', 'Status', 'Tasks']}>
          {rows.map((s) => (
            <tr key={s.id} className="transition-colors hover:bg-[var(--color-surface-2)]/60">
              <td className="px-3 py-2.5 font-medium">{s.name}</td>
              <td className="px-3 py-2.5">
                <IDView value={s.id} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{s.projectId}</td>
              <td className="px-3 py-2.5 ops-mono">{s.target}</td>
              <td className="px-3 py-2.5 text-[12px] text-[var(--color-muted)]">{s.type}</td>
              <td className="px-3 py-2.5">
                <StatusBadge status={s.status} />
              </td>
              <td className="px-3 py-2.5 tabular-nums">{s.taskCount}</td>
            </tr>
          ))}
        </DataTable>
      )}
    </div>
  )
}
