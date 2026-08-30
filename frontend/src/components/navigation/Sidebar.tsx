import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  FolderKanban,
  Target,
  ListTodo,
  ShieldCheck,
  KeyRound,
  Play,
  ClipboardList,
  Bug,
  Server,
  AlertTriangle,
  Briefcase,
  FileText,
  Cpu,
  GitBranch,
  Package,
  BadgeCheck,
  ScrollText,
  Stethoscope,
  Settings,
} from 'lucide-react'

type Item = { to: string; label: string; icon: ReactNode; danger?: boolean }

const groups: { title: string; items: Item[] }[] = [
  {
    title: 'Workspace',
    items: [
      { to: '/dashboard', label: 'Dashboard', icon: <LayoutDashboard size={15} /> },
      { to: '/projects', label: 'Projects', icon: <FolderKanban size={15} /> },
      { to: '/scopes', label: 'Scopes', icon: <Target size={15} /> },
      { to: '/tasks', label: 'Tasks', icon: <ListTodo size={15} /> },
    ],
  },
  {
    title: 'Operations',
    items: [
      { to: '/evaluate', label: 'Evaluate', icon: <ShieldCheck size={15} /> },
      { to: '/grants', label: 'Grants', icon: <KeyRound size={15} /> },
      { to: '/execute', label: 'Execute', icon: <Play size={15} />, danger: true },
    ],
  },
  {
    title: 'Security',
    items: [
      { to: '/assessments', label: 'Assessments', icon: <ClipboardList size={15} /> },
      { to: '/findings', label: 'Findings', icon: <Bug size={15} /> },
      { to: '/assets', label: 'Assets', icon: <Server size={15} /> },
      { to: '/risks', label: 'Risks', icon: <AlertTriangle size={15} /> },
      { to: '/cases', label: 'Cases', icon: <Briefcase size={15} /> },
      { to: '/reports', label: 'Reports', icon: <FileText size={15} /> },
    ],
  },
  {
    title: 'Fleet',
    items: [
      { to: '/workers', label: 'Workers', icon: <Cpu size={15} /> },
      { to: '/assignments', label: 'Assignments', icon: <GitBranch size={15} /> },
    ],
  },
  {
    title: 'Supply Chain',
    items: [
      { to: '/packs', label: 'Packs', icon: <Package size={15} /> },
      { to: '/trust', label: 'Trust', icon: <BadgeCheck size={15} /> },
    ],
  },
  {
    title: 'Observability',
    items: [{ to: '/audit', label: 'Audit', icon: <ScrollText size={15} /> }],
  },
  {
    title: 'System',
    items: [
      { to: '/system', label: 'System', icon: <Settings size={15} /> },
      { to: '/doctor', label: 'Doctor', icon: <Stethoscope size={15} /> },
    ],
  },
]

export function Sidebar() {
  return (
    <aside className="flex h-full w-[15.5rem] shrink-0 flex-col border-r border-[var(--color-border)] bg-[var(--color-surface-1)]">
      <div className="border-b border-[var(--color-border)] px-4 py-3.5">
        <div className="text-[11px] font-bold tracking-[0.2em] text-[var(--color-fg)]">CYBEROS</div>
        <div className="mt-0.5 text-[10px] uppercase tracking-[0.14em] text-[var(--color-muted)]">
          Operator Console
        </div>
      </div>
      <nav className="flex-1 overflow-y-auto px-2 py-3" aria-label="Primary">
        {groups.map((g) => (
          <div key={g.title} className="mb-3.5">
            <div className="mb-1 px-2 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
              {g.title}
            </div>
            <ul className="space-y-0.5">
              {g.items.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) =>
                      `flex items-center gap-2 rounded-[var(--radius-sm)] px-2 py-1.5 text-[13px] transition-colors duration-150 ${
                        isActive
                          ? item.danger
                            ? 'bg-orange-500/12 text-orange-200 ring-1 ring-orange-500/25'
                            : 'bg-[var(--color-surface-3)] text-[var(--color-fg)] ring-1 ring-[var(--color-border)]'
                          : 'text-[var(--color-muted)] hover:bg-[var(--color-surface-2)] hover:text-[var(--color-fg-secondary)]'
                      }`
                    }
                  >
                    <span className="opacity-80">{item.icon}</span>
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>
      <div className="border-t border-[var(--color-border)] px-3 py-2 text-[10px] text-[var(--color-muted)]">
        Thin client · Server authoritative
      </div>
    </aside>
  )
}
