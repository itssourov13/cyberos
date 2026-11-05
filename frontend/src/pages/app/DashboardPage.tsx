import { Link } from 'react-router-dom'
import { useApi } from '@/api'
import type { Severity } from '@/api/types'
import { Async, Bars, Badge, Meter, PageHeader, Panel, PanelHead, Stat } from '@/components/ui'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { SPINE_ALL_PASS } from '@/lib/spine'
import { relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

const SEV_ORDER: Severity[] = ['critical', 'high', 'medium', 'low', 'info']
const SEV_VAR: Record<Severity, string> = { critical: 'danger', high: 'danger', medium: 'warn', low: 'info', info: 'text-3' }

export function DashboardPage() {
  const api = useApi()
  const dash = useAsync(() => api.dashboard(), [api])
  const audit = useAsync(() => api.audit(), [api])

  return (
    <>
      <PageHeader eyebrow="Overview" title="Operations dashboard" description="Posture of your engagements, the control plane and the worker fleet." />

      <Async state={dash} rows={3}>
        {(d) => {
          const totalSev = SEV_ORDER.reduce((n, s) => n + d.severityMix[s], 0) || 1
          return (
            <>
              <div className="grid grid-4">
                <Stat label="Active scopes" value={d.activeScopes} delta={`${d.projects} projects`} />
                <Stat label="Open findings" value={d.openFindings} tone={d.openFindings ? 'warn' : 'ok'} delta="open + triaged" />
                <Stat label="Grants issued · 24h" value={d.grantsIssued24h} tone="grant" spark={d.executionsPerHour.slice(-12)} />
                <Stat label="Denials · 24h" value={d.denials24h} tone="danger" delta="fail-closed outcomes" />
              </div>

              <div className="grid grid-main">
                <Panel>
                  <PanelHead title="Executions per hour" sub="last 24h" />
                  <div className="panel-body">
                    <Bars values={d.executionsPerHour} />
                  </div>
                </Panel>
                <Panel>
                  <PanelHead title="Fleet" sub="workers online" />
                  <div className="panel-body col">
                    <div className="row">
                      <span style={{ fontFamily: 'var(--font-display)', fontSize: 30, fontWeight: 600 }}>
                        {d.workersOnline}/{d.workersTotal}
                      </span>
                      <Link to="/app/workers" className="link spacer right">
                        View workers →
                      </Link>
                    </div>
                    <Meter value={d.workersTotal ? d.workersOnline / d.workersTotal : 0} tone={d.workersOnline < d.workersTotal / 2 ? 'warn' : undefined} />
                    <hr className="hr" style={{ margin: '4px 0' }} />
                    <span className="field-label">Findings by severity</span>
                    <div className="col" style={{ gap: 8 }}>
                      {SEV_ORDER.map((s) => (
                        <div key={s} className="row" style={{ gap: 10 }}>
                          <span className="mono muted" style={{ width: 64 }}>
                            {s}
                          </span>
                          <div className="grow">
                            <Meter value={d.severityMix[s] / totalSev} tone={s === 'critical' || s === 'high' ? 'danger' : s === 'medium' ? 'warn' : undefined} />
                          </div>
                          <span className="mono" style={{ width: 20, textAlign: 'right', color: `var(--${SEV_VAR[s]})` }}>
                            {d.severityMix[s]}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </Panel>
              </div>
            </>
          )
        }}
      </Async>

      <div className="grid grid-main">
        <Panel>
          <PanelHead title="Recent audit events" actions={<Link to="/app/events" className="link">All events →</Link>} />
          <Async state={audit}>
            {(events) => (
              <div className="ledger">
                {events.slice(0, 6).map((e) => (
                  <div key={e.id} className="ledger-row">
                    <Badge tone={e.tone} dot>
                      {e.action.split('.')[0]}
                    </Badge>
                    <span className="truncate">
                      <span className="mono">{e.action}</span> <span className="dim">· {e.subject}</span>
                    </span>
                    <span className="muted nowrap">{relTime(e.at)}</span>
                  </div>
                ))}
              </div>
            )}
          </Async>
        </Panel>
        <Panel>
          <PanelHead title="The spine" sub="every request" />
          <div className="panel-body col">
            <SpineFlow stages={SPINE_ALL_PASS} compact />
            <p className="dim">A request that fails any link stops there. Try one in Evaluate to see it end to end.</p>
            <Link to="/app/evaluate" className="btn btn-primary">
              Evaluate a request
            </Link>
          </div>
        </Panel>
      </div>
    </>
  )
}
