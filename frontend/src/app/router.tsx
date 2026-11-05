import { lazy, Suspense, type ComponentType } from 'react'
import { createBrowserRouter, Navigate, type RouteObject } from 'react-router-dom'
import { RequireSession } from './guards/RequireSession'
import { AppShell } from './layouts/AppShell'
import { SiteLayout } from './layouts/SiteLayout'

const page = (loader: () => Promise<Record<string, unknown>>, name: string) =>
  lazy(() => loader().then((m) => ({ default: m[name] as ComponentType })))

const Home = page(() => import('@/pages/site/HomePage'), 'HomePage')
const Platform = page(() => import('@/pages/site/MarketingPages'), 'PlatformPage')
const Architecture = page(() => import('@/pages/site/MarketingPages'), 'ArchitecturePage')
const Security = page(() => import('@/pages/site/MarketingPages'), 'SecurityPage')
const SignIn = page(() => import('@/pages/auth/SignInPage'), 'SignInPage')
const NotFound = page(() => import('@/pages/NotFoundPage'), 'NotFoundPage')

const Dashboard = page(() => import('@/pages/app/DashboardPage'), 'DashboardPage')
const Projects = page(() => import('@/pages/app/EngagementPages'), 'ProjectsPage')
const ProjectDetail = page(() => import('@/pages/app/EngagementPages'), 'ProjectDetailPage')
const Scopes = page(() => import('@/pages/app/EngagementPages'), 'ScopesPage')
const Tasks = page(() => import('@/pages/app/EngagementPages'), 'TasksPage')
const Assessments = page(() => import('@/pages/app/EngagementPages'), 'AssessmentsPage')
const Assets = page(() => import('@/pages/app/IntelPages'), 'AssetsPage')
const Findings = page(() => import('@/pages/app/IntelPages'), 'FindingsPage')
const Risk = page(() => import('@/pages/app/IntelPages'), 'RiskPage')
const Cases = page(() => import('@/pages/app/IntelPages'), 'CasesPage')
const Reports = page(() => import('@/pages/app/IntelPages'), 'ReportsPage')
const Evaluate = page(() => import('@/pages/app/ControlPages'), 'EvaluatePage')
const Decisions = page(() => import('@/pages/app/ControlPages'), 'DecisionsPage')
const Grants = page(() => import('@/pages/app/ControlPages'), 'GrantsPage')
const Execution = page(() => import('@/pages/app/ControlPages'), 'ExecutionPage')
const Evidence = page(() => import('@/pages/app/ProofPages'), 'EvidencePage')
const Provenance = page(() => import('@/pages/app/ProofPages'), 'ProvenancePage')
const Workers = page(() => import('@/pages/app/FleetPages'), 'WorkersPage')
const Assignments = page(() => import('@/pages/app/FleetPages'), 'AssignmentsPage')
const Packs = page(() => import('@/pages/app/FleetPages'), 'PacksPage')
const Autonomy = page(() => import('@/pages/app/PlatformPages'), 'AutonomyPage')
const Events = page(() => import('@/pages/app/PlatformPages'), 'EventsPage')
const System = page(() => import('@/pages/app/PlatformPages'), 'SystemPage')
const Settings = page(() => import('@/pages/app/PlatformPages'), 'SettingsPage')

const lazyEl = (C: ComponentType) => (
  <Suspense fallback={null}>
    <C />
  </Suspense>
)

const routes: RouteObject[] = [
  {
    element: <SiteLayout />,
    children: [
      { index: true, element: lazyEl(Home) },
      { path: 'platform', element: lazyEl(Platform) },
      { path: 'architecture', element: lazyEl(Architecture) },
      { path: 'security', element: lazyEl(Security) },
    ],
  },
  { path: 'sign-in', element: lazyEl(SignIn) },
  { path: 'login', element: <Navigate to="/sign-in" replace /> },
  {
    path: 'app',
    element: (
      <RequireSession>
        <AppShell />
      </RequireSession>
    ),
    children: [
      { index: true, element: <Dashboard /> },
      { path: 'projects', element: <Projects /> },
      { path: 'projects/:projectId', element: <ProjectDetail /> },
      { path: 'scopes', element: <Scopes /> },
      { path: 'tasks', element: <Tasks /> },
      { path: 'assessments', element: <Assessments /> },
      { path: 'assets', element: <Assets /> },
      { path: 'findings', element: <Findings /> },
      { path: 'risk', element: <Risk /> },
      { path: 'cases', element: <Cases /> },
      { path: 'reports', element: <Reports /> },
      { path: 'evaluate', element: <Evaluate /> },
      { path: 'decisions', element: <Decisions /> },
      { path: 'grants', element: <Grants /> },
      { path: 'execution', element: <Execution /> },
      { path: 'evidence', element: <Evidence /> },
      { path: 'provenance', element: <Provenance /> },
      { path: 'workers', element: <Workers /> },
      { path: 'assignments', element: <Assignments /> },
      { path: 'packs', element: <Packs /> },
      { path: 'autonomy', element: <Autonomy /> },
      { path: 'events', element: <Events /> },
      { path: 'system', element: <System /> },
      { path: 'settings', element: <Settings /> },
      { path: '*', element: <NotFound /> },
    ],
  },
  { path: '*', element: lazyEl(NotFound) },
]

export const router = createBrowserRouter(routes)
