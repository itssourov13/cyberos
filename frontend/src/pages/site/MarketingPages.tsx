import { Link } from 'react-router-dom'
import { Bot, Briefcase, Bug, Cpu, FileText, Gauge, KeyRound, Package, ScrollText, Server } from 'lucide-react'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { STAGE_ORDER } from '@/components/domain/spine-stages'
import { SPINE_ALL_PASS } from '@/lib/spine'

const AREAS = [
  { icon: Briefcase, title: 'Engagements', body: 'Projects, scopes, tasks and assessments with explicit authorisation windows.', tone: 'ok' },
  { icon: Bug, title: 'Findings & assets', body: 'An asset graph and normalised findings, each tied to the evidence that proves it.', tone: 'danger' },
  { icon: Gauge, title: 'Risk prioritisation', body: 'Exposure and severity combine into a ranked queue operators can act on.', tone: 'grant' },
  { icon: FileText, title: 'Cases & reports', body: 'Turn findings into cases, track remediation and publish reports from verified data.', tone: 'info' },
  { icon: KeyRound, title: 'Control plane', body: 'Evaluation, decisions and execution grants — authoritative, server-side, audited.', tone: 'grant' },
  { icon: Cpu, title: 'Tool broker', body: 'The one route to execution, through a host or sandbox, under a verified grant.', tone: 'trust' },
  { icon: Server, title: 'Remote workers', body: 'mTLS-attested workers, leased assignments and signed, verified results.', tone: 'info' },
  { icon: Package, title: 'Packs & trust', body: 'Signed tool packs with separate integrity, signature and trust states.', tone: 'trust' },
  { icon: Bot, title: 'Controlled autonomy', body: 'Optional AI planning that can propose, never authorise or execute.', tone: 'evidence' },
  { icon: ScrollText, title: 'Events & audit', body: 'An append-only record of every decision, grant, execution and verification.', tone: 'evidence' },
] as const

export function PlatformPage() {
  return (
    <>
      <div className="container page-hero">
        <span className="eyebrow">Platform</span>
        <h1>One console for the whole engagement.</h1>
        <p>From scoping to the final report, every workflow in CYBEROS is shaped by the same rule: the platform proves it was allowed before it acts.</p>
      </div>
      <section className="section">
        <div className="container cards">
          {AREAS.map((a) => (
            <article key={a.title} className="card" data-tone={a.tone}>
              <span className="ico">
                <a.icon size={20} aria-hidden="true" />
              </span>
              <h3>{a.title}</h3>
              <p>{a.body}</p>
            </article>
          ))}
        </div>
      </section>
    </>
  )
}

export function ArchitecturePage() {
  return (
    <>
      <div className="container page-hero">
        <span className="eyebrow">Architecture</span>
        <h1>The execution spine.</h1>
        <p>A Go modular monolith owns every authoritative decision. The React console is a thin, non-authoritative client over its REST API.</p>
      </div>
      <section className="section">
        <div className="container col" style={{ gap: 'var(--s-6)' }}>
          <div className="panel" style={{ padding: 'var(--s-5)' }}>
            <SpineFlow stages={SPINE_ALL_PASS} />
          </div>
          <ol className="steps">
            {STAGE_ORDER.map((s) => (
              <li key={s.id} className="step">
                <div>
                  <h3>{s.label}</h3>
                  <p>{s.blurb}</p>
                </div>
              </li>
            ))}
          </ol>
          <div className="prose">
            <h2>Remote execution</h2>
            <p className="mono">Broker → Remote Host → mTLS Worker → Assignment → Signed Result → Verification → Audit</p>
            <p>Remote workers never self-authorise. They hold a leased assignment issued by the control plane, execute, and return a signed result that the backend verifies before it becomes evidence.</p>
            <h2>Technology</h2>
            <p>Go modular monolith, REST/JSON with Server-Sent Events for live updates, PostgreSQL with SQL migrations. Frontend: TypeScript, React and Vite.</p>
          </div>
        </div>
      </section>
    </>
  )
}

export function SecurityPage() {
  return (
    <>
      <div className="container page-hero">
        <span className="eyebrow">Security model</span>
        <h1>The browser is never the authority.</h1>
        <p>The console can ask and display. It cannot decide, grant, trust, verify or run. Those boundaries are structural, not a matter of discipline.</p>
      </div>
      <section className="section">
        <div className="container cards">
          {[
            ['No client-side grants', 'Grants are minted and signed by the control plane. The browser only shows opaque references.'],
            ['No client-side decisions', 'Authorisation is evaluated server-side. A denied request stays denied whatever the UI shows.'],
            ['No trust authority', 'Pack trust is an administrative act recorded by the backend, never toggled by the front end.'],
            ['No fake verification', 'Result verification happens in the verifier service. The UI renders its verdict.'],
            ['No direct execution', 'There is no browser path to a tool. Everything flows through the broker.'],
            ['Fail closed', 'API errors surface as visible failures, not silent defaults.'],
          ].map(([t, b]) => (
            <article key={t} className="card" data-tone="trust">
              <h3>{t}</h3>
              <p>{b}</p>
            </article>
          ))}
        </div>
        <div className="container" style={{ marginTop: 'var(--s-6)' }}>
          <Link to="/app/packs" className="link">
            See how trust is presented in the console →
          </Link>
        </div>
      </section>
    </>
  )
}
