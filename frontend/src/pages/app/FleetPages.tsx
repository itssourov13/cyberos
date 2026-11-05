import { useState } from 'react'
import { useApi } from '@/api'
import type { Assignment, Pack, Worker } from '@/api/types'
import { Async, Badge, Banner, DataTable, Digest, Drawer, KV, Meter, PageHeader, Panel, PanelHead } from '@/components/ui'
import { AssignmentStateBadge, SignatureBadge, TrustBadge, WorkerStateBadge } from '@/components/domain/badges'
import { fmtDateTime, pct, relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

export function WorkersPage() {
  const api = useApi()
  const state = useAsync(() => api.workers(), [api])
  const [sel, setSel] = useState<Worker | undefined>(undefined)
  return (
    <>
      <PageHeader eyebrow="Fleet" title="Workers" description="Remote workers attested over mTLS. A worker only runs what a leased assignment allows." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Worker>
              caption="Workers"
              rows={rows}
              rowKey={(w) => w.id}
              onRowClick={(w) => setSel(w)}
              selectedKey={sel?.id}
              columns={[
                { key: 'name', header: 'Worker', sort: (w) => w.name, render: (w) => <strong className="mono">{w.name}</strong> },
                { key: 'state', header: 'State', sort: (w) => w.state, render: (w) => <WorkerStateBadge state={w.state} /> },
                { key: 'load', header: 'Load', sort: (w) => w.load, render: (w) => <div className="row" style={{ minWidth: 130, gap: 8 }}><span className="mono" style={{ width: 36 }}>{pct(w.load)}</span><div className="grow"><Meter value={w.load} tone={w.load > 0.85 ? 'danger' : w.load > 0.65 ? 'warn' : undefined} /></div></div> },
                { key: 'caps', header: 'Capabilities', render: (w) => <span className="dim">{w.capabilities.length}</span> },
                { key: 'hb', header: 'Heartbeat', sort: (w) => w.lastHeartbeat, render: (w) => <span className="muted">{relTime(w.lastHeartbeat)}</span> },
              ]}
            />
          )}
        </Async>
      </Panel>
      <Drawer open={!!sel} title={sel?.name ?? ''} onClose={() => setSel(undefined)}>
        {sel && (
          <>
            <KV items={[['State', <WorkerStateBadge key="State" state={sel.state} />], ['Project', <span key="Project" className="mono">{sel.projectId}</span>], ['Identity', <Digest key="Identity" value={sel.identityFingerprint} />], ['Last heartbeat', fmtDateTime(sel.lastHeartbeat)], ['Load', pct(sel.load)]]} />
            <div className="col" style={{ gap: 8 }}>
              <span className="field-label">Capabilities</span>
              <div className="chips">{sel.capabilities.length ? sel.capabilities.map((c) => <span key={c} className="chip">{c}</span>) : <span className="muted">None</span>}</div>
            </div>
            <Banner tone="info">Suspending or revoking a worker is an administrative action recorded by the backend. It arrives with API integration.</Banner>
          </>
        )}
      </Drawer>
    </>
  )
}

export function AssignmentsPage() {
  const api = useApi()
  const state = useAsync(() => api.assignments(), [api])
  return (
    <>
      <PageHeader eyebrow="Fleet" title="Assignments" description="Leased units of work. A result only counts once the backend has verified its signature." />
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Assignment>
              caption="Assignments"
              rows={rows}
              rowKey={(a) => a.id}
              columns={[
                { key: 'id', header: 'Assignment', render: (a) => <span className="cell-id">{a.id}</span> },
                { key: 'task', header: 'Execution', render: (a) => <span className="cell-id">{a.taskId}</span> },
                { key: 'worker', header: 'Worker', render: (a) => <span className="cell-id">{a.workerId}</span> },
                { key: 'state', header: 'State', sort: (a) => a.state, render: (a) => <AssignmentStateBadge state={a.state} /> },
                { key: 'lease', header: 'Lease', sort: (a) => a.leaseExpires, render: (a) => <span className="muted">{relTime(a.leaseExpires)}</span> },
                { key: 'ver', header: 'Result', render: (a) => <Badge tone={a.resultVerified ? 'evidence' : 'neutral'} dot>{a.resultVerified ? 'verified by backend' : 'unverified'}</Badge> },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}

const LAYERS: [string, string][] = [
  ['Integrity', 'Does the content match its digest?'],
  ['Signature', 'Does a known key vouch for it?'],
  ['Trust', 'Has an administrator chosen to rely on this publisher?'],
  ['Authorization', 'May this principal use it here and now?'],
  ['Execution', 'Only the broker, only with a grant.'],
]

export function PacksPage() {
  const api = useApi()
  const state = useAsync(() => api.packs(), [api])
  const [sel, setSel] = useState<Pack | undefined>(undefined)
  return (
    <>
      <PageHeader eyebrow="Supply chain" title="Tool packs & trust" description="Each pack’s integrity, signature and trust are separate, independently recorded states." />
      <Panel>
        <PanelHead title="Five separate questions" />
        <div className="panel-body">
          <div className="distinct">
            {LAYERS.map(([k, v]) => (
              <div key={k}>
                <strong>{k}</strong>
                <span>{v}</span>
              </div>
            ))}
          </div>
        </div>
      </Panel>
      <Panel>
        <Async state={state}>
          {(rows) => (
            <DataTable<Pack>
              caption="Tool packs"
              rows={rows}
              rowKey={(p) => p.id}
              onRowClick={(p) => setSel(p)}
              selectedKey={sel?.id}
              columns={[
                { key: 'name', header: 'Pack', sort: (p) => p.name, render: (p) => <div className="col" style={{ gap: 0 }}><strong>{p.name}</strong><span className="muted">{p.publisher}</span></div> },
                { key: 'ver', header: 'Version', render: (p) => <span className="mono">{p.version}</span> },
                { key: 'sig', header: 'Signature', sort: (p) => p.signature, render: (p) => <SignatureBadge signature={p.signature} /> },
                { key: 'trust', header: 'Trust', sort: (p) => p.trust, render: (p) => <TrustBadge trust={p.trust} /> },
                { key: 'caps', header: 'Capabilities', render: (p) => <span className="dim">{p.capabilities.length}</span> },
              ]}
            />
          )}
        </Async>
      </Panel>
      <Drawer open={!!sel} title={sel ? `${sel.name} ${sel.version}` : ''} onClose={() => setSel(undefined)}>
        {sel && (
          <>
            <KV items={[['Publisher', sel.publisher], ['Digest', <Digest key="Digest" value={sel.digest} />], ['Signature', <SignatureBadge key="Signature" signature={sel.signature} />], ['Trust', <TrustBadge key="Trust" trust={sel.trust} />]]} />
            <div className="col" style={{ gap: 8 }}>
              <span className="field-label">Capabilities provided</span>
              <div className="chips">{sel.capabilities.map((c) => <span key={c} className="chip">{c}</span>)}</div>
            </div>
            <div className="authority">
              <span>A valid signature does not make a pack trusted, and trust does not authorise a run. Trust changes are administrative acts made on the backend — never from this browser.</span>
            </div>
          </>
        )}
      </Drawer>
    </>
  )
}
