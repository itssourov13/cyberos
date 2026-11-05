import { useEffect, useState } from 'react'
import { Play, ShieldAlert } from 'lucide-react'
import { useApi } from '@/api'
import type { SpineTrace } from '@/api/types'
import { Async, Badge, Banner, PageHeader, Panel, PanelHead } from '@/components/ui'
import { AuditSlice } from '@/components/domain/AuditSlice'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { STAGE_ORDER } from '@/components/domain/spine-stages'
import { SPINE_ALL_PENDING } from '@/lib/spine'
import { useAsync } from '@/lib/useAsync'
import { useToast } from '@/app/providers/toast-context'

export function EvaluatePage() {
  const api = useApi()
  const toast = useToast()
  const projects = useAsync(() => api.projects(), [api])
  const scopes = useAsync(() => api.scopes(), [api])
  const packs = useAsync(() => api.packs(), [api])

  const [projectId, setProjectId] = useState('prj-atlas')
  const [scopeId, setScopeId] = useState('scp-atlas-web')
  const [capability, setCapability] = useState('recon.tls.scan')
  const [target, setTarget] = useState('portal.northwind.example')
  const [busy, setBusy] = useState(false)
  const [trace, setTrace] = useState<SpineTrace>()
  const [revealed, setRevealed] = useState(0)

  const projectScopes = (scopes.data ?? []).filter((s) => s.projectId === projectId)
  const capabilities = [...new Set((packs.data ?? []).flatMap((p) => p.capabilities))].sort()

  useEffect(() => {
    if (!trace || revealed >= trace.stages.length) return
    const t = window.setTimeout(() => setRevealed((r) => r + 1), 420)
    return () => window.clearTimeout(t)
  }, [trace, revealed])

  async function submit() {
    setBusy(true)
    setTrace(undefined)
    setRevealed(0)
    try {
      const t = await api.submitSpineRequest({ principal: 'operator:preview', projectId, scopeId, capability, target })
      setTrace(t)
    } catch (e) {
      toast.push(e instanceof Error ? e.message : 'Request failed', 'danger')
    } finally {
      setBusy(false)
    }
  }

  const shown = trace ? trace.stages.slice(0, revealed) : []
  const finished = trace && revealed >= trace.stages.length

  return (
    <>
      <PageHeader eyebrow="Control plane" title="Evaluate a request" description="Submit a request to the control plane and watch it travel the spine. The browser asks; the backend decides." />
      <div className="authority">
        <ShieldAlert size={16} aria-hidden="true" />
        <span>
          <strong>Authority is server-side.</strong> This form only <em>submits</em> a request. Decisions, grants and execution are produced and verified by the Go backend.
        </span>
      </div>

      <div className="grid grid-main">
        <Panel>
          <PanelHead title="Request" />
          <div className="panel-body col">
            <Async state={projects} rows={2}>
              {(ps) => (
                <div className="grid grid-2">
                  <div className="field">
                    <label htmlFor="ev-project">Project</label>
                    <select id="ev-project" className="select" value={projectId} onChange={(e) => { setProjectId(e.target.value); setScopeId(scopes.data?.find((s) => s.projectId === e.target.value)?.id ?? '') }}>
                      {ps.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="ev-scope">Scope</label>
                    <select id="ev-scope" className="select" value={scopeId} onChange={(e) => setScopeId(e.target.value)}>
                      {projectScopes.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.status})</option>)}
                      {projectScopes.length === 0 && <option value="">No scopes</option>}
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="ev-cap">Capability</label>
                    <select id="ev-cap" className="select" value={capability} onChange={(e) => setCapability(e.target.value)}>
                      {capabilities.map((c) => <option key={c}>{c}</option>)}
                    </select>
                    <span className="hint">Try auth.spray (untrusted pack) or an expired scope to see a denial.</span>
                  </div>
                  <div className="field">
                    <label htmlFor="ev-target">Target</label>
                    <input id="ev-target" className="input" value={target} onChange={(e) => setTarget(e.target.value)} spellCheck={false} />
                    <span className="hint">Must match a target listed in the selected scope; anything else is denied.</span>
                  </div>
                </div>
              )}
            </Async>
            <div>
              <button type="button" className="btn btn-primary" onClick={submit} disabled={busy || !scopeId || !target.trim()}>
                <Play size={14} /> {busy ? 'Submitting…' : 'Submit to control plane'}
              </button>
            </div>
          </div>
        </Panel>

        <Panel>
          <PanelHead title="Outcome" actions={finished && trace && <Badge tone={trace.final === 'completed' ? 'ok' : 'danger'} dot>{trace.final}</Badge>} />
          <div className="panel-body col">
            <SpineFlow stages={trace ? shown : SPINE_ALL_PENDING} activeIndex={trace && !finished ? revealed : undefined} compact />
            {!trace && <p className="dim">{busy ? 'Waiting for the control plane…' : 'Submit a request to see each stage’s verdict.'}</p>}
            {trace?.simulated && <Banner tone="warn" title="Simulated trace">Produced by the preview adapter. A live backend returns the authoritative trace.</Banner>}
          </div>
        </Panel>
      </div>

      {trace && shown.length > 0 && (
        <Panel>
          <PanelHead title="Stage detail" sub={trace.requestId} />
          <div className="panel-body pipe-detail">
            {shown.map((s) => (
              <div key={s.id} className="pipe-row" data-state={s.outcome}>
                <strong>{STAGE_ORDER.find((d) => d.id === s.id)?.label}</strong>
                <div className="col" style={{ gap: 2 }}>
                  <span>{s.detail}</span>
                  {s.ref && <span className="mono muted">ref {s.ref}</span>}
                </div>
              </div>
            ))}
          </div>
        </Panel>
      )}
    </>
  )
}

export function DecisionsPage() {
  return (
    <>
      <PageHeader eyebrow="Control plane" title="Decisions" description="Every allow or deny the control plane has issued, with the reason recorded." />
      <AuditSlice title="Decision log" prefixes={['decision.']} empty="No decisions recorded" />
      <p className="muted">Preview: derived from the audit stream. A dedicated decisions endpoint (with policy rationale) is a deferred backend requirement.</p>
    </>
  )
}

export function GrantsPage() {
  return (
    <>
      <PageHeader eyebrow="Control plane" title="Execution grants" description="Short-lived grants bound to principal, capability, target and pack digest. Issued only by the control plane." />
      <div className="authority">
        <ShieldAlert size={16} aria-hidden="true" />
        <span>There is deliberately no “issue grant” control in this console. Grants cannot be created, extended or edited from the browser.</span>
      </div>
      <AuditSlice title="Grant ledger" prefixes={['grant.']} empty="No grants issued" />
    </>
  )
}

export function ExecutionPage() {
  return (
    <>
      <PageHeader eyebrow="Control plane" title="Execution" description="Runs dispatched by the tool broker. The browser can observe, never start, a run." />
      <AuditSlice title="Execution activity" prefixes={['execution.', 'result.']} empty="No executions recorded" />
    </>
  )
}
