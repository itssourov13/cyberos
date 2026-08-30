import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import type { Grant } from '@/types/models'
import { PageHeader, LoadingState, EmptyState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { DataTable } from '@/components/ui/DataTable'
import { DigestView, IDView } from '@/components/ui/DigestView'
import { GrantBrokerChain, SecurityBindingPanel } from '@/components/security/SecurityBindingPanel'
import { SecurityPipeline } from '@/components/security/SecurityPipeline'

export function GrantsPage() {
  const [rows, setRows] = useState<Grant[] | null>(null)
  const [selected, setSelected] = useState<Grant | null>(null)
  useEffect(() => {
    api.listGrants().then((g) => {
      setRows(g)
      if (g[0]) setSelected(g[0])
    })
  }, [])
  return (
    <div className="space-y-5">
      <PageHeader
        title="Grants"
        description="Server-issued Execution Grants. Display-only. PackContentDigest is not editable."
      />
      <div className="rounded-[var(--radius-md)] border border-sky-500/25 bg-sky-500/5 px-4 py-3 text-[12px] text-sky-100/90">
        Grant is server-issued. The browser cannot create, modify, or authorize a Grant.
      </div>
      <SecurityPipeline activeIndex={4} />
      {!rows && <LoadingState />}
      {rows && rows.length === 0 && <EmptyState title="No grants" description="No server-issued grants in MOCK data." />}
      {rows && rows.length > 0 && (
        <DataTable headers={['Grant', 'Project', 'Scope', 'Capability', 'Pack digest', 'Status', 'Expires', '']}>
          {rows.map((g) => (
            <tr
              key={g.id}
              className="cursor-pointer transition-colors hover:bg-[var(--color-surface-2)]/60"
              onClick={() => setSelected(g)}
            >
              <td className="px-3 py-2.5">
                <IDView value={g.id} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{g.projectId}</td>
              <td className="px-3 py-2.5 ops-mono text-[var(--color-muted)]">{g.scopeId}</td>
              <td className="px-3 py-2.5 ops-mono text-sky-200/80">{g.capabilityId}</td>
              <td className="px-3 py-2.5">
                {g.packContentDigest ? <DigestView value={g.packContentDigest} truncate={12} /> : '—'}
              </td>
              <td className="px-3 py-2.5">
                <StatusBadge status={g.status} />
              </td>
              <td className="px-3 py-2.5 ops-mono text-[11px] text-[var(--color-muted)]">{g.expiresAt}</td>
              <td className="px-3 py-2.5 text-[12px] text-sky-300/90">Detail</td>
            </tr>
          ))}
        </DataTable>
      )}
      {selected && (
        <div className="space-y-4">
          <GrantBrokerChain grantId={selected.id} decisionId="dec-demo-1" />
          <SecurityBindingPanel
            title="Grant bindings"
            fields={{
              grantId: selected.id,
              projectId: selected.projectId,
              scopeId: selected.scopeId,
              taskId: selected.taskId,
              capabilityId: selected.capabilityId,
              packDigest: selected.packContentDigest,
            }}
          />
        </div>
      )}
    </div>
  )
}
