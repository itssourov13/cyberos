import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { Banner, Skeleton } from '@/components/ui'
import { useSession } from '@/app/providers/session-context'

/**
 * UX gate only. Real protection is the backend: every `/api/v1` call is
 * authenticated and authorised server-side regardless of what this renders.
 */
export function RequireSession({ children }: { children: ReactNode }) {
  const session = useSession()
  const location = useLocation()

  if (session.status === 'loading') {
    return (
      <div className="content" aria-busy="true">
        <Skeleton h={30} w="30%" />
        <Skeleton h={240} />
      </div>
    )
  }
  if (session.status === 'unavailable') {
    return (
      <div className="content">
        <Banner tone="danger" title="Cannot reach the CYBEROS API">
          {session.error ?? 'The backend did not respond.'}{' '}
          <button type="button" className="link" onClick={session.retry}>
            Try again
          </button>
        </Banner>
      </div>
    )
  }
  if (session.status === 'anonymous') {
    return <Navigate to="/sign-in" replace state={{ from: location.pathname + location.search, expired: session.expired }} />
  }
  return <>{children}</>
}
