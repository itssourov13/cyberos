import { Link } from 'react-router-dom'
import { useApi } from '@/api'
import { Async, Banner, DataTable, Digest, PageHeader, Panel, PanelHead } from '@/components/ui'
import { SeverityBadge } from '@/components/domain/badges'
import { fmtDateTime, relTime } from '@/lib/format'
import { useAsync } from '@/lib/useAsync'

export function EvidencePage() {
  const api = useApi()
  const findings = useAsync(() => api.findings(), [api])
  return (
    <>
      <PageHeader eyebrow="Proof" title="Evidence" description="Evidence behind each finding, identified by the backend-recorded digest of the exact bytes." />
      {api.mode === 'preview' && (
        <Banner tone="warn" title="Sample identifiers, not digests">
          Values starting with sample: are deterministic placeholders from the preview adapter. They are not cryptographic digests and are not evidence.
        </Banner>
      )}
      <Banner tone="info" title="Verification happens on the backend">
        Whether a result is verified is decided by the verifier service. This view displays its verdict and digests; it never computes trust.
      </Banner>
      <Panel>
        <Async state={findings}>
          {(rows) => (
            <DataTable
              caption="Evidence records"
              rows={rows}
              rowKey={(f) => f.id}
              columns={[
                { key: 'digest', header: 'Digest', render: (f) => <Digest value={f.evidenceDigest} /> },
                { key: 'finding', header: 'Supports finding', sort: (f) => f.title, render: (f) => <div className="col" style={{ gap: 0 }}><strong>{f.title}</strong><span className="cell-id">{f.id}</span></div> },
                { key: 'sev', header: 'Severity', render: (f) => <SeverityBadge severity={f.severity} /> },
                { key: 'asm', header: 'Assessment', render: (f) => <span className="cell-id">{f.assessmentId}</span> },
                { key: 'at', header: 'Recorded', sort: (f) => f.foundAt, render: (f) => <span className="muted">{relTime(f.foundAt)}</span> },
              ]}
            />
          )}
        </Async>
      </Panel>
    </>
  )
}

export function ProvenancePage() {
  const api = useApi()
  const state = useAsync(() => api.audit(), [api])
  return (
    <>
      <PageHeader eyebrow="Proof" title="Provenance" description="The chain of custody linking each outcome to the decision, grant and execution that produced it." />
      <Panel>
        <PanelHead title="Chain of custody" sub="oldest first" actions={<Link to="/app/events" className="link">Raw events →</Link>} />
        <Async state={state}>
          {(events) => (
            <div className="panel-body timeline">
              {[...events].sort((a, b) => a.at.localeCompare(b.at)).map((e) => (
                <div key={e.id} className="tl-item">
                  <span className={`tl-dot ${e.tone}`} aria-hidden="true" />
                  <div className="col" style={{ gap: 4 }}>
                    <span className="tl-title mono">{e.action}</span>
                    <span>{e.subject}</span>
                    <span className="tl-meta">{e.actor} · {fmtDateTime(e.at)}</span>
                    {e.digest ? <Digest value={e.digest} /> : null}
                  </div>
                </div>
              ))}
            </div>
          )}
        </Async>
      </Panel>
      <p className="muted">Preview: linked via the audit stream. Real provenance graphs (decision → grant → execution → result) are a deferred backend requirement.</p>
    </>
  )
}
