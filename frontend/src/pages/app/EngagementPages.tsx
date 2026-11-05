import { useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Search } from 'lucide-react'
import { useApi } from '@/api'
import type { Project, Scope, Task } from '@/api/types'
import { Async, Badge, DataTable, Drawer, Empty, KV, PageHeader, Panel, PanelHead, Stat, Tabs, type Column } from '@/components/ui'
import { ProjectStatusBadge, ScopeStatusBadge, SeverityBadge, TaskStateBadge } from '@/components/domain/badges'
import { fmtDateTime, relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

type ScopeFilter = 'all' | Scope['status']

const scopeCols = (showProject: boolean): Column<Scope>[] => [
  { key: 'name', header: 'Scope', sort: (s) => s.name, render: (s) => <strong>{s.name}</strong> },
  ...(showProject ? [{ key: 'project', header: 'Project', render: (s: Scope) => <Link className="link mono" to={`/app/projects/${s.projectId}`}>{s.projectId}</Link> }] : []),
  { key: 'kind', header: 'Kind', render: (s) => <span className="chip">{s.kind}</span> },
  { key: 'targets', header: 'Targets', render: (s) => <span className="mono dim">{s.targets.length ? s.targets.join(', ') : '—'}</span> },
  { key: 'window', header: 'Window', render: (s) => <span className="muted nowrap">{s.window.from ? fmtDateTime(s.window.from) : 'open'} → {s.window.to ? fmtDateTime(s.window.to) : 'open'}</span> },
  { key: 'status', header: 'Status', sort: (s) => s.status, render: (s) => <ScopeStatusBadge status={s.status} /> },
]

const taskCols: Column<Task>[] = [
  { key: 'id', header: 'ID', render: (t) => <span className="cell-id">{t.id}</span> },
  { key: 'title', header: 'Task', sort: (t) => t.title, render: (t) => <strong>{t.title}</strong> },
  { key: 'cap', header: 'Capability', render: (t) => <span className="chip">{t.capability}</span> },
  { key: 'assignee', header: 'Assignee', render: (t) => <span className="dim">{t.assignee}</span> },
  { key: 'state', header: 'State', sort: (t) => t.state, render: (t) => <TaskStateBadge state={t.state} /> },
  { key: 'updated', header: 'Updated', sort: (t) => t.updatedAt, render: (t) => <span className="muted">{relTime(t.updatedAt)}</span> },
]

export function ProjectsPage() {
  const api = useApi()
  const nav = useNavigate()
  const state = useAsync(() => api.projects(), [api])
  return (
    <>
      <PageHeader eyebrow="Engagements" title="Projects" description="Each project groups the scopes, tasks and evidence of one engagement." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Project>
              caption="Projects"
              rows={rows}
              rowKey={(p) => p.id}
              onRowClick={(p) => nav(`/app/projects/${p.id}`)}
              columns={[
                { key: 'name', header: 'Project', sort: (p) => p.name, render: (p) => <div className="col" style={{ gap: 0 }}><strong>{p.name}</strong><span className="muted">{p.client}</span></div> },
                { key: 'status', header: 'Status', sort: (p) => p.status, render: (p) => <ProjectStatusBadge status={p.status} /> },
                { key: 'scopes', header: 'Scopes', sort: (p) => p.scopeCount, align: 'right', render: (p) => p.scopeCount },
                { key: 'tasks', header: 'Open tasks', sort: (p) => p.openTasks, align: 'right', render: (p) => p.openTasks },
                { key: 'updated', header: 'Updated', sort: (p) => p.updatedAt, render: (p) => <span className="muted">{relTime(p.updatedAt)}</span> },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}

export function ProjectDetailPage() {
  const { projectId = '' } = useParams()
  const api = useApi()
  const projects = useAsync(() => api.projects(), [api])
  const scopes = useAsync(() => api.scopes(projectId), [api, projectId])
  const tasks = useAsync(() => api.tasks(projectId), [api, projectId])
  const project = projects.data?.find((p) => p.id === projectId)

  return (
    <>
      <PageHeader
        eyebrow="Project"
        title={project?.name ?? projectId}
        description={project ? `Client: ${project.client}` : undefined}
        actions={<>{project && <ProjectStatusBadge status={project.status} />}<Link to="/app/projects" className="btn btn-sm">All projects</Link></>}
      />
      {projects.data && !project && <Empty title="Project not found">No project with id {projectId}.</Empty>}
      {project && (
        <div className="grid grid-3">
          <Stat label="Scopes" value={project.scopeCount} />
          <Stat label="Open tasks" value={project.openTasks} />
          <Stat label="Last activity" value={relTime(project.updatedAt)} />
        </div>
      )}
      <Panel>
        <PanelHead title="Scopes" />
        <Async state={scopes}>{(rows) => <DataTable<Scope> caption="Project scopes" rows={rows} rowKey={(s) => s.id} columns={scopeCols(false)} emptyTitle="No scopes" />}</Async>
      </Panel>
      <Panel>
        <PanelHead title="Tasks" />
        <Async state={tasks}>{(rows) => <DataTable<Task> caption="Project tasks" rows={rows} rowKey={(t) => t.id} columns={taskCols} emptyTitle="No tasks" />}</Async>
      </Panel>
    </>
  )
}

export function ScopesPage() {
  const api = useApi()
  const state = useAsync(() => api.scopes(), [api])
  const [filter, setFilter] = useState<ScopeFilter>('all')
  return (
    <>
      <PageHeader eyebrow="Engagements" title="Scopes" description="Authorised targets and the time window in which they may be touched. Scope is the first link of the spine." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <>
              <div className="toolbar">
                <Tabs<ScopeFilter>
                  value={filter}
                  onChange={(v) => setFilter(v)}
                  tabs={[
                    { value: 'all', label: 'All', count: rows.length },
                    { value: 'in-window', label: 'In window', count: rows.filter((r) => r.status === 'in-window').length },
                    { value: 'scheduled', label: 'Scheduled', count: rows.filter((r) => r.status === 'scheduled').length },
                    { value: 'expired', label: 'Expired', count: rows.filter((r) => r.status === 'expired').length },
                  ]}
                />
              </div>
              <DataTable<Scope> caption="Scopes" rows={filter === 'all' ? rows : rows.filter((r) => r.status === filter)} rowKey={(s) => s.id} columns={scopeCols(true)} />
            </>
          )}
        </Async>
      </Panel>
    </>
  )
}

export function TasksPage() {
  const api = useApi()
  const state = useAsync(() => api.tasks(), [api])
  const [q, setQ] = useState('')
  const [sel, setSel] = useState<Task | undefined>(undefined)
  const filter = (rows: Task[]) => rows.filter((t) => `${t.title} ${t.capability} ${t.id}`.toLowerCase().includes(q.toLowerCase()))
  return (
    <>
      <PageHeader eyebrow="Engagements" title="Tasks" description="Work requested against a scope. A task only runs after the control plane has decided and issued a grant." />
      <Panel>
        <div className="toolbar">
          <div className="search">
            <Search size={15} aria-hidden="true" />
            <input className="input" placeholder="Filter tasks…" aria-label="Filter tasks" value={q} onChange={(e) => setQ(e.target.value)} />
          </div>
        </div>
        <Async state={state}>{(rows) => <DataTable<Task> caption="Tasks" rows={filter(rows)} rowKey={(t) => t.id} columns={taskCols} onRowClick={(t) => setSel(t)} selectedKey={sel?.id} emptyTitle="No matching tasks" />}</Async>
      </Panel>
      <Drawer open={!!sel} title={sel?.title ?? ''} onClose={() => setSel(undefined)}>
        {sel && (
          <>
            <KV items={[['ID', <span key="ID" className="mono">{sel.id}</span>], ['Project', <Link key="Project" className="link" to={`/app/projects/${sel.projectId}`}>{sel.projectId}</Link>], ['Scope', <span key="Scope" className="mono">{sel.scopeId}</span>], ['Capability', <span key="Capability" className="chip">{sel.capability}</span>], ['Assignee', sel.assignee], ['State', <TaskStateBadge key="State" state={sel.state} />], ['Updated', fmtDateTime(sel.updatedAt)]]} />
            <Link to="/app/evaluate" className="btn btn-primary">Evaluate a similar request</Link>
          </>
        )}
      </Drawer>
    </>
  )
}

/** Assessments are derived from findings until the backend exposes them directly. */
export function AssessmentsPage() {
  const api = useApi()
  const findings = useAsync(() => api.findings(), [api])
  const groups = useMemo(() => {
    const m = new Map<string, NonNullable<typeof findings.data>>()
    for (const f of findings.data ?? []) m.set(f.assessmentId, [...(m.get(f.assessmentId) ?? []), f])
    return [...m.entries()].sort((a, b) => b[0].localeCompare(a[0]))
  }, [findings.data])
  return (
    <>
      <PageHeader eyebrow="Engagements" title="Assessments" description="Findings grouped by the assessment that produced them." />
      <Async state={findings}>
        {() =>
          groups.length === 0 ? (
            <Empty title="No assessments yet" />
          ) : (
            <div className="grid grid-2">
              {groups.map(([id, items]) => (
                <Panel key={id}>
                  <PanelHead title={id} sub={`${items.length} finding${items.length === 1 ? '' : 's'}`} />
                  <div className="ledger">
                    {items.map((f) => (
                      <div key={f.id} className="ledger-row">
                        <SeverityBadge severity={f.severity} />
                        <span className="truncate">{f.title}</span>
                        <Badge>{f.status}</Badge>
                      </div>
                    ))}
                  </div>
                </Panel>
              ))}
            </div>
          )
        }
      </Async>
      <p className="muted">Preview: a dedicated assessments endpoint is a deferred backend requirement.</p>
    </>
  )
}
