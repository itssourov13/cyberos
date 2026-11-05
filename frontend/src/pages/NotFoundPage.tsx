import { Link, useLocation } from 'react-router-dom'
import { Empty } from '@/components/ui'

export function NotFoundPage() {
  const { pathname } = useLocation()
  const inApp = pathname.startsWith('/app')
  return (
    <div className="container" style={{ padding: 'var(--s-8) var(--s-5)' }}>
      <Empty title="Page not found">
        <code>{pathname}</code>
      </Empty>
      <div style={{ textAlign: 'center' }}>
        <Link to={inApp ? '/app' : '/'} className="btn btn-primary">
          {inApp ? 'Back to dashboard' : 'Back to home'}
        </Link>
      </div>
    </div>
  )
}
