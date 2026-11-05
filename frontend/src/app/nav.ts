import {
  Bot,
  Briefcase,
  Bug,
  ClipboardCheck,
  Cpu,
  Crosshair,
  FileText,
  Fingerprint,
  Gauge,
  GitBranch,
  HeartPulse,
  KeyRound,
  LayoutDashboard,
  Link2,
  ListChecks,
  Package,
  Play,
  ScrollText,
  Scale,
  Server,
  Settings,
  type LucideIcon,
} from 'lucide-react'

export interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
}
export interface NavGroup {
  title: string
  items: NavItem[]
}

/** Operator console information architecture. Mirrors the execution spine. */
export const NAV: NavGroup[] = [
  { title: 'Overview', items: [{ to: '/app', label: 'Dashboard', icon: LayoutDashboard, end: true }] },
  {
    title: 'Engagements',
    items: [
      { to: '/app/projects', label: 'Projects', icon: Briefcase },
      { to: '/app/scopes', label: 'Scopes', icon: Crosshair },
      { to: '/app/tasks', label: 'Tasks', icon: ListChecks },
      { to: '/app/assessments', label: 'Assessments', icon: ClipboardCheck },
    ],
  },
  {
    title: 'Intelligence',
    items: [
      { to: '/app/assets', label: 'Assets', icon: Server },
      { to: '/app/findings', label: 'Findings', icon: Bug },
      { to: '/app/risk', label: 'Risk', icon: Gauge },
      { to: '/app/cases', label: 'Cases', icon: Briefcase },
      { to: '/app/reports', label: 'Reports', icon: FileText },
    ],
  },
  {
    title: 'Control plane',
    items: [
      { to: '/app/evaluate', label: 'Evaluate', icon: Play },
      { to: '/app/decisions', label: 'Decisions', icon: Scale },
      { to: '/app/grants', label: 'Grants', icon: KeyRound },
      { to: '/app/execution', label: 'Execution', icon: Cpu },
    ],
  },
  {
    title: 'Proof',
    items: [
      { to: '/app/evidence', label: 'Evidence', icon: Fingerprint },
      { to: '/app/provenance', label: 'Provenance', icon: GitBranch },
    ],
  },
  {
    title: 'Fleet & supply',
    items: [
      { to: '/app/workers', label: 'Workers', icon: Server },
      { to: '/app/assignments', label: 'Assignments', icon: Link2 },
      { to: '/app/packs', label: 'Packs & trust', icon: Package },
    ],
  },
  {
    title: 'Platform',
    items: [
      { to: '/app/autonomy', label: 'Autonomy', icon: Bot },
      { to: '/app/events', label: 'Events & audit', icon: ScrollText },
      { to: '/app/system', label: 'System health', icon: HeartPulse },
      { to: '/app/settings', label: 'Settings', icon: Settings },
    ],
  },
]

export const ALL_NAV = NAV.flatMap((g) => g.items)
