import { Suspense, useCallback, useState } from 'react'
import { useOverlay, useSpotlight } from '@/app/effects'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { LogOut, Menu } from 'lucide-react'
import { useApi } from '@/api'
import { Wordmark } from '@/components/brand/Logo'
import { Banner, Skeleton } from '@/components/ui'
import { ALL_NAV, NAV } from '@/app/nav'
import { useSession } from '@/app/providers/session-context'
import { ThemeToggle } from './ThemeToggle'

function PageFallback() {
  return (
    <div className="col" aria-busy="true">
      <Skeleton h={30} w="30%" />
      <Skeleton h={120} />
      <Skeleton h={240} />
    </div>
  )
}

export function AppShell() {
  const api = useApi()
  const session = useSession()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  // The drawer is "open for" the path it was opened on, so it closes by itself on navigation.
  const [openAt, setOpenAt] = useState<string | null>(null)
  const navOpen = openAt === pathname
  const setNavOpen = (open: boolean) => setOpenAt(open ? pathname : null)
  const closeNav = useCallback(() => setOpenAt(null), [])
  useOverlay(navOpen, closeNav)
  useSpotlight()

  const current = [...ALL_NAV].reverse().find((n) => (n.end ? pathname === n.to : pathname.startsWith(n.to)))

  return (
    <div className="shell" data-nav={navOpen ? 'open' : 'closed'}>
      <a href="#content" className="skip-link">
        Skip to content
      </a>
      <aside className="sidebar" aria-label="Console navigation">
        <Link to="/" aria-label="CYBEROS public site">
          <Wordmark />
        </Link>
        {NAV.map((group) => (
          <nav key={group.title} className="nav-group" aria-label={group.title}>
            <span className="nav-title">{group.title}</span>
            {group.items.map((item) => (
              <NavLink key={item.to} to={item.to} end={item.end} className="nav-link">
                <item.icon size={16} aria-hidden="true" />
                {item.label}
              </NavLink>
            ))}
          </nav>
        ))}
        <span className="spacer" />
        {session.principalId && (
          <div className="user-chip" title={session.tenantId ? `Tenant ${session.tenantId}` : undefined}>
            <span className="avatar" aria-hidden="true">
              {session.principalId.slice(0, 1).toUpperCase()}
            </span>
            <span className="mono truncate">{session.principalId}</span>
          </div>
        )}
        <button
          type="button"
          className="nav-link"
          onClick={() => {
            void session.signOut().finally(() => navigate('/sign-in', { replace: true }))
          }}
        >
          <LogOut size={16} aria-hidden="true" />
          Sign out
        </button>
      </aside>
      <div className="shell-scrim" onClick={() => setNavOpen(false)} aria-hidden="true" />

      <div className="col" style={{ gap: 0, minWidth: 0 }}>
        <header className="topbar">
          <button type="button" className="btn btn-ghost btn-icon btn-sm shell-toggle" aria-label="Open navigation" aria-expanded={navOpen} onClick={() => setNavOpen(!navOpen)}>
            <Menu size={18} />
          </button>
          <div className="crumbs">
            <span>Console</span>
            <span aria-hidden="true">/</span>
            <b className="truncate">{current?.label ?? 'Not found'}</b>
          </div>
          <span className="spacer" />
          <span className="mode-chip" data-mode={api.mode} title={api.mode === 'preview' ? 'Showing built-in sample data. No backend is connected.' : 'Connected to the CYBEROS API.'}>
            {api.mode === 'preview' ? 'Preview data' : 'Live'}
          </span>
          <ThemeToggle />
        </header>
        <main id="content" className="content" tabIndex={-1}>
          {session.status === 'open' && (
            <Banner tone="warn" title="Authentication is disabled on this server">
              The backend is running in explicit development mode without sign-in. Do not expose this instance.
            </Banner>
          )}
          {api.mode === 'preview' && (
            <Banner tone="warn" title="Preview data — not connected to the CYBEROS backend">
              Everything on this screen is sample data from a local adapter. Decisions, grants, trust and verification are always produced by the Go control plane, never by this browser.
            </Banner>
          )}
          <Suspense fallback={<PageFallback />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  )
}
