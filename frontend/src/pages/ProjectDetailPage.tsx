import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { Project } from '@/types/models'
import { PageHeader, LoadingState, MetricCard, ErrorState } from '@/components/ui/PageHeader'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { IDView } from '@/components/ui/DigestView'

export function ProjectDetailPage() {
  const { id } = useParams()
  const [project, setProject] = useState<Project | null | undefined>(undefined)
  const [tasks, setTasks] = useState(0)
  const [findings, setFindings] = useState(0)
  useEffect(() => {
    if (!id) return
    api.getProject(id).then(setProject)
    api.listTasks(id).then((t) => setTasks(t.length))
    api.listFindings().then((f) => setFindings(f.filter((x) => x.projectId === id).length))
  }, [id])
  if (project === undefined) return <LoadingState />
  if (!project) return <ErrorState title="Project not found" reason="UNKNOWN_PROJECT" />
  return (
    <div className="space-y-5">
      <PageHeader
        title={project.name}
        description="Project overview metrics from MOCK fixtures. Not live authorization state."
        actions={
          <div className="flex gap-2">
            <Link to="/tasks" className="ops-btn ops-btn-secondary text-[12px]">
              Tasks
            </Link>
            <Link to="/scopes" className="ops-btn ops-btn-secondary text-[12px]">
              Scopes
            </Link>
          </div>
        }
      />
      <div className="flex flex-wrap items-center gap-3">
        <StatusBadge status={project.status} />
        <IDView value={project.id} label="Project" />
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        <MetricCard label="Tasks" value={tasks} />
        <MetricCard label="Findings" value={findings} />
        <MetricCard label="Environment" value="DEMO" hint="MOCK" />
      </div>
    </div>
  )
}
