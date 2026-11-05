import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { ArrowRight, Menu, ShieldCheck, X } from 'lucide-react'
import { Wordmark } from '@/components/brand/Logo'
import { useOverlay, useReveal, useScrollState, useSpotlight } from '@/app/effects'
import { ThemeToggle } from './ThemeToggle'

const LINKS = [
  { to: '/platform', label: 'Platform' },
  { to: '/architecture', label: 'Architecture' },
  { to: '/security', label: 'Security model' },
]

const CURRENT_YEAR = new Date().getFullYear()

export function SiteLayout() {
  const { pathname } = useLocation()
  // Menu is "open for" the path it was opened on, so it closes by itself on navigation.
  const [openAt, setOpenAt] = useState<string | null>(null)
  const open = openAt === pathname
  const close = useCallback(() => setOpenAt(null), [])
  const headerRef = useRef<HTMLElement>(null)

  useEffect(() => {
    window.scrollTo({ top: 0 })
  }, [pathname])
  useScrollState(() => headerRef.current)
  useOverlay(open, close)
  useReveal()
  useSpotlight()

  return (
    <>
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <header className="site-header" ref={headerRef} data-menu={open ? 'open' : 'closed'}>
        <div className="container">
          <Link to="/" aria-label="CYBEROS home">
            <Wordmark />
          </Link>
          <nav id="site-nav" className={open ? 'site-nav open' : 'site-nav'} aria-label="Primary">
            {LINKS.map((l) => (
              <NavLink key={l.to} to={l.to}>
                {l.label}
              </NavLink>
            ))}
            <span className="site-nav-cta">
              <Link to="/sign-in" className="btn btn-lg">
                Sign in
              </Link>
              <Link to="/app" className="btn btn-primary btn-lg">
                Open console <ArrowRight size={16} aria-hidden="true" />
              </Link>
            </span>
          </nav>
          <span className="spacer" />
          <ThemeToggle />
          <Link to="/sign-in" className="btn btn-ghost site-cta-desktop">
            Sign in
          </Link>
          <Link to="/app" className="btn btn-primary site-cta-desktop">
            Open console
          </Link>
          <button type="button" className="btn btn-ghost btn-icon menu-btn" aria-expanded={open} aria-controls="site-nav" aria-label={open ? 'Close menu' : 'Open menu'} onClick={() => setOpenAt(open ? null : pathname)}>
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </header>
      {open && <div className="site-scrim" onClick={close} aria-hidden="true" />}
      <main id="main">
        <Outlet />
      </main>
      <footer className="site-footer">
        <div className="container">
          <div className="footer-grid">
            <div className="footer-brand">
              <Wordmark size={26} />
              <p>Authorized security operations, proven. Scope, decision, grant, broker, execution and evidence — one chain of custody.</p>
              <span className="footer-status">
                <ShieldCheck size={14} aria-hidden="true" /> Fail-closed by design
              </span>
            </div>
            <nav className="footer-col" aria-label="Product">
              <span className="footer-title">Product</span>
              {LINKS.map((l) => (
                <Link key={l.to} to={l.to}>
                  {l.label}
                </Link>
              ))}
            </nav>
            <nav className="footer-col" aria-label="Console">
              <span className="footer-title">Console</span>
              <Link to="/app">Open console</Link>
              <Link to="/sign-in">Sign in</Link>
            </nav>
          </div>
          <div className="footer-base">
            <span>© {CURRENT_YEAR} CYBEROS</span>
            <span className="spacer" />
            <span>Decisions, grants, trust and verification are produced by the Go control plane — never by the browser.</span>
          </div>
        </div>
      </footer>
    </>
  )
}
