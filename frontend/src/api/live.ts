import { ApiError, type CyberosApi } from './client'
import { request } from './http'
import type {
  Asset,
  CaseItem,
  DashboardSummary,
  Finding,
  Project,
  Report,
  Scope,
  Severity,
  SpineRequest,
  SpineStage,
  SpineTrace,
  SystemInfo,
  Task,
  Worker,
} from './types'

/**
 * Live adapter over the real `/api/v1` backend.
 *
 * Rules:
 * - Only endpoints that exist in the Go API are called.
 * - Backend structs without JSON tags arrive PascalCase; mapping happens here
 *   and nowhere else.
 * - Where the backend has no data for a view-model field the adapter leaves it
 *   empty/`null`. It never invents values and never falls back to fixtures.
 * - Surfaces the backend does not expose yet throw `ApiError('unsupported')`
 *   so the UI shows an honest error state.
 */

type Raw = Record<string, unknown>
const str = (v: unknown): string => (typeof v === 'string' ? v : v == null ? '' : String(v))
const arr = (v: unknown): Raw[] => (Array.isArray(v) ? (v as Raw[]) : [])
const iso = (v: unknown): string => {
  const s = str(v)
  return s && !s.startsWith('0001-') ? s : ''
}
const pick = (o: Raw, ...keys: string[]): unknown => {
  for (const k of keys) if (o[k] !== undefined) return o[k]
  return undefined
}

const unsupported = (what: string): never => {
  throw new ApiError('unsupported', `${what} is not exposed by the backend yet.`)
}

/* ---------- mappers ---------- */
const mapProject = (p: Raw): Project => ({
  id: str(p.ID),
  name: str(p.Name),
  client: str(p.Description),
  status: str(p.Status).toLowerCase() === 'closed' || str(p.Status).toLowerCase() === 'archived' ? 'closed' : str(p.Status).toLowerCase() === 'paused' ? 'paused' : 'active',
  scopeCount: 0,
  openTasks: 0,
  updatedAt: iso(p.UpdatedAt),
})

function mapScope(s: Raw): Scope {
  const c = (s.Constraints ?? {}) as Raw
  const from = iso(s.StartsAt)
  const to = iso(s.ExpiresAt)
  const st = str(s.Status).toUpperCase()
  const now = Date.now()
  const status: Scope['status'] =
    st === 'EXPIRED' || st === 'REVOKED' || st === 'INACTIVE' || (to && new Date(to).getTime() < now)
      ? 'expired'
      : from && new Date(from).getTime() > now
        ? 'scheduled'
        : 'in-window'
  return {
    id: str(s.ID),
    projectId: str(s.ProjectID),
    name: str(s.Name),
    kind: 'scope',
    targets: arr(c.allowed_asset_ids).length ? (c.allowed_asset_ids as unknown[]).map(str) : [],
    window: { from, to },
    status,
  }
}

const TASK_STATE: Record<string, Task['state']> = {
  CREATED: 'queued',
  SCHEDULED: 'queued',
  PENDING: 'queued',
  WAITING_APPROVAL: 'evaluating',
  AUTHORIZED: 'granted',
  READY: 'granted',
  RUNNING: 'running',
  SUCCEEDED: 'done',
  FAILED: 'failed',
  CANCELLED: 'denied',
  EXPIRED: 'denied',
}
const mapTask = (t: Raw): Task => ({
  id: str(t.ID),
  projectId: str(t.ProjectID),
  scopeId: str(t.ScopeID),
  title: str(t.Title) || str(t.ID),
  capability: str(t.CapabilityID),
  state: TASK_STATE[str(t.State).toUpperCase()] ?? 'queued',
  assignee: str(t.PrincipalID),
  updatedAt: iso(t.UpdatedAt),
})

const severity = (v: unknown): Severity => {
  const s = str(v).toLowerCase()
  return s === 'critical' || s === 'high' || s === 'medium' || s === 'low' ? s : 'info'
}
const mapFinding = (f: Raw): Finding => ({
  id: str(f.ID),
  title: str(f.Title),
  severity: severity(f.Severity),
  assetId: str(f.Target),
  assessmentId: str(f.AssessmentID),
  status: str(f.Status).toUpperCase() === 'RESOLVED' ? 'fixed' : 'open',
  foundAt: iso(f.FirstSeen) || iso(f.ProducedAt),
  evidenceDigest: arr(f.EvidenceIDs).length ? str((f.EvidenceIDs as unknown[])[0]) : '',
})

const mapAsset = (a: Raw): Asset => ({
  id: str(a.ID),
  name: str(a.Name) || str(a.Value),
  kind: str(a.Type).toLowerCase(),
  exposure: 'unknown',
  riskScore: null,
})

const CASE_STATUS: Record<string, CaseItem['status']> = {
  OPEN: 'open',
  REOPENED: 'open',
  ACKNOWLEDGED: 'in-progress',
  IN_PROGRESS: 'in-progress',
  RESOLVED: 'resolved',
  ACCEPTED: 'resolved',
}
const mapCase = (c: Raw): CaseItem => ({
  id: str(c.ID),
  title: str(c.Title),
  status: CASE_STATUS[str(c.Status).toUpperCase()] ?? 'open',
  severity: severity(c.Priority),
  owner: str(c.CreatedBy),
  findingIds: arr(c.FindingIDs).length ? (c.FindingIDs as unknown[]).map(str) : [],
})

const mapReport = (r: Raw): Report => ({
  id: str(r.ID),
  title: `Report ${str(r.AssessmentID) || str(r.ID)}`,
  projectId: str(r.ProjectID),
  generatedAt: iso(r.GeneratedAt),
  formats: ['json', 'markdown'],
})

function mapWorker(w: Raw): Worker {
  const st = str(w.status).toLowerCase()
  return {
    id: str(w.worker_id),
    projectId: str(w.project_id),
    name: str(w.host_id) || str(w.worker_id),
    state: st === 'suspended' ? 'suspended' : st === 'revoked' ? 'revoked' : st === 'active' ? 'online' : 'stale',
    identityFingerprint: str(w.cert_fingerprint),
    lastHeartbeat: iso(w.last_seen_at),
    load: 0,
    capabilities: [],
  }
}

/* ---------- helpers ---------- */
/** Tenant callers must name a project on most reads, so fan out over the caller's projects. */
async function projectsRaw(signal?: AbortSignal): Promise<Raw[]> {
  const r = await request<{ projects?: Raw[] }>('/projects', { signal })
  return arr(r.projects)
}
async function perProject<T>(path: string, key: string, map: (r: Raw) => T, projectId?: string, extra?: Record<string, string>): Promise<T[]> {
  const ids = projectId ? [projectId] : (await projectsRaw()).map((p) => str(p.ID))
  const pages = await Promise.all(ids.map((id) => request<Raw>(path, { query: { project_id: id, ...extra } })))
  return pages.flatMap((p) => arr(p[key]).map(map))
}

function traceFrom(req: SpineRequest, stages: SpineStage[], final: SpineTrace['final'], id: string): SpineTrace {
  return { requestId: id, request: req, stages, final, simulated: false }
}

export const liveApi: CyberosApi = {
  mode: 'live',

  async dashboard(): Promise<DashboardSummary> {
    const [counts, scopes, findings] = await Promise.all([
      request<Raw>('/dashboard'),
      this.scopes(),
      this.findings(),
    ])
    const open = findings.filter((f) => f.status !== 'fixed')
    const mix: Record<Severity, number> = { critical: 0, high: 0, medium: 0, low: 0, info: 0 }
    open.forEach((f) => (mix[f.severity] += 1))
    return {
      projects: Number(counts.projects) || 0,
      activeScopes: scopes.filter((s) => s.status === 'in-window').length,
      openFindings: open.length,
      // The backend exposes no grant/denial/worker/throughput aggregates yet.
      grantsIssued24h: null,
      denials24h: null,
      workersOnline: null,
      workersTotal: null,
      executionsPerHour: null,
      severityMix: mix,
    }
  },

  async system(): Promise<SystemInfo> {
    const [health, v] = await Promise.all([request<Raw>('/health'), request<Raw>('/version')])
    const ok = str(health.status) === 'ok'
    return {
      version: str(v.version ?? v.Version) || 'unknown',
      commit: str(v.commit ?? v.Commit) || 'unknown',
      status: ok ? 'ok' : 'degraded',
      checks: [{ name: 'API health', ok, note: str(health.status) }],
      mode: 'live',
    }
  },

  async projects(): Promise<Project[]> {
    const base = (await projectsRaw()).map(mapProject)
    return Promise.all(
      base.map(async (p) => {
        const [sc, tk] = await Promise.all([
          request<Raw>('/scopes', { query: { project_id: p.id } }),
          request<Raw>('/tasks', { query: { project_id: p.id } }),
        ])
        const open = arr(tk.tasks).filter((t) => !['SUCCEEDED', 'FAILED', 'CANCELLED', 'EXPIRED'].includes(str(t.State).toUpperCase())).length
        return { ...p, scopeCount: arr(sc.scopes).length, openTasks: open }
      }),
    )
  },

  scopes: (projectId) => perProject('/scopes', 'scopes', mapScope, projectId),
  tasks: (projectId) => perProject('/tasks', 'tasks', mapTask, projectId),

  async capabilities(): Promise<string[]> {
    const r = await request<{ capabilities?: Raw[] }>('/capabilities')
    return arr(r.capabilities).map((c) => str(pick(c, 'ID', 'id'))).filter(Boolean).sort()
  },

  async submitSpineRequest(req: SpineRequest): Promise<SpineTrace> {
    // The authenticated principal is bound server-side; none is sent.
    const body = { project_id: req.projectId, scope_id: req.scopeId, task_id: req.taskId ?? '', capability_id: req.capability }
    const rid = `req-${Date.now().toString(36)}`
    const stages: SpineStage[] = [{ id: 'scope', label: 'Scope', outcome: 'pass', detail: `Scope ${req.scopeId} evaluated by the control plane.` }]
    let grant: Raw | undefined
    try {
      const r = await request<{ grant?: Raw; decision?: Raw }>('/grants', { method: 'POST', body })
      grant = r.grant
      stages.push({ id: 'decision', label: 'Decision', outcome: 'pass', detail: str(r.decision?.reason) || 'Allowed.', ref: str(r.decision?.id) })
    } catch (e) {
      if (e instanceof ApiError && (e.code === 'forbidden' || e.code === 'invalid' || e.code === 'conflict')) {
        stages.push({ id: 'decision', label: 'Decision', outcome: 'deny', detail: e.message })
        for (const id of ['grant', 'broker', 'execution', 'evidence'] as const) stages.push({ id, label: id, outcome: 'skipped', detail: 'Not reached.' })
        return traceFrom(req, stages, 'denied', rid)
      }
      throw e
    }
    stages.push({ id: 'grant', label: 'Grant', outcome: 'pass', detail: 'Execution grant issued by the control plane.', ref: str(grant?.id) })
    try {
      const res = await request<Raw>('/execute', { method: 'POST', body: { grant, parameters: req.target ? { target: req.target } : {} }, timeoutMs: 60_000 })
      const failed = /fail|denied|error/i.test(str(res.status))
      stages.push({ id: 'broker', label: 'Broker', outcome: 'pass', detail: 'Grant verified by the broker.' })
      stages.push({ id: 'execution', label: 'Execution', outcome: failed ? 'deny' : 'pass', detail: str(res.error_message) || str(res.status) })
      stages.push({ id: 'evidence', label: 'Evidence', outcome: failed ? 'skipped' : 'pass', detail: failed ? 'No evidence recorded.' : 'Result returned by the backend.' })
      return traceFrom(req, stages, failed ? 'denied' : 'completed', rid)
    } catch (e) {
      if (e instanceof ApiError && (e.code === 'forbidden' || e.code === 'invalid' || e.code === 'conflict')) {
        stages.push({ id: 'broker', label: 'Broker', outcome: 'deny', detail: e.message })
        for (const id of ['execution', 'evidence'] as const) stages.push({ id, label: id, outcome: 'skipped', detail: 'Not reached.' })
        return traceFrom(req, stages, 'denied', rid)
      }
      throw e
    }
  },

  async workers(): Promise<Worker[]> {
    // Registration/suspend/revoke exist per project, but the list route is a
    // stub in the backend, so no truthful fleet view can be built yet.
    return unsupported('Worker listing')
  },
  assignments: async () => unsupported('Assignments'),
  packs: async () => unsupported('Pack and trust inventory'),
  audit: async () => unsupported('The audit feed'),

  assets: () => perProject('/assets', 'assets', mapAsset),
  findings: () => perProject('/findings', 'findings', mapFinding),
  cases: () => perProject('/cases', 'cases', mapCase),
  reports: () => perProject('/reports', 'reports', mapReport),

  // `mapWorker` is exported for the worker admin surfaces once listing exists.
}

export { mapWorker }
