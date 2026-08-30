import type { ApiClient } from '../types'
import {
  MOCK_ASSIGNMENTS,
  MOCK_ASSESSMENTS,
  MOCK_ASSETS,
  MOCK_AUDIT,
  MOCK_CASES,
  MOCK_DASHBOARD,
  MOCK_DECISIONS,
  MOCK_FINDINGS,
  MOCK_GRANTS,
  MOCK_PACKS,
  MOCK_PROJECTS,
  MOCK_REPORTS,
  MOCK_RISKS,
  MOCK_SCOPES,
  MOCK_SIGNERS,
  MOCK_TASKS,
  MOCK_WORKERS,
} from './data'

function delay(ms = 80) {
  return new Promise((r) => setTimeout(r, ms))
}

export const mockApiClient: ApiClient = {
  async getDashboard() {
    await delay()
    return structuredClone(MOCK_DASHBOARD)
  },
  async listProjects() {
    await delay()
    return structuredClone(MOCK_PROJECTS)
  },
  async getProject(id) {
    await delay()
    return structuredClone(MOCK_PROJECTS.find((p) => p.id === id))
  },
  async listScopes(projectId) {
    await delay()
    const all = structuredClone(MOCK_SCOPES)
    return projectId ? all.filter((s) => s.projectId === projectId) : all
  },
  async listTasks(projectId) {
    await delay()
    const all = structuredClone(MOCK_TASKS)
    return projectId ? all.filter((t) => t.projectId === projectId) : all
  },
  async getTask(id) {
    await delay()
    return structuredClone(MOCK_TASKS.find((t) => t.id === id))
  },
  async listDecisions() {
    await delay()
    return structuredClone(MOCK_DECISIONS)
  },
  async listGrants() {
    await delay()
    return structuredClone(MOCK_GRANTS)
  },
  async mockEvaluate(input) {
    await delay(120)
    return {
      id: `dec-mock-${Date.now()}`,
      outcome: 'ALLOW',
      reason: 'MOCK evaluate — server CP not called; display only',
      projectId: input.projectId,
      scopeId: input.scopeId,
      capabilityId: input.capabilityId,
      correlationId: `corr-mock-${Date.now()}`,
      timestamp: new Date().toISOString(),
    }
  },
  async listWorkers(projectId) {
    await delay()
    const all = structuredClone(MOCK_WORKERS)
    return projectId ? all.filter((w) => w.projectId === projectId) : all
  },
  async getWorker(id) {
    await delay()
    return structuredClone(MOCK_WORKERS.find((w) => w.id === id))
  },
  async listAssignments(projectId) {
    await delay()
    const all = structuredClone(MOCK_ASSIGNMENTS)
    return projectId ? all.filter((a) => a.projectId === projectId) : all
  },
  async getAssignment(id) {
    await delay()
    return structuredClone(MOCK_ASSIGNMENTS.find((a) => a.id === id))
  },
  async listAssessments() {
    await delay()
    return structuredClone(MOCK_ASSESSMENTS)
  },
  async listFindings() {
    await delay()
    return structuredClone(MOCK_FINDINGS)
  },
  async listAssets() {
    await delay()
    return structuredClone(MOCK_ASSETS)
  },
  async listRisks() {
    await delay()
    return structuredClone(MOCK_RISKS)
  },
  async listCases() {
    await delay()
    return structuredClone(MOCK_CASES)
  },
  async listReports() {
    await delay()
    return structuredClone(MOCK_REPORTS)
  },
  async listPacks() {
    await delay()
    return structuredClone(MOCK_PACKS)
  },
  async listSigners() {
    await delay()
    return structuredClone(MOCK_SIGNERS)
  },
  async listAudit(filters) {
    await delay()
    let rows = structuredClone(MOCK_AUDIT)
    if (filters?.projectId) rows = rows.filter((r) => r.projectId === filters.projectId)
    if (filters?.type && filters.type !== 'ALL') rows = rows.filter((r) => r.type === filters.type)
    if (filters?.q) {
      const q = filters.q.toLowerCase()
      rows = rows.filter(
        (r) =>
          r.message.toLowerCase().includes(q) ||
          r.correlationId.toLowerCase().includes(q) ||
          (r.grantId ?? '').includes(q) ||
          (r.assignmentId ?? '').includes(q) ||
          (r.workerId ?? '').includes(q),
      )
    }
    return rows
  },
  async getHealth() {
    return { status: 'ok' }
  },
}
