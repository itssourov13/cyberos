import type {
  Assignment,
  Asset,
  AuditEvent,
  CaseItem,
  DashboardSummary,
  Finding,
  Pack,
  Project,
  Report,
  Scope,
  SpineRequest,
  SpineTrace,
  SystemInfo,
  Task,
  Worker,
} from './types'

/** Every failure surfaces as a typed, fail-closed error. */
export class ApiError extends Error {
  readonly code: 'unavailable' | 'unauthenticated' | 'forbidden' | 'invalid' | 'not-found' | 'unknown'
  constructor(code: ApiError['code'], message: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
  }
}

/**
 * The single seam between the UI and the platform.
 *
 * Pages and hooks depend on this interface only. Phase 24C ships a preview
 * adapter; Phase 24D adds an adapter over `/api/v1` implementing the same
 * surface. There are deliberately no methods that issue grants, change pack
 * trust, verify results or run tools: the browser can only *ask* the backend
 * (`submitSpineRequest`) and *display* what it answers.
 */
export interface CyberosApi {
  readonly mode: 'preview' | 'live'

  dashboard(): Promise<DashboardSummary>
  system(): Promise<SystemInfo>

  projects(): Promise<Project[]>
  scopes(projectId?: string): Promise<Scope[]>
  tasks(projectId?: string): Promise<Task[]>

  /** Ask the control plane to evaluate and (if allowed) run a request. */
  submitSpineRequest(req: SpineRequest): Promise<SpineTrace>

  workers(): Promise<Worker[]>
  assignments(): Promise<Assignment[]>

  packs(): Promise<Pack[]>

  assets(): Promise<Asset[]>
  findings(): Promise<Finding[]>
  cases(): Promise<CaseItem[]>
  reports(): Promise<Report[]>

  audit(): Promise<AuditEvent[]>
}
