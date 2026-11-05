import type { ReactNode } from 'react'
import type { AsyncState } from '@/lib/useAsync'
import { Banner, Skeleton } from './primitives'

/** Renders loading / error / data states for a useAsync result. Fails visibly, never silently. */
export function Async<T>({ state, children, rows = 4 }: { state: AsyncState<T>; children: (data: T) => ReactNode; rows?: number }) {
  if (state.error) {
    return (
      <div className="panel-body col">
        <Banner tone="danger" title="Could not load this view">
          {state.error.message}
        </Banner>
        <div>
          <button type="button" className="btn btn-sm" onClick={state.reload}>
            Retry
          </button>
        </div>
      </div>
    )
  }
  if (state.data === undefined) {
    return (
      <div className="panel-body col" aria-busy="true" aria-live="polite">
        {Array.from({ length: rows }, (_, i) => (
          <Skeleton key={i} h={18} w={`${92 - i * 9}%`} />
        ))}
        <span className="sr-only">Loading…</span>
      </div>
    )
  }
  return <>{children(state.data)}</>
}
