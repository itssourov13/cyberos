import { useEffect, useLayoutEffect } from 'react'
import { useLocation } from 'react-router-dom'

const prefersReducedMotion = () => typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches

/**
 * Scroll-reveal. Elements marked `data-reveal` fade/rise into view once.
 * Children of a `data-stagger` container are revealed in sequence.
 * Pages are lazy-loaded, so the DOM is re-scanned as it changes. Content is only
 * hidden once this hook has run, so a failure here never hides the page.
 */
export function useReveal() {
  const { pathname } = useLocation()
  useLayoutEffect(() => {
    const root = document.documentElement
    if (prefersReducedMotion() || typeof IntersectionObserver === 'undefined') {
      root.classList.remove('fx')
      return
    }
    root.classList.add('fx')

    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            ;(e.target as HTMLElement).dataset.in = ''
            io.unobserve(e.target)
          }
        }
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.08 },
    )

    const scan = () => {
      document.querySelectorAll<HTMLElement>('[data-stagger]:not([data-staggered])').forEach((group) => {
        group.dataset.staggered = ''
        Array.from(group.children).forEach((child, i) => {
          const el = child as HTMLElement
          el.dataset.reveal = ''
          el.style.setProperty('--d', `${Math.min(i, 8) * 70}ms`)
        })
      })
      document.querySelectorAll<HTMLElement>('[data-reveal]:not([data-in]):not([data-watched])').forEach((el) => {
        el.dataset.watched = ''
        io.observe(el)
      })
    }

    // Runs as a microtask after DOM insertion, i.e. before the browser paints new nodes.
    const mo = new MutationObserver(scan)
    scan()
    mo.observe(document.body, { childList: true, subtree: true })
    return () => {
      mo.disconnect()
      io.disconnect()
      // Let a remount (StrictMode, route change) re-observe anything not yet revealed.
      document.querySelectorAll<HTMLElement>('[data-watched]:not([data-in])').forEach((el) => {
        delete el.dataset.watched
      })
    }
  }, [pathname])
}

/** Pointer-tracked spotlight for cards: writes --mx / --my on the hovered surface. */
export function useSpotlight() {
  useEffect(() => {
    if (prefersReducedMotion()) return
    let frame = 0
    const onMove = (ev: PointerEvent) => {
      const target = (ev.target as HTMLElement | null)?.closest<HTMLElement>('.card, .stat, .step, .distinct > div')
      if (!target) return
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        const r = target.getBoundingClientRect()
        target.style.setProperty('--mx', `${ev.clientX - r.left}px`)
        target.style.setProperty('--my', `${ev.clientY - r.top}px`)
      })
    }
    document.addEventListener('pointermove', onMove, { passive: true })
    return () => {
      document.removeEventListener('pointermove', onMove)
      cancelAnimationFrame(frame)
    }
  }, [])
}

/** Tracks window scroll: whether the page has left the top, and 0..1 reading progress. */
export function useScrollState(target: () => HTMLElement | null) {
  useEffect(() => {
    let frame = 0
    const update = () => {
      frame = 0
      const el = target()
      if (!el) return
      const max = document.documentElement.scrollHeight - window.innerHeight
      const progress = max > 0 ? Math.min(1, Math.max(0, window.scrollY / max)) : 0
      el.dataset.scrolled = window.scrollY > 8 ? 'true' : 'false'
      el.style.setProperty('--progress', progress.toFixed(4))
    }
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(update)
    }
    update()
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll, { passive: true })
    return () => {
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
      if (frame) cancelAnimationFrame(frame)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
}

/** Closes an overlay on Escape and locks page scroll while it is open. */
export function useOverlay(open: boolean, close: () => void, lockScroll = true) {
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    document.addEventListener('keydown', onKey)
    const prev = document.documentElement.style.overflow
    if (lockScroll) document.documentElement.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.documentElement.style.overflow = prev
    }
  }, [open, close, lockScroll])
}
