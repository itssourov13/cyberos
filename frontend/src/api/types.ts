import type { DashboardSnapshot, Project, Worker, Assignment, Decision, Grant } from '@/types/models'
import type {
  MockAssessment,
  MockAsset,
  MockAuditRow,
  MockCase,
  MockFinding,
  MockPack,
  MockReport,
  MockRisk,
  MockScope,
  MockSigner,
  MockTask,
} from './mock/data'

export interface ApiClient {
  getDashboard(): Promise<DashboardSnapshot>
  listProjects(): Promise<Project[]>
  getProject(id: string): Promise<Project | undefined>
  listScopes(projectId?: string): Promise<MockScope[]>
  listTasks(projectId?: string): Promise<MockTask[]>
  getTask(id: string): Promise<MockTask | undefined>
  listDecisions(): Promise<Decision[]>
  listGrants(): Promise<Grant[]>
  mockEvaluate(input: {
    projectId: string
    scopeId: string
    taskId: string
    capabilityId: string
  }): Promise<Decision>
  listWorkers(projectId?: string): Promise<Worker[]>
  getWorker(id: string): Promise<Worker | undefined>
  listAssignments(projectId?: string): Promise<Assignment[]>
  getAssignment(id: string): Promise<Assignment | undefined>
  listAssessments(): Promise<MockAssessment[]>
  listFindings(): Promise<MockFinding[]>
  listAssets(): Promise<MockAsset[]>
  listRisks(): Promise<MockRisk[]>
  listCases(): Promise<MockCase[]>
  listReports(): Promise<MockReport[]>
  listPacks(): Promise<MockPack[]>
  listSigners(): Promise<MockSigner[]>
  listAudit(filters?: { projectId?: string; type?: string; q?: string }): Promise<MockAuditRow[]>
  getHealth(): Promise<{ status: string }>
}

export type { DashboardSnapshot, Project, Worker, Assignment, Decision, Grant }
