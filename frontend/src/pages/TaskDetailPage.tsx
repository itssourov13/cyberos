import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { MockTask } from '@/api/mock/data'
import { PageHeader, LoadingState, ErrorState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { IDView } from '@/components/ui/DigestView'
import { SecurityPipeline } from '@/components/security/SecurityPipeline'

export function TaskDetailPage() {
  const { id } = useParams()
  const [task, setTask] = useState<MockTask | null | undefined>(undefined)
  useEffect(() => {
    if (!id) return
    api.getTask(id).then(setTask)
  }, [id])
  if (task === undefined) return <LoadingState />
  if (!task) return <ErrorState title="Task not found" reason="UNKNOWN_TASK" />
  return (
    <div className="space-y-5">
      <PageHeader
        title={task.name}
        description="Task identity is not a Grant and does not authorize execution."
        actions={
          <Link to="/evaluate" className="ops-btn ops-btn-secondary text-[12px]">
            Evaluate
          </Link>
        }
      />
      <div className="flex flex-wrap gap-3">
        <StatusBadge status={task.status} />
        <IDView value={task.id} label="Task" />
        <IDView value={task.projectId} label="Project" />
        <IDView value={task.scopeId} label="Scope" />
      </div>
      <div className="ops-panel px-4 py-3 text-[13px] text-[var(--color-muted)]">
        Capability <span className="ops-mono text-sky-200/90">{task.capabilityId}</span>
        <span className="mx-2 text-[var(--color-border-strong)]">·</span>
        Priority <span className="text-[var(--color-fg-secondary)]">{task.priority}</span>
      </div>
      <SecurityPipeline activeIndex={0} />
    </div>
  )
}
