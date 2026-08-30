/** CYBEROS UI models — display-oriented; server remains authoritative. */

export type HealthState = 'HEALTHY' | 'DEGRADED' | 'UNKNOWN' | 'UNAVAILABLE'
export type DecisionOutcome = 'ALLOW' | 'DENY' | 'UNKNOWN'
export type TrustState = 'TRUSTED' | 'UNTRUSTED' | 'VERIFIED' | 'REVOKED' | 'UNKNOWN'
export type WorkerStatus = 'ACTIVE' | 'DISABLED' | 'REVOKED' | 'PENDING' | 'UNKNOWN'
export type AssignmentStatus =
  | 'PENDING'
  | 'DISPATCHED'
  | 'CLAIMED'
  | 'RUNNING'
  | 'COMPLETED'
  | 'FAILED'
  | 'EXPIRED'
  | 'CANCELLED'

export interface Project {
  id: string
  name: string
  status: string
}

export interface Scope {
  id: string
  projectId: string
  name: string
  status: string
}

export interface Task {
  id: string
  projectId: string
  scopeId: string
  name: string
  status: string
}

export interface Decision {
  id: string
  outcome: DecisionOutcome
  reason: string
  projectId: string
  scopeId: string
  capabilityId: string
  correlationId?: string
  timestamp: string
}

export interface Grant {
  id: string
  projectId: string
  scopeId: string
  taskId?: string
  capabilityId: string
  packContentDigest?: string
  status: string
  expiresAt: string
}

export interface Worker {
  id: string
  projectId: string
  hostId: string
  status: WorkerStatus
  certFingerprint?: string
  lastSeenAt?: string
}

export interface Assignment {
  id: string
  grantId: string
  projectId: string
  workerId: string
  hostId: string
  packContentDigest?: string
  status: AssignmentStatus
  leaseUntil?: string
  resultHash?: string
}

export interface AuditEvent {
  id: string
  type: string
  projectId?: string
  message: string
  timestamp: string
  correlationId?: string
}

export interface SystemComponentHealth {
  name: string
  state: HealthState
  detail?: string
}

export interface DashboardSnapshot {
  environment: 'DEVELOPMENT' | 'PRODUCTION'
  health: SystemComponentHealth[]
  activeWorkers: number
  pendingAssignments: number
  runningAssignments: number
  trustedPacks: number
  deniedDecisions: number
  recentFindings: number
  recentActivity: AuditEvent[]
}
