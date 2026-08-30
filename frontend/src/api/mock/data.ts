/** MOCK_ / DEMO_ fixtures — non-authoritative development data only. */
import type {
  Assignment,
  AuditEvent,
  DashboardSnapshot,
  Decision,
  Grant,
  Project,
  Worker,
} from '@/types/models'

export const MOCK_PROJECTS: Project[] = [
  { id: 'proj-demo-1', name: 'DEMO Red Team', status: 'ACTIVE' },
  { id: 'proj-demo-2', name: 'DEMO Internal', status: 'ACTIVE' },
  { id: 'proj-demo-3', name: 'DEMO Lab', status: 'DISABLED' },
]

export interface MockScope {
  id: string
  projectId: string
  name: string
  target: string
  type: string
  status: 'IN_SCOPE' | 'OUT_OF_SCOPE' | 'EXPIRED' | 'UNKNOWN'
  taskCount: number
  createdAt: string
}

export const MOCK_SCOPES: MockScope[] = [
  {
    id: 'scope-demo-1',
    projectId: 'proj-demo-1',
    name: 'External perimeter',
    target: '203.0.113.0/24',
    type: 'CIDR',
    status: 'IN_SCOPE',
    taskCount: 12,
    createdAt: '2026-08-01T10:00:00Z',
  },
  {
    id: 'scope-demo-2',
    projectId: 'proj-demo-1',
    name: 'Staging apps',
    target: '*.staging.example',
    type: 'DOMAIN',
    status: 'IN_SCOPE',
    taskCount: 5,
    createdAt: '2026-08-10T12:00:00Z',
  },
  {
    id: 'scope-demo-3',
    projectId: 'proj-demo-2',
    name: 'Expired window',
    target: '10.0.0.0/8',
    type: 'CIDR',
    status: 'EXPIRED',
    taskCount: 0,
    createdAt: '2026-01-01T00:00:00Z',
  },
]

export interface MockTask {
  id: string
  projectId: string
  scopeId: string
  name: string
  status: string
  priority: string
  capabilityId: string
  updatedAt: string
  createdAt: string
}

export const MOCK_TASKS: MockTask[] = [
  {
    id: 'task-demo-1',
    projectId: 'proj-demo-1',
    scopeId: 'scope-demo-1',
    name: 'Perimeter recon',
    status: 'RUNNING',
    priority: 'HIGH',
    capabilityId: 'cap.scan.network',
    createdAt: '2026-08-20T09:00:00Z',
    updatedAt: '2026-08-29T18:00:00Z',
  },
  {
    id: 'task-demo-2',
    projectId: 'proj-demo-1',
    scopeId: 'scope-demo-2',
    name: 'Staging TLS review',
    status: 'COMPLETED',
    priority: 'MEDIUM',
    capabilityId: 'cap.scan.http',
    createdAt: '2026-08-15T11:00:00Z',
    updatedAt: '2026-08-16T14:00:00Z',
  },
  {
    id: 'task-demo-3',
    projectId: 'proj-demo-2',
    scopeId: 'scope-demo-3',
    name: 'Denied attempt',
    status: 'DENIED',
    priority: 'LOW',
    capabilityId: 'cap.scan.network',
    createdAt: '2026-08-28T08:00:00Z',
    updatedAt: '2026-08-28T08:05:00Z',
  },
]

export const MOCK_DECISIONS: Decision[] = [
  {
    id: 'dec-demo-1',
    outcome: 'ALLOW',
    reason: 'MOCK policy allow — not live CP',
    projectId: 'proj-demo-1',
    scopeId: 'scope-demo-1',
    capabilityId: 'cap.scan.network',
    correlationId: 'corr-demo-1',
    timestamp: '2026-08-29T17:00:00Z',
  },
  {
    id: 'dec-demo-2',
    outcome: 'DENY',
    reason: 'MOCK scope expired',
    projectId: 'proj-demo-2',
    scopeId: 'scope-demo-3',
    capabilityId: 'cap.scan.network',
    correlationId: 'corr-demo-2',
    timestamp: '2026-08-28T08:05:00Z',
  },
]

export const MOCK_GRANTS: Grant[] = [
  {
    id: 'grt-demo-1',
    projectId: 'proj-demo-1',
    scopeId: 'scope-demo-1',
    taskId: 'task-demo-1',
    capabilityId: 'cap.scan.network',
    packContentDigest: 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
    status: 'ISSUED',
    expiresAt: '2026-08-29T19:00:00Z',
  },
]

export interface MockFinding {
  id: string
  title: string
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO'
  status: string
  assetId: string
  projectId: string
  discoveredAt: string
}

export const MOCK_FINDINGS: MockFinding[] = [
  {
    id: 'fnd-demo-1',
    title: 'MOCK Exposed management interface',
    severity: 'HIGH',
    status: 'OPEN',
    assetId: 'ast-demo-1',
    projectId: 'proj-demo-1',
    discoveredAt: '2026-08-21T10:00:00Z',
  },
  {
    id: 'fnd-demo-2',
    title: 'MOCK Weak TLS configuration',
    severity: 'MEDIUM',
    status: 'TRIAGED',
    assetId: 'ast-demo-2',
    projectId: 'proj-demo-1',
    discoveredAt: '2026-08-16T15:00:00Z',
  },
]

export interface MockAsset {
  id: string
  hostname: string
  address: string
  type: string
  projectId: string
  status: string
  lastSeen: string
}

export const MOCK_ASSETS: MockAsset[] = [
  {
    id: 'ast-demo-1',
    hostname: 'edge-01.demo',
    address: '203.0.113.10',
    type: 'HOST',
    projectId: 'proj-demo-1',
    status: 'ACTIVE',
    lastSeen: '2026-08-29T18:00:00Z',
  },
  {
    id: 'ast-demo-2',
    hostname: 'app.staging.example',
    address: '203.0.113.20',
    type: 'APPLICATION',
    projectId: 'proj-demo-1',
    status: 'ACTIVE',
    lastSeen: '2026-08-28T12:00:00Z',
  },
]

export interface MockRisk {
  id: string
  title: string
  severity: string
  likelihood: string
  impact: string
  status: string
  projectId: string
}

export const MOCK_RISKS: MockRisk[] = [
  {
    id: 'rsk-demo-1',
    title: 'MOCK Perimeter exposure',
    severity: 'HIGH',
    likelihood: 'MEDIUM',
    impact: 'HIGH',
    status: 'OPEN',
    projectId: 'proj-demo-1',
  },
]

export interface MockCase {
  id: string
  title: string
  status: string
  priority: string
  owner: string
  projectId: string
  updatedAt: string
}

export const MOCK_CASES: MockCase[] = [
  {
    id: 'case-demo-1',
    title: 'MOCK Perimeter investigation',
    status: 'OPEN',
    priority: 'HIGH',
    owner: 'analyst-demo',
    projectId: 'proj-demo-1',
    updatedAt: '2026-08-29T16:00:00Z',
  },
]

export interface MockReport {
  id: string
  title: string
  type: string
  projectId: string
  generatedAt: string
  status: string
}

export const MOCK_REPORTS: MockReport[] = [
  {
    id: 'rpt-demo-1',
    title: 'MOCK Assessment summary',
    type: 'ASSESSMENT',
    projectId: 'proj-demo-1',
    generatedAt: '2026-08-25T09:00:00Z',
    status: 'READY',
  },
]

export interface MockAssessment {
  id: string
  projectId: string
  target: string
  status: string
  progress: number
  findings: number
  updatedAt: string
}

export const MOCK_ASSESSMENTS: MockAssessment[] = [
  {
    id: 'asm-demo-1',
    projectId: 'proj-demo-1',
    target: '203.0.113.0/24',
    status: 'RUNNING',
    progress: 62,
    findings: 2,
    updatedAt: '2026-08-29T17:30:00Z',
  },
]

export const MOCK_WORKERS: Worker[] = [
  {
    id: 'worker-demo-01',
    projectId: 'proj-demo-1',
    hostId: 'host-edge-1',
    status: 'ACTIVE',
    certFingerprint: 'sha256:aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899',
    lastSeenAt: new Date().toISOString(),
  },
  {
    id: 'worker-demo-02',
    projectId: 'proj-demo-1',
    hostId: 'host-edge-2',
    status: 'DISABLED',
    certFingerprint: 'sha256:11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff',
  },
]

export const MOCK_ASSIGNMENTS: Assignment[] = [
  {
    id: 'asn-demo-1',
    grantId: 'grt-demo-1',
    projectId: 'proj-demo-1',
    workerId: 'worker-demo-01',
    hostId: 'host-edge-1',
    packContentDigest: 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
    status: 'RUNNING',
    leaseUntil: new Date(Date.now() + 600_000).toISOString(),
  },
]

export const MOCK_DASHBOARD: DashboardSnapshot = {
  environment: 'DEVELOPMENT',
  health: [
    { name: 'Control Plane', state: 'HEALTHY', detail: 'MOCK — not live' },
    { name: 'Broker', state: 'HEALTHY', detail: 'MOCK — not live' },
    { name: 'API', state: 'HEALTHY', detail: 'MOCK — not live' },
    { name: 'Database', state: 'UNKNOWN', detail: 'MOCK — DSN not verified' },
    { name: 'Remote Workers', state: 'DEGRADED', detail: 'MOCK — sample fleet' },
  ],
  activeWorkers: 1,
  pendingAssignments: 0,
  runningAssignments: 1,
  trustedPacks: 2,
  deniedDecisions: 1,
  recentFindings: 2,
  recentActivity: [
    {
      id: 'evt-1',
      type: 'DECISION',
      projectId: 'proj-demo-1',
      message: 'MOCK Decision ALLOW capability cap.scan.network',
      timestamp: '2026-08-29T17:00:00Z',
      correlationId: 'corr-demo-1',
    },
    {
      id: 'evt-2',
      type: 'GRANT',
      projectId: 'proj-demo-1',
      message: 'MOCK Grant issued (display only)',
      timestamp: '2026-08-29T17:01:00Z',
      correlationId: 'corr-demo-1',
    },
  ] as AuditEvent[],
}

export interface MockPack {
  id: string
  name: string
  version: string
  contentDigest: string
  signature: 'VALID' | 'INVALID' | 'UNKNOWN'
  signerKeyId: string
  trustState: 'TRUSTED' | 'UNTRUSTED' | 'VERIFIED' | 'REVOKED' | 'UNKNOWN'
  projectId: string
  updatedAt: string
}

export const MOCK_PACKS: MockPack[] = [
  {
    id: 'pack-demo-1',
    name: 'DEMO Network Recon',
    version: '1.4.2',
    contentDigest: 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
    signature: 'VALID',
    signerKeyId: 'key-demo-1',
    trustState: 'TRUSTED',
    projectId: 'proj-demo-1',
    updatedAt: '2026-08-20T10:00:00Z',
  },
  {
    id: 'pack-demo-2',
    name: 'DEMO HTTP Probe',
    version: '0.9.1',
    contentDigest: 'sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd',
    signature: 'VALID',
    signerKeyId: 'key-demo-1',
    trustState: 'UNTRUSTED',
    projectId: 'proj-demo-1',
    updatedAt: '2026-08-22T11:00:00Z',
  },
  {
    id: 'pack-demo-3',
    name: 'DEMO Revoked Tool',
    version: '0.1.0',
    contentDigest: 'sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
    signature: 'VALID',
    signerKeyId: 'key-demo-2',
    trustState: 'REVOKED',
    projectId: 'proj-demo-2',
    updatedAt: '2026-08-01T08:00:00Z',
  },
]

export interface MockSigner {
  keyId: string
  algorithm: string
  status: 'ACTIVE' | 'DISABLED' | 'REVOKED'
  publicFingerprint: string
}

export const MOCK_SIGNERS: MockSigner[] = [
  {
    keyId: 'key-demo-1',
    algorithm: 'Ed25519',
    status: 'ACTIVE',
    publicFingerprint: 'sha256:1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff',
  },
  {
    keyId: 'key-demo-2',
    algorithm: 'Ed25519',
    status: 'REVOKED',
    publicFingerprint: 'sha256:ffffeeeeddddccccbbbbaaaa9999888877776666555544443333222211110000',
  },
]

export interface MockAuditRow {
  id: string
  timestamp: string
  type: string
  severity: string
  projectId: string
  correlationId: string
  grantId?: string
  assignmentId?: string
  workerId?: string
  actor: string
  message: string
  mock: boolean
}

export const MOCK_AUDIT: MockAuditRow[] = [
  {
    id: 'aud-1',
    timestamp: '2026-08-29T17:00:00Z',
    type: 'DECISION_ALLOWED',
    severity: 'INFO',
    projectId: 'proj-demo-1',
    correlationId: 'corr-demo-1',
    grantId: 'grt-demo-1',
    actor: 'principal-demo',
    message: 'MOCK Decision ALLOW',
    mock: true,
  },
  {
    id: 'aud-2',
    timestamp: '2026-08-29T17:01:00Z',
    type: 'GRANT_ISSUED',
    severity: 'INFO',
    projectId: 'proj-demo-1',
    correlationId: 'corr-demo-1',
    grantId: 'grt-demo-1',
    actor: 'system',
    message: 'MOCK Grant issued',
    mock: true,
  },
  {
    id: 'aud-3',
    timestamp: '2026-08-29T17:02:00Z',
    type: 'ASSIGNMENT_CLAIMED',
    severity: 'INFO',
    projectId: 'proj-demo-1',
    correlationId: 'corr-demo-1',
    grantId: 'grt-demo-1',
    assignmentId: 'asn-demo-1',
    workerId: 'worker-demo-01',
    actor: 'worker-demo-01',
    message: 'MOCK Assignment claimed',
    mock: true,
  },
  {
    id: 'aud-4',
    timestamp: '2026-08-29T17:05:00Z',
    type: 'RESULT_VERIFIED',
    severity: 'SUCCESS',
    projectId: 'proj-demo-1',
    correlationId: 'corr-demo-1',
    assignmentId: 'asn-demo-1',
    workerId: 'worker-demo-01',
    actor: 'system',
    message: 'MOCK Result verified (display only)',
    mock: true,
  },
  {
    id: 'aud-5',
    timestamp: '2026-08-28T08:05:00Z',
    type: 'DECISION_DENIED',
    severity: 'WARN',
    projectId: 'proj-demo-2',
    correlationId: 'corr-demo-2',
    actor: 'principal-demo',
    message: 'MOCK Decision DENY — scope expired',
    mock: true,
  },
]
