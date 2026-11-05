import { Check, X } from 'lucide-react'
import { useApi } from '@/api'
import { useTheme } from '@/app/providers/theme-context'
import { Async, Badge, Banner, KV, PageHeader, Panel, PanelHead, Segmented } from '@/components/ui'
import { AuditSlice } from '@/components/domain/AuditSlice'
import { useAsync } from '@/lib/useAsync'

const AUTONOMY_STAGES = [
  ['Observe', 'Events and outbox items are delivered to the autonomy processor.'],
  ['Plan', 'The planner proposes candidate work. A proposal is only a suggestion.'],
  ['Policy eligibility', 'Policy decides whether a proposal may even be considered.'],
  ['Hand off', 'Eligible work is handed to the Orchestrator, which re-enters the normal spine.'],
]

export function AutonomyPage() {
  return (
    <>
      <PageHeader eyebrow="Platform" title="Controlled autonomy" description="Optional AI assistance that can observe and propose — never authorise, grant or execute." />
      <Banner tone="info" title="No autonomy authority">
        Autonomous proposals travel the same Scope → Decision → Grant → Broker spine as a human request. Nothing in this console can enable, disable or widen autonomy; that is a backend policy concern.
      </Banner>
      <ol className="steps">
        {AUTONOMY_STAGES.map(([t, d]) => (
          <li key={t} className="step">
            <div>
              <h3>{t}</h3>
              <p>{d}</p>
            </div>
          </li>
        ))}
      </ol>
      <p className="muted">Preview: autonomy status, proposals and policy eligibility endpoints are deferred backend requirements.</p>
    </>
  )
}

export function EventsPage() {
  return (
    <>
      <PageHeader eyebrow="Platform" title="Events & audit" description="The append-only record of decisions, grants, executions and administrative actions." />
      <AuditSlice title="Audit trail" empty="No events recorded" />
    </>
  )
}

export function SystemPage() {
  const api = useApi()
  const state = useAsync(() => api.system(), [api])
  return (
    <>
      <PageHeader eyebrow="Platform" title="System health" description="Readiness of the API and its security dependencies." />
      <Async state={state}>
        {(s) => (
          <>
            <Panel>
              <PanelHead title="Build" actions={<Badge tone={s.status === 'ok' ? 'ok' : 'warn'} dot>{s.status}</Badge>} />
              <div className="panel-body">
                <KV items={[['Version', <span key="Version" className="mono">{s.version}</span>], ['Commit', <span key="Commit" className="mono">{s.commit}</span>], ['Data source', s.mode === 'preview' ? 'Preview adapter (sample data)' : 'CYBEROS API']]} />
              </div>
            </Panel>
            <Panel>
              <PanelHead title="Checks" />
              <div className="ledger">
                {s.checks.map((c) => (
                  <div key={c.name} className="ledger-row">
                    {c.ok ? <Check size={16} color="var(--ok)" aria-label="passing" /> : <X size={16} color="var(--warn)" aria-label="not passing" />}
                    <div className="col" style={{ gap: 0 }}>
                      <strong>{c.name}</strong>
                      <span className="muted">{c.note}</span>
                    </div>
                    <Badge tone={c.ok ? 'ok' : 'warn'}>{c.ok ? 'ok' : 'unknown'}</Badge>
                  </div>
                ))}
              </div>
            </Panel>
          </>
        )}
      </Async>
    </>
  )
}

export function SettingsPage() {
  const { theme, toggle } = useTheme()
  const api = useApi()
  return (
    <>
      <PageHeader eyebrow="Platform" title="Settings" description="Preferences for this browser. Organisation and security policy are managed on the backend." />
      <Panel>
        <PanelHead title="Appearance" />
        <div className="panel-body col">
          <div className="row">
            <div className="col" style={{ gap: 0 }}>
              <strong>Theme</strong>
              <span className="muted">Stored in this browser only.</span>
            </div>
            <span className="spacer" />
            <Segmented label="Theme" value={theme} onChange={(v) => v !== theme && toggle()} options={[{ value: 'dark', label: 'Dark' }, { value: 'light', label: 'Light' }]} />
          </div>
          <span className="muted">Animation automatically reduces when your system requests reduced motion.</span>
        </div>
      </Panel>
      <Panel>
        <PanelHead title="Connection" />
        <div className="panel-body">
          <KV items={[['Mode', <Badge key="Mode" tone={api.mode === 'live' ? 'ok' : 'warn'} dot>{api.mode}</Badge>], ['API', api.mode === 'live' ? '/api/v1' : 'Not connected']]} />
        </div>
      </Panel>
    </>
  )
}
