import { Async, Badge, Digest, Empty, Panel, PanelHead } from '@/components/ui'
import { useApi } from '@/api'
import { relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

/**
 * Read-only slice of the audit stream. Preview stand-in for dedicated
 * decision / grant / execution / provenance endpoints (deferred backend work).
 */
export function AuditSlice({ prefixes, title, empty }: { prefixes?: string[]; title: string; empty: string }) {
  const api = useApi()
  const state = useAsync(() => api.audit(), [api])
  return (
    <Panel>
      <PanelHead title={title} sub="from the audit stream" />
      <Async state={state}>
        {(all) => {
          const rows = prefixes ? all.filter((e) => prefixes.some((p) => e.action.startsWith(p))) : all
          if (rows.length === 0) return <Empty title={empty} />
          return (
            <div className="ledger">
              {rows.map((e) => (
                <div key={e.id} className="ledger-row" style={{ gridTemplateColumns: 'auto minmax(0,1fr) auto auto' }}>
                  <Badge tone={e.tone} dot>
                    {e.action}
                  </Badge>
                  <span className="truncate">
                    <span className="dim">{e.subject}</span> <span className="muted">· {e.actor}</span>
                  </span>
                  <Digest value={e.digest} />
                  <span className="muted nowrap">{relTime(e.at)}</span>
                </div>
              ))}
            </div>
          )
        }}
      </Async>
    </Panel>
  )
}
