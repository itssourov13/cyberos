/**
 * CYBEROS frontend domain contracts.
 *
 * These are *view models*. The Go backend remains authoritative for every
 * decision, grant, trust state and execution result. Nothing in this file (or
 * anywhere in the browser) mints, signs, verifies or authorises anything.
 */

export type Tone = 'ok' | 'warn' | 'danger' | 'info' | 'trust' | 'evidence' | 'grant' | 'neutral'

export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'info'

/* ---------- Engagements ---------- */
export interface Project {
  id: string
  name: string
  client: string
  status: 'active' | 'paused' | 'closed'
  scopeCount: number
  openTasks: number
  updatedAt: string
}

export interface Scope {
  id: string
  projectId: string
  name: string
  kind: 'domain' | 'cidr' | 'repo' | 'cloud-account' | 'host'
  targets: string[]
  window: { from: string; to: string }
  status: 'in-window' | 'scheduled' | 'expired'
}

export interface Task {
  id: string
  projectId: string
  scopeId: string
  title: string
  capability: string
  state: 'queued' | 'evaluating' | 'granted' | 'running' | 'done' | 'denied' | 'failed'
  assignee: string
  updatedAt: string
}

/* ---------- Execution spine ---------- */
export type SpineStageId = 'scope' | 'decision' | 'grant' | 'broker' | 'execution' | 'evidence'

export type StageOutcome = 'pass' | 'deny' | 'skipped' | 'pending'

export interface SpineStage {
  id: SpineStageId
  label: string
  outcome: StageOutcome
  detail: string
  /** Opaque reference minted by the backend. The browser only displays it. */
  ref?: string
}

export interface SpineRequest {
  principal: string
  projectId: string
  scopeId: string
  capability: string
  target: string
}

export interface SpineTrace {
  requestId: string
  request: SpineRequest
  stages: SpineStage[]
  final: 'completed' | 'denied'
  /** Always true for the preview adapter. Real adapters set this false. */
  simulated: boolean
}

/* ---------- Fleet ---------- */
export interface Worker {
  id: string
  projectId: string
  name: string
  state: 'online' | 'suspended' | 'revoked' | 'stale'
  identityFingerprint: string
  lastHeartbeat: string
  load: number
  capabilities: string[]
}

export interface Assignment {
  id: string
  workerId: string
  taskId: string
  state: 'leased' | 'completed' | 'revoked' | 'stale'
  leaseExpires: string
  resultVerified: boolean
}

/* ---------- Supply chain ---------- */
export interface Pack {
  id: string
  name: string
  version: string
  publisher: string
  /** Backend-recorded content digest. In preview mode this is a `sample:` placeholder. */
  digest: string
  signature: 'valid' | 'invalid' | 'missing'
  trust: 'trusted' | 'untrusted' | 'revoked'
  capabilities: string[]
}

/* ---------- Evidence ---------- */
export interface Asset {
  id: string
  name: string
  kind: string
  exposure: 'internet' | 'internal' | 'restricted'
  riskScore: number
}

export interface Finding {
  id: string
  title: string
  severity: Severity
  assetId: string
  assessmentId: string
  status: 'open' | 'triaged' | 'accepted' | 'fixed'
  foundAt: string
  /** Backend-recorded digest. In preview mode this is a `sample:` placeholder, never a real digest. */
  evidenceDigest: string
}

export interface CaseItem {
  id: string
  title: string
  status: 'open' | 'in-progress' | 'resolved'
  severity: Severity
  owner: string
  findingIds: string[]
}

export interface Report {
  id: string
  title: string
  projectId: string
  generatedAt: string
  formats: string[]
}

/* ---------- Audit / system ---------- */
export interface AuditEvent {
  id: string
  at: string
  actor: string
  action: string
  subject: string
  tone: Tone
  digest: string
}

export interface DashboardSummary {
  projects: number
  activeScopes: number
  openFindings: number
  grantsIssued24h: number
  denials24h: number
  workersOnline: number
  workersTotal: number
  executionsPerHour: number[]
  severityMix: Record<Severity, number>
}

export interface SystemInfo {
  version: string
  commit: string
  status: 'ok' | 'degraded'
  checks: { name: string; ok: boolean; note: string }[]
  mode: 'preview' | 'live'
}
