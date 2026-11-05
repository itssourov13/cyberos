import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Download } from 'lucide-react'
import { useApi } from '@/api'
import type { Asset, Finding, Report, Severity } from '@/api/types'
import { Async, Badge, DataTable, Drawer, KV, Digest, Meter, PageHeader, Panel, PanelHead, Segmented, Banner } from '@/components/ui'
import { CaseStatusBadge, FindingStatusBadge, SeverityBadge } from '@/components/domain/badges'
import { fmtDate, relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

type SeverityFilter = 'all' | Severity
interface RiskRow {
  f: Finding
  asset: Asset | undefined
  score: number
}

const exposureTone = { internet: 'danger', internal: 'info', restricted: 'trust' } as const
const SEV_WEIGHT: Record<Severity, number> = { critical: 100, high: 75, medium: 45, low: 20, info: 5 }
const EXPO_WEIGHT = { internet: 1, internal: 0.7, restricted: 0.5 } as const

export function AssetsPage() {
  const api = useApi()
  const state = useAsync(() => api.assets(), [api])
  return (
    <>
      <PageHeader eyebrow="Intelligence" title="Assets" description="Everything discovered inside your authorised scopes, with exposure and risk." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Asset>
              caption="Assets"
              rows={rows}
              rowKey={(a) => a.id}
              columns={[
                { key: 'name', header: 'Asset', sort: (a) => a.name, render: (a) => <strong className="mono">{a.name}</strong> },
                { key: 'kind', header: 'Kind', sort: (a) => a.kind, render: (a) => <span className="dim">{a.kind}</span> },
                { key: 'exp', header: 'Exposure', sort: (a) => a.exposure, render: (a) => <Badge tone={exposureTone[a.exposure]} dot>{a.exposure}</Badge> },
                { key: 'risk', header: 'Risk', sort: (a) => a.riskScore, render: (a) => <div className="row" style={{ gap: 10, minWidth: 140 }}><span className="score" style={{ width: 28 }}>{a.riskScore}</span><div className="grow"><Meter value={a.riskScore / 100} tone={a.riskScore >= 70 ? 'danger' : a.riskScore >= 40 ? 'warn' : undefined} /></div></div> },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}

export function FindingsPage() {
  const api = useApi()
  const state = useAsync(() => api.findings(), [api])
  const [sev, setSev] = useState<SeverityFilter>('all')
  const [sel, setSel] = useState<Finding | undefined>(undefined)
  return (
    <>
      <PageHeader eyebrow="Intelligence" title="Findings" description="Normalised results, each anchored to the evidence digest that proves it." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <>
              <div className="toolbar">
                <Segmented<SeverityFilter> label="Severity filter" value={sev} onChange={(v) => setSev(v)} options={[{ value: 'all', label: 'All' }, { value: 'critical', label: 'Critical' }, { value: 'high', label: 'High' }, { value: 'medium', label: 'Medium' }, { value: 'low', label: 'Low' }]} />
              </div>
              <DataTable<Finding>
                caption="Findings"
                rows={sev === 'all' ? rows : rows.filter((f) => f.severity === sev)}
                rowKey={(f) => f.id}
                onRowClick={(f) => setSel(f)}
                selectedKey={sel?.id}
                columns={[
                  { key: 'sev', header: 'Severity', sort: (f) => SEV_WEIGHT[f.severity], render: (f) => <SeverityBadge severity={f.severity} /> },
                  { key: 'title', header: 'Finding', sort: (f) => f.title, render: (f) => <strong>{f.title}</strong> },
                  { key: 'asset', header: 'Asset', render: (f) => <span className="cell-id">{f.assetId}</span> },
                  { key: 'status', header: 'Status', sort: (f) => f.status, render: (f) => <FindingStatusBadge status={f.status} /> },
                  { key: 'found', header: 'Found', sort: (f) => f.foundAt, render: (f) => <span className="muted">{relTime(f.foundAt)}</span> },
                ]}
              />
            </>
          )}
        </Async>
      </Panel>
      <Drawer open={!!sel} title={sel?.title ?? ''} onClose={() => setSel(undefined)}>
        {sel && (
          <>
            <KV items={[['ID', <span key="ID" className="mono">{sel.id}</span>], ['Severity', <SeverityBadge key="Severity" severity={sel.severity} />], ['Status', <FindingStatusBadge key="Status" status={sel.status} />], ['Asset', <span key="Asset" className="mono">{sel.assetId}</span>], ['Assessment', <span key="Assessment" className="mono">{sel.assessmentId}</span>], ['Found', relTime(sel.foundAt)], ['Evidence', <Digest key="Evidence" value={sel.evidenceDigest} />]]} />
            <Link to="/app/evidence" className="link">Open evidence →</Link>
          </>
        )}
      </Drawer>
    </>
  )
}

/** Display-only ranking for preview; the authoritative risk model lives in the backend. */
export function RiskPage() {
  const api = useApi()
  const findings = useAsync(() => api.findings(), [api])
  const assets = useAsync(() => api.assets(), [api])
  const ranked = useMemo<RiskRow[]>(() => {
    const a = new Map((assets.data ?? []).map((x) => [x.id, x]))
    return (findings.data ?? [])
      .filter((f) => f.status === 'open' || f.status === 'triaged')
      .map((f) => ({ f, asset: a.get(f.assetId), score: Math.round(SEV_WEIGHT[f.severity] * EXPO_WEIGHT[a.get(f.assetId)?.exposure ?? 'internal']) }))
      .sort((x, y) => y.score - x.score)
  }, [findings.data, assets.data])
  return (
    <>
      <PageHeader eyebrow="Intelligence" title="Risk prioritisation" description="Open findings ranked by severity and asset exposure — your remediation queue." />
      <Banner tone="info" title="Preview ranking">This ordering is computed locally from sample data for illustration. The authoritative risk model runs in the backend.</Banner>
      <Panel>
        <Async state={findings}>
          {() => (
            <DataTable<RiskRow>
              caption="Risk queue"
              rows={ranked}
              rowKey={(r) => r.f.id}
              emptyTitle="Nothing open — queue is clear"
              columns={[
                { key: 'rank', header: '#', render: (r) => <span className="mono muted">{ranked.indexOf(r) + 1}</span> },
                { key: 'score', header: 'Priority', sort: (r) => r.score, render: (r) => <span className="score" style={{ color: r.score >= 70 ? 'var(--danger)' : r.score >= 40 ? 'var(--warn)' : 'var(--text-2)' }}>{r.score}</span> },
                { key: 'title', header: 'Finding', render: (r) => <strong>{r.f.title}</strong> },
                { key: 'sev', header: 'Severity', render: (r) => <SeverityBadge severity={r.f.severity} /> },
                { key: 'asset', header: 'Asset', render: (r) => <span className="mono dim">{r.asset?.name ?? r.f.assetId}</span> },
                { key: 'exp', header: 'Exposure', render: (r) => (r.asset ? <Badge tone={exposureTone[r.asset.exposure]}>{r.asset.exposure}</Badge> : '—') },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}

export function CasesPage() {
  const api = useApi()
  const state = useAsync(() => api.cases(), [api])
  const cols = [['open', 'Open'], ['in-progress', 'In progress'], ['resolved', 'Resolved']] as const
  return (
    <>
      <PageHeader eyebrow="Intelligence" title="Cases" description="Remediation work grouped from findings and tracked to resolution." />
      <Async state={state}>
        {(rows) => (
          <div className="grid grid-3">
            {cols.map(([key, label]) => (
              <Panel key={key}>
                <PanelHead title={label} sub={String(rows.filter((c) => c.status === key).length)} />
                <div className="panel-body col">
                  {rows.filter((c) => c.status === key).map((c) => (
                    <article key={c.id} className="card" style={{ padding: 'var(--s-4)' }}>
                      <div className="row"><SeverityBadge severity={c.severity} /><span className="spacer" /><span className="cell-id">{c.id}</span></div>
                      <strong>{c.title}</strong>
                      <div className="row"><span className="muted">Owner</span><span>{c.owner}</span><span className="spacer" /><CaseStatusBadge status={c.status} /></div>
                      <span className="muted">{c.findingIds.length} linked finding{c.findingIds.length === 1 ? '' : 's'}</span>
                    </article>
                  ))}
                  {rows.every((c) => c.status !== key) && <span className="muted">No cases</span>}
                </div>
              </Panel>
            ))}
          </div>
        )}
      </Async>
    </>
  )
}

export function ReportsPage() {
  const api = useApi()
  const state = useAsync(() => api.reports(), [api])
  return (
    <>
      <PageHeader eyebrow="Intelligence" title="Reports" description="Reports are generated by the backend from verified evidence." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Report>
              caption="Reports"
              rows={rows}
              rowKey={(r) => r.id}
              columns={[
                { key: 'title', header: 'Report', sort: (r) => r.title, render: (r) => <strong>{r.title}</strong> },
                { key: 'project', header: 'Project', render: (r) => <Link className="link mono" to={`/app/projects/${r.projectId}`}>{r.projectId}</Link> },
                { key: 'gen', header: 'Generated', sort: (r) => r.generatedAt, render: (r) => <span className="muted">{fmtDate(r.generatedAt)}</span> },
                { key: 'fmt', header: 'Formats', render: (r) => <div className="chips">{r.formats.map((f) => <span key={f} className="chip">{f}</span>)}</div> },
                { key: 'dl', header: '', align: 'right', render: () => <button type="button" className="btn btn-sm" disabled title="Downloads arrive with backend integration"><Download size={13} /> Download</button> },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}
