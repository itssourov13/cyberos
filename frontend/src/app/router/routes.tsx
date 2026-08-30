import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/app/layout/AppShell'
import { DashboardPage } from '@/pages/DashboardPage'
import { LoginPage } from '@/pages/LoginPage'
import { ProjectsPage } from '@/pages/ProjectsPage'
import { ProjectDetailPage } from '@/pages/ProjectDetailPage'
import { ScopesPage } from '@/pages/ScopesPage'
import { TasksPage } from '@/pages/TasksPage'
import { TaskDetailPage } from '@/pages/TaskDetailPage'
import { EvaluatePage } from '@/pages/EvaluatePage'
import { GrantsPage } from '@/pages/GrantsPage'
import { ExecutePage } from '@/pages/ExecutePage'
import {
  AssessmentsPage,
  FindingsPage,
  AssetsPage,
  RisksPage,
  CasesPage,
  ReportsPage,
} from '@/pages/DomainPages'
import {
  WorkersPage,
  WorkerDetailPage,
  AssignmentsPage,
  AssignmentDetailPage,
} from '@/pages/FleetPages'
import {
  PacksPage,
  TrustPage,
  AuditPage,
  SystemPage,
  DoctorPage,
} from '@/pages/SupplyAuditSystemPages'

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route element={<AppShell />}>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/projects/:id" element={<ProjectDetailPage />} />
        <Route path="/scopes" element={<ScopesPage />} />
        <Route path="/tasks" element={<TasksPage />} />
        <Route path="/tasks/:id" element={<TaskDetailPage />} />
        <Route path="/evaluate" element={<EvaluatePage />} />
        <Route path="/grants" element={<GrantsPage />} />
        <Route path="/execute" element={<ExecutePage />} />
        <Route path="/assessments" element={<AssessmentsPage />} />
        <Route path="/findings" element={<FindingsPage />} />
        <Route path="/assets" element={<AssetsPage />} />
        <Route path="/risks" element={<RisksPage />} />
        <Route path="/cases" element={<CasesPage />} />
        <Route path="/reports" element={<ReportsPage />} />
        <Route path="/workers" element={<WorkersPage />} />
        <Route path="/workers/:id" element={<WorkerDetailPage />} />
        <Route path="/assignments" element={<AssignmentsPage />} />
        <Route path="/assignments/:id" element={<AssignmentDetailPage />} />
        <Route path="/packs" element={<PacksPage />} />
        <Route path="/trust" element={<TrustPage />} />
        <Route path="/audit" element={<AuditPage />} />
        <Route path="/system" element={<SystemPage />} />
        <Route path="/doctor" element={<DoctorPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
