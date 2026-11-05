import type { Assignment, Finding, Pack, Project, Scope, Severity, Task, Tone, Worker, CaseItem } from '@/api/types'
import { Badge } from '@/components/ui'

/** Presentation-only mappings from backend state to visual tone. */

const severityTone: Record<Severity, Tone> = { critical: 'danger', high: 'danger', medium: 'warn', low: 'info', info: 'neutral' }
export const SeverityBadge = ({ severity }: { severity: Severity }) => (
  <Badge tone={severityTone[severity]} dot>
    {severity}
  </Badge>
)

const taskTone: Record<Task['state'], Tone> = { queued: 'neutral', evaluating: 'info', granted: 'grant', running: 'info', done: 'ok', denied: 'danger', failed: 'warn' }
export const TaskStateBadge = ({ state }: { state: Task['state'] }) => (
  <Badge tone={taskTone[state]} live={state === 'running' || state === 'evaluating'} dot={state !== 'running' && state !== 'evaluating'}>
    {state}
  </Badge>
)

const workerTone: Record<Worker['state'], Tone> = { online: 'ok', suspended: 'warn', revoked: 'danger', stale: 'warn' }
export const WorkerStateBadge = ({ state }: { state: Worker['state'] }) => (
  <Badge tone={workerTone[state]} live={state === 'online'} dot={state !== 'online'}>
    {state}
  </Badge>
)

const assignTone: Record<Assignment['state'], Tone> = { leased: 'info', completed: 'ok', revoked: 'danger', stale: 'warn' }
export const AssignmentStateBadge = ({ state }: { state: Assignment['state'] }) => (
  <Badge tone={assignTone[state]} dot>
    {state}
  </Badge>
)

const trustTone: Record<Pack['trust'], Tone> = { trusted: 'trust', untrusted: 'warn', revoked: 'danger' }
export const TrustBadge = ({ trust }: { trust: Pack['trust'] }) => (
  <Badge tone={trustTone[trust]} dot>
    {trust}
  </Badge>
)

const sigTone: Record<Pack['signature'], Tone> = { valid: 'ok', invalid: 'danger', missing: 'warn' }
export const SignatureBadge = ({ signature }: { signature: Pack['signature'] }) => (
  <Badge tone={sigTone[signature]} dot>
    sig {signature}
  </Badge>
)

const scopeTone: Record<Scope['status'], Tone> = { 'in-window': 'ok', scheduled: 'info', expired: 'danger' }
export const ScopeStatusBadge = ({ status }: { status: Scope['status'] }) => (
  <Badge tone={scopeTone[status]} dot>
    {status}
  </Badge>
)

const projectTone: Record<Project['status'], Tone> = { active: 'ok', paused: 'warn', closed: 'neutral' }
export const ProjectStatusBadge = ({ status }: { status: Project['status'] }) => (
  <Badge tone={projectTone[status]} dot>
    {status}
  </Badge>
)

const findingTone: Record<Finding['status'], Tone> = { open: 'danger', triaged: 'warn', accepted: 'info', fixed: 'ok' }
export const FindingStatusBadge = ({ status }: { status: Finding['status'] }) => (
  <Badge tone={findingTone[status]} dot>
    {status}
  </Badge>
)

const caseTone: Record<CaseItem['status'], Tone> = { open: 'warn', 'in-progress': 'info', resolved: 'ok' }
export const CaseStatusBadge = ({ status }: { status: CaseItem['status'] }) => (
  <Badge tone={caseTone[status]} dot>
    {status}
  </Badge>
)
