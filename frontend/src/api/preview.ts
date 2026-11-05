import type { CyberosApi } from './client'
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
  SpineStage,
  SpineTrace,
  SystemInfo,
  Task,
  Worker,
} from './types'

/**
 * PREVIEW ADAPTER — fixtures only.
 * Lives in one file so it can be deleted when the live adapter lands (Phase 24D).
 * Nothing here is a security decision; the simulated trace exists to show the UI
 * what an authoritative backend answer will look like.
 */

const wait = <T,>(value: T, ms = 220): Promise<T> => new Promise((r) => setTimeout(() => r(value), ms))
const NOW = Date.now()
const iso = (minsAgo: number) => new Date(NOW - minsAgo * 60_000).toISOString()
/**
 * Deterministic SAMPLE identifier. This is NOT a cryptographic digest: it is a
 * non-cryptographic hash rendered with an explicit `sample:` prefix (same 7-char
 * width the slices below rely on) so it can never be mistaken for real evidence.
 * Real digests come from the backend.
 */
export const SAMPLE_ID_PREFIX = 'sample:'
const sampleId = (seed: string) => {
  let h = 2166136261
  for (let i = 0; i < seed.length; i++) h = Math.imul(h ^ seed.charCodeAt(i), 16777619)
  let out = ''
  for (let i = 0; i < 8; i++) {
    h = Math.imul(h ^ (h >>> 13), 1274126177)
    out += (h >>> 0).toString(16).padStart(8, '0')
  }
  return SAMPLE_ID_PREFIX + out
}

const projects: Project[] = [
  { id: 'prj-atlas', name: 'Atlas External Perimeter', client: 'Northwind Logistics', status: 'active', scopeCount: 3, openTasks: 7, updatedAt: iso(12) },
  { id: 'prj-helix', name: 'Helix Cloud Posture', client: 'Helix Bio', status: 'active', scopeCount: 2, openTasks: 4, updatedAt: iso(95) },
  { id: 'prj-orion', name: 'Orion Internal Red Team', client: 'Orion Finance', status: 'paused', scopeCount: 1, openTasks: 0, updatedAt: iso(2880) },
]

const scopes: Scope[] = [
  { id: 'scp-atlas-web', projectId: 'prj-atlas', name: 'Public web estate', kind: 'domain', targets: ['*.northwind.example', 'portal.northwind.example'], window: { from: iso(4320), to: iso(-4320) }, status: 'in-window' },
  { id: 'scp-atlas-net', projectId: 'prj-atlas', name: 'Edge netblocks', kind: 'cidr', targets: ['203.0.113.0/24'], window: { from: iso(4320), to: iso(-4320) }, status: 'in-window' },
  { id: 'scp-atlas-repo', projectId: 'prj-atlas', name: 'Customer SDK repo', kind: 'repo', targets: ['git.northwind.example/sdk'], window: { from: iso(-1440), to: iso(-10080) }, status: 'scheduled' },
  { id: 'scp-helix-aws', projectId: 'prj-helix', name: 'Production AWS account', kind: 'cloud-account', targets: ['aws:482910045517'], window: { from: iso(2880), to: iso(-2880) }, status: 'in-window' },
  { id: 'scp-helix-gcp', projectId: 'prj-helix', name: 'Analytics GCP project', kind: 'cloud-account', targets: ['gcp:helix-analytics'], window: { from: iso(20000), to: iso(10000) }, status: 'expired' },
  { id: 'scp-orion-hosts', projectId: 'prj-orion', name: 'Corporate hosts', kind: 'host', targets: ['10.20.0.0/22'], window: { from: iso(20000), to: iso(5000) }, status: 'expired' },
]

const tasks: Task[] = [
  { id: 'tsk-101', projectId: 'prj-atlas', scopeId: 'scp-atlas-web', title: 'TLS configuration sweep', capability: 'recon.tls.scan', state: 'done', assignee: 'worker-eu-1', updatedAt: iso(40) },
  { id: 'tsk-102', projectId: 'prj-atlas', scopeId: 'scp-atlas-web', title: 'Subdomain enumeration', capability: 'recon.dns.enum', state: 'running', assignee: 'worker-eu-2', updatedAt: iso(6) },
  { id: 'tsk-103', projectId: 'prj-atlas', scopeId: 'scp-atlas-net', title: 'Service fingerprinting', capability: 'recon.port.scan', state: 'granted', assignee: 'worker-eu-1', updatedAt: iso(3) },
  { id: 'tsk-104', projectId: 'prj-atlas', scopeId: 'scp-atlas-net', title: 'Credential spray check', capability: 'auth.spray', state: 'denied', assignee: '—', updatedAt: iso(55) },
  { id: 'tsk-105', projectId: 'prj-helix', scopeId: 'scp-helix-aws', title: 'IAM least-privilege review', capability: 'cloud.iam.audit', state: 'evaluating', assignee: 'worker-us-1', updatedAt: iso(1) },
  { id: 'tsk-106', projectId: 'prj-helix', scopeId: 'scp-helix-aws', title: 'Public bucket discovery', capability: 'cloud.storage.scan', state: 'queued', assignee: '—', updatedAt: iso(20) },
  { id: 'tsk-107', projectId: 'prj-helix', scopeId: 'scp-helix-aws', title: 'Secrets-in-config scan', capability: 'cloud.config.scan', state: 'failed', assignee: 'worker-us-1', updatedAt: iso(130) },
]

const workers: Worker[] = [
  { id: 'wrk-eu-1', projectId: 'prj-atlas', name: 'worker-eu-1', state: 'online', identityFingerprint: sampleId('eu1').slice(0, 23), lastHeartbeat: iso(0.4), load: 0.62, capabilities: ['recon.tls.scan', 'recon.port.scan'] },
  { id: 'wrk-eu-2', projectId: 'prj-atlas', name: 'worker-eu-2', state: 'online', identityFingerprint: sampleId('eu2').slice(0, 23), lastHeartbeat: iso(0.9), load: 0.31, capabilities: ['recon.dns.enum'] },
  { id: 'wrk-us-1', projectId: 'prj-helix', name: 'worker-us-1', state: 'stale', identityFingerprint: sampleId('us1').slice(0, 23), lastHeartbeat: iso(18), load: 0.0, capabilities: ['cloud.iam.audit', 'cloud.config.scan'] },
  { id: 'wrk-us-2', projectId: 'prj-helix', name: 'worker-us-2', state: 'suspended', identityFingerprint: sampleId('us2').slice(0, 23), lastHeartbeat: iso(240), load: 0.0, capabilities: ['cloud.storage.scan'] },
  { id: 'wrk-legacy', projectId: 'prj-orion', name: 'worker-legacy', state: 'revoked', identityFingerprint: sampleId('legacy').slice(0, 23), lastHeartbeat: iso(9000), load: 0, capabilities: [] },
]

const assignments: Assignment[] = [
  { id: 'asg-77', workerId: 'wrk-eu-2', taskId: 'tsk-102', state: 'leased', leaseExpires: iso(-9), resultVerified: false },
  { id: 'asg-76', workerId: 'wrk-eu-1', taskId: 'tsk-101', state: 'completed', leaseExpires: iso(30), resultVerified: true },
  { id: 'asg-75', workerId: 'wrk-us-1', taskId: 'tsk-107', state: 'stale', leaseExpires: iso(140), resultVerified: false },
  { id: 'asg-74', workerId: 'wrk-legacy', taskId: 'tsk-090', state: 'revoked', leaseExpires: iso(9100), resultVerified: false },
]

const packs: Pack[] = [
  { id: 'pk-recon', name: 'recon-core', version: '1.8.2', publisher: 'cyberos-official', digest: sampleId('recon-core'), signature: 'valid', trust: 'trusted', capabilities: ['recon.tls.scan', 'recon.dns.enum', 'recon.port.scan'] },
  { id: 'pk-cloud', name: 'cloud-audit', version: '0.9.4', publisher: 'cyberos-official', digest: sampleId('cloud-audit'), signature: 'valid', trust: 'trusted', capabilities: ['cloud.iam.audit', 'cloud.storage.scan', 'cloud.config.scan'] },
  { id: 'pk-auth', name: 'auth-probe', version: '0.3.0', publisher: 'community/redbench', digest: sampleId('auth-probe'), signature: 'valid', trust: 'untrusted', capabilities: ['auth.spray'] },
  { id: 'pk-fuzz', name: 'web-fuzz', version: '2.0.0-rc1', publisher: 'unknown', digest: sampleId('web-fuzz'), signature: 'missing', trust: 'untrusted', capabilities: ['web.fuzz'] },
  { id: 'pk-old', name: 'legacy-smb', version: '0.1.0', publisher: 'community/oldlab', digest: sampleId('legacy-smb'), signature: 'invalid', trust: 'revoked', capabilities: ['net.smb.enum'] },
]

const assets: Asset[] = [
  { id: 'ast-1', name: 'portal.northwind.example', kind: 'web application', exposure: 'internet', riskScore: 78 },
  { id: 'ast-2', name: '203.0.113.14', kind: 'host', exposure: 'internet', riskScore: 61 },
  { id: 'ast-3', name: 'helix-prod-data', kind: 'object storage', exposure: 'restricted', riskScore: 44 },
  { id: 'ast-4', name: 'ci-runner-pool', kind: 'compute', exposure: 'internal', riskScore: 23 },
]

const findings: Finding[] = [
  { id: 'fnd-9', title: 'TLS 1.0 still negotiated on portal', severity: 'high', assetId: 'ast-1', assessmentId: 'asm-3', status: 'open', foundAt: iso(38), evidenceDigest: sampleId('f9') },
  { id: 'fnd-8', title: 'Management interface exposed on 8443', severity: 'critical', assetId: 'ast-2', assessmentId: 'asm-3', status: 'triaged', foundAt: iso(300), evidenceDigest: sampleId('f8') },
  { id: 'fnd-7', title: 'Bucket policy allows broad principal', severity: 'medium', assetId: 'ast-3', assessmentId: 'asm-2', status: 'open', foundAt: iso(900), evidenceDigest: sampleId('f7') },
  { id: 'fnd-6', title: 'Missing HSTS header', severity: 'low', assetId: 'ast-1', assessmentId: 'asm-3', status: 'accepted', foundAt: iso(1500), evidenceDigest: sampleId('f6') },
  { id: 'fnd-5', title: 'Verbose server banner', severity: 'info', assetId: 'ast-2', assessmentId: 'asm-1', status: 'fixed', foundAt: iso(5000), evidenceDigest: sampleId('f5') },
]

const cases: CaseItem[] = [
  { id: 'case-12', title: 'Exposed management plane — Northwind edge', status: 'in-progress', severity: 'critical', owner: 'a.rahman', findingIds: ['fnd-8'] },
  { id: 'case-11', title: 'Legacy TLS retirement', status: 'open', severity: 'high', owner: 'unassigned', findingIds: ['fnd-9', 'fnd-6'] },
  { id: 'case-10', title: 'Helix bucket policy tightening', status: 'resolved', severity: 'medium', owner: 'j.okafor', findingIds: ['fnd-7'] },
]

const reports: Report[] = [
  { id: 'rpt-4', title: 'Atlas perimeter — interim findings', projectId: 'prj-atlas', generatedAt: iso(600), formats: ['pdf', 'json'] },
  { id: 'rpt-3', title: 'Helix posture baseline', projectId: 'prj-helix', generatedAt: iso(4000), formats: ['pdf', 'json', 'csv'] },
]

const audit: AuditEvent[] = [
  { id: 'ev-1', at: iso(1), actor: 'control-plane', action: 'decision.allow', subject: 'tsk-105 · cloud.iam.audit', tone: 'ok', digest: sampleId('ev1') },
  { id: 'ev-2', at: iso(3), actor: 'control-plane', action: 'grant.issue', subject: 'tsk-103 · recon.port.scan', tone: 'grant', digest: sampleId('ev2') },
  { id: 'ev-3', at: iso(6), actor: 'tool-broker', action: 'execution.start', subject: 'tsk-102 · recon.dns.enum', tone: 'info', digest: sampleId('ev3') },
  { id: 'ev-4', at: iso(40), actor: 'verifier', action: 'result.verified', subject: 'asg-76', tone: 'evidence', digest: sampleId('ev4') },
  { id: 'ev-5', at: iso(55), actor: 'control-plane', action: 'decision.deny', subject: 'tsk-104 · auth.spray (pack untrusted)', tone: 'danger', digest: sampleId('ev5') },
  { id: 'ev-6', at: iso(130), actor: 'worker-us-1', action: 'execution.fail', subject: 'tsk-107', tone: 'warn', digest: sampleId('ev6') },
  { id: 'ev-7', at: iso(240), actor: 'operator:a.rahman', action: 'worker.suspend', subject: 'wrk-us-2', tone: 'trust', digest: sampleId('ev7') },
]

/* ---------- Target-in-scope matching (strict, fail-closed) ---------- */

interface ParsedIp {
  v: 4 | 6
  n: bigint
}

function parseIPv4(s: string): ParsedIp | null {
  const parts = s.split('.')
  if (parts.length !== 4) return null
  let n = 0n
  for (const p of parts) {
    if (!/^(0|[1-9]\d{0,2})$/.test(p) || Number(p) > 255) return null
    n = (n << 8n) | BigInt(p)
  }
  return { v: 4, n }
}

function parseIPv6(s: string): ParsedIp | null {
  if (!/^[0-9a-f:]+$/i.test(s) || s.includes(':::')) return null
  const halves = s.split('::')
  if (halves.length > 2) return null
  const groups = (x: string) => (x === '' ? [] : x.split(':'))
  const head = groups(halves[0])
  const tail = halves.length === 2 ? groups(halves[1]) : []
  const missing = 8 - head.length - tail.length
  if (halves.length === 1 ? missing !== 0 : missing < 1) return null
  const fill = halves.length === 2 ? Array<string>(missing).fill('0') : []
  let n = 0n
  for (const g of [...head, ...fill, ...tail]) {
    if (!/^[0-9a-f]{1,4}$/i.test(g)) return null
    n = (n << 16n) | BigInt(parseInt(g, 16))
  }
  return { v: 6, n }
}

const parseIp = (s: string): ParsedIp | null => (s.includes(':') ? parseIPv6(s) : parseIPv4(s))

interface Net extends ParsedIp {
  len: number
}

function parseCidr(s: string): Net | null {
  const [addr, lenStr, ...rest] = s.split('/')
  if (rest.length > 0 || lenStr === undefined || !/^\d{1,3}$/.test(lenStr)) return null
  const ip = parseIp(addr)
  const len = Number(lenStr)
  if (!ip || len > (ip.v === 4 ? 32 : 128)) return null
  return { ...ip, len }
}

/** True when `target` (an IP or a CIDR) lies entirely inside `net`. */
function inNet(net: Net, target: string): boolean {
  const bits = net.v === 4 ? 32 : 128
  let t: Net | null
  if (target.includes('/')) t = parseCidr(target)
  else {
    const ip = parseIp(target)
    t = ip ? { ...ip, len: bits } : null
  }
  if (!t || t.v !== net.v || t.len < net.len) return false
  const shift = BigInt(bits - net.len)
  return t.n >> shift === net.n >> shift
}

const HOSTNAME = /^(?=.{1,253}$)[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/

function matchHostPattern(pattern: string, target: string): boolean {
  if (!HOSTNAME.test(target) && !parseIp(target)) return false
  if (pattern.startsWith('*.')) {
    const base = pattern.slice(2)
    // Strict label boundary: "<one or more valid labels>.<base>"; the apex itself is not covered.
    return HOSTNAME.test(base) && target.length > base.length + 1 && target.endsWith('.' + base)
  }
  return pattern === target
}

const normRepo = (s: string) =>
  s
    .trim()
    .toLowerCase()
    .replace(/^https?:\/\//, '')
    .replace(/\/+$/, '')
    .replace(/\.git$/, '')

/** Does `rawTarget` fall inside one of the scope's declared targets? Anything unrecognised is denied. */
export function targetInScope(scope: Pick<Scope, 'kind' | 'targets'>, rawTarget: string): boolean {
  const raw = rawTarget.trim()
  if (!raw || /\s/.test(raw)) return false
  const host = raw.toLowerCase().replace(/\.$/, '')
  return scope.targets.some((entry) => {
    const pattern = entry.trim().toLowerCase()
    switch (scope.kind) {
      case 'domain':
        return matchHostPattern(pattern, host)
      case 'host': {
        const net = pattern.includes('/') ? parseCidr(pattern) : null
        return net ? inNet(net, host) : matchHostPattern(pattern, host)
      }
      case 'cidr': {
        const net = parseCidr(pattern)
        return net !== null && inNet(net, raw.toLowerCase())
      }
      case 'repo':
        return normRepo(pattern) === normRepo(raw)
      case 'cloud-account':
        return pattern === raw.toLowerCase()
      default:
        return false
    }
  })
}

function simulate(req: SpineRequest): SpineTrace {
  const scope = scopes.find((s) => s.id === req.scopeId)
  const pack = packs.find((p) => p.capabilities.includes(req.capability))
  const stages: SpineStage[] = []
  const id = 'req-' + sampleId(JSON.stringify(req)).slice(7, 15)

  const scopeActive = !!scope && scope.status === 'in-window' && scope.projectId === req.projectId
  const targetOk = !!scope && targetInScope(scope, req.target)
  const scopeOk = scopeActive && targetOk
  stages.push({
    id: 'scope',
    label: 'Scope',
    outcome: scopeOk ? 'pass' : 'deny',
    detail: !scope
      ? 'Scope not found for this project.'
      : scope.projectId !== req.projectId
        ? 'Scope does not belong to this project.'
        : scope.status !== 'in-window'
          ? `Scope is ${scope.status}; outside its authorised window.`
          : !targetOk
            ? `Target “${req.target.trim() || '(empty)'}” is not listed in scope “${scope.name}”.`
            : `Target inside “${scope.name}” and its authorised window.`,
    ref: scope?.id,
  })

  const packOk = !!pack && pack.trust === 'trusted' && pack.signature === 'valid'
  const decisionOk = scopeOk && packOk
  stages.push({
    id: 'decision',
    label: 'Decision',
    outcome: !scopeOk ? 'skipped' : packOk ? 'pass' : 'deny',
    detail: !scopeOk ? 'Not evaluated.' : !pack ? 'No pack provides this capability.' : packOk ? `Capability provided by trusted pack ${pack.name}@${pack.version}.` : `Pack ${pack.name} is ${pack.trust} (signature ${pack.signature}).`,
    ref: scopeOk ? 'dec-' + sampleId(id).slice(7, 13) : undefined,
  })

  const rest: [SpineStage['id'], string, string][] = [
    ['grant', 'Grant', 'Short-lived grant bound to principal, capability, target and pack digest.'],
    ['broker', 'Broker', 'Tool broker verified the grant before dispatch.'],
    ['execution', 'Execution', 'Executed on an attested worker.'],
    ['evidence', 'Evidence', 'Signed result recorded; verification performed by the backend.'],
  ]
  for (const [sid, label, detail] of rest) {
    stages.push({
      id: sid,
      label,
      outcome: decisionOk ? 'pass' : 'skipped',
      detail: decisionOk ? detail : 'Not reached — stopped fail-closed upstream.',
      ref: decisionOk ? sid.slice(0, 3) + '-' + sampleId(id + sid).slice(7, 13) : undefined,
    })
  }
  return { requestId: id, request: req, stages, final: decisionOk ? 'completed' : 'denied', simulated: true }
}

const summary: DashboardSummary = {
  projects: projects.length,
  activeScopes: scopes.filter((s) => s.status === 'in-window').length,
  openFindings: findings.filter((f) => f.status === 'open' || f.status === 'triaged').length,
  grantsIssued24h: 42,
  denials24h: 9,
  workersOnline: workers.filter((w) => w.state === 'online').length,
  workersTotal: workers.length,
  executionsPerHour: [4, 6, 3, 8, 12, 9, 14, 11, 16, 13, 18, 15, 12, 17, 21, 19, 14, 10, 9, 12, 15, 18, 16, 13],
  severityMix: { critical: 1, high: 1, medium: 1, low: 1, info: 1 },
}

const system: SystemInfo = {
  version: '0.25.0-preview',
  commit: 'preview',
  status: 'degraded',
  mode: 'preview',
  checks: [
    { name: 'API reachable', ok: false, note: 'Preview mode — no backend connection.' },
    { name: 'Storage', ok: false, note: 'Not queried in preview.' },
    { name: 'Worker mTLS', ok: false, note: 'Not queried in preview.' },
    { name: 'Trust registry', ok: false, note: 'Not queried in preview.' },
  ],
}

export const previewApi: CyberosApi = {
  mode: 'preview',
  dashboard: () => wait(summary),
  system: () => wait(system),
  projects: () => wait(projects),
  scopes: (projectId) => wait(projectId ? scopes.filter((s) => s.projectId === projectId) : scopes),
  tasks: (projectId) => wait(projectId ? tasks.filter((t) => t.projectId === projectId) : tasks),
  capabilities: () => wait([...new Set(packs.flatMap((p) => p.capabilities))].sort()),
  submitSpineRequest: (req) => wait(simulate(req), 700),
  workers: () => wait(workers),
  assignments: () => wait(assignments),
  packs: () => wait(packs),
  assets: () => wait(assets),
  findings: () => wait(findings),
  cases: () => wait(cases),
  reports: () => wait(reports),
  audit: () => wait(audit),
}
