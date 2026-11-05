import { useEffect, useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { Menu, X } from 'lucide-react'
import { Wordmark } from '@/components/brand/Logo'
import { ThemeToggle } from './ThemeToggle'

const LINKS = [
  { to: '/platform', label: 'Platform' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/security', label: 'Security model' },
]

export function SiteLayout() {
  const { pathname } = useLocation()
  // Menu is "open for" the path it was opened on, so it closes by itself on navigation.
  const [openAt, setOpenAt] = useState<string | null>(null)
  const open = openAt === pathname
  useEffect(() => {
    window.scrollTo({ top: 0 })
  }, [pathname])

  return (
    <>
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <header className="site-header">
        <div className="container">
          <Link to="/" aria-label="CYBEROS home">
            <Wordmark />
          </Link>
          <nav className={open ? 'site-nav open' : 'site-nav'} aria-label="Primary">
            {LINKS.map((l) => (
              <NavLink key={l.to} to={l.to}>
                {l.label}
              </NavLink>
            ))}
          </nav>
          <span className="spacer" />
          <ThemeToggle />
          <Link to="/sign-in" className="btn btn-ghost site-cta-desktop">
            Sign in
          </Link>
          <Link to="/app" className="btn btn-primary site-cta-desktop">
            Open console
          </Link>
          <button type="button" className="btn btn-ghost btn-icon menu-btn" aria-expanded={open} aria-label="Toggle menu" onClick={() => setOpenAt(open ? null : pathname)}>
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </header>
      <main id="main">
        <Outlet />
      </main>
      <footer className="site-footer">
        <div className="container">
          <Wordmark size={22} />
          <span>Authorized security operations, proven.</span>
          <span className="spacer" />
          <nav aria-label="Footer">
            {LINKS.map((l) => (
              <Link key={l.to} to={l.to}>
                {l.label}
              </Link>
            ))}
            <Link to="/sign-in">Sign in</Link>
          </nav>
        </div>
      </footer>
    </>
  )
}
