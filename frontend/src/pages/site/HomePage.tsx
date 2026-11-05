import { Link } from 'react-router-dom'
import { ArrowRight, Crosshair, Fingerprint, KeyRound, Network, PackageCheck, ScrollText, ShieldCheck } from 'lucide-react'
import { LiveSpine } from '@/components/site/LiveSpine'

const PILLARS = [
  { icon: Crosshair, tone: 'ok', title: 'Scope first', body: 'Every action is bound to an authorised scope and time window before anything else is considered.' },
  { icon: KeyRound, tone: 'grant', title: 'Grants, not trust-me', body: 'The control plane issues short-lived grants bound to principal, capability, target and pack digest.' },
  { icon: ShieldCheck, tone: 'trust', title: 'One broker', body: 'A single tool broker is the only path to execution. No side doors, no browser-direct tooling.' },
  { icon: PackageCheck, tone: 'trust', title: 'Signed packs', body: 'Integrity, signature, trust and authorisation are separate checks — never collapsed into one.' },
  { icon: Network, tone: 'info', title: 'Attested workers', body: 'Remote workers connect over mTLS, take leased assignments and return signed results.' },
  { icon: Fingerprint, tone: 'evidence', title: 'Evidence with provenance', body: 'Results are verified by the backend and chained to the decision and grant that allowed them.' },
] as const

const STEPS = [
  ['Define the engagement', 'Projects carry scopes: the targets and the window you are authorised to touch.'],
  ['Ask, don’t run', 'Operators and automation submit a request. The control plane decides; the browser never does.'],
  ['Execute through the broker', 'Approved work is dispatched to a sandbox or an mTLS worker under a verified grant.'],
  ['Keep the proof', 'Signed results become evidence, findings, cases and reports — all traceable and auditable.'],
]

export function HomePage() {
  return (
    <>
      <section className="hero">
        <div className="container hero-inner">
          <span className="eyebrow reveal">The cybersecurity operating platform</span>
          <h1 className="reveal reveal-2">
            Authorized security operations, <em>proven</em>.
          </h1>
          <p className="lede reveal reveal-3">
            CYBEROS runs offensive and defensive security work through a single chain of custody: scope, decision, grant, broker, execution and evidence. Nothing runs unless every link says yes — and every yes is on the record.
          </p>
          <div className="hero-actions reveal reveal-4">
            <Link to="/app" className="btn btn-primary btn-lg">
              Open the console <ArrowRight size={16} />
            </Link>
            <Link to="/architecture" className="btn btn-lg">
              See how it works
            </Link>
          </div>
          <div className="reveal reveal-4" style={{ width: '100%', display: 'flex', justifyContent: 'center' }}>
            <LiveSpine />
          </div>
        </div>
      </section>

      <section className="section">
        <div className="container">
          <div className="section-head">
            <span className="eyebrow">Principles</span>
            <h2>Built so that “allowed” has to be earned.</h2>
            <p>Security tooling is powerful and dangerous. CYBEROS puts the authority in one place and makes the rest of the system prove it asked.</p>
          </div>
          <div className="cards">
            {PILLARS.map((p) => (
              <article key={p.title} className="card" data-tone={p.tone}>
                <span className="ico">
                  <p.icon size={20} aria-hidden="true" />
                </span>
                <h3>{p.title}</h3>
                <p>{p.body}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="section">
        <div className="container">
          <div className="section-head">
            <span className="eyebrow">Five things that are not the same</span>
            <h2>Integrity ≠ Signature ≠ Trust ≠ Authorization ≠ Execution</h2>
            <p>Most platforms blur these together. CYBEROS keeps each one a separate, inspectable decision.</p>
          </div>
          <div className="distinct">
            <div><strong>Integrity</strong><span>The bytes match the digest.</span></div>
            <div><strong>Signature</strong><span>A known key vouches for those bytes.</span></div>
            <div><strong>Trust</strong><span>An administrator chose to rely on that publisher.</span></div>
            <div><strong>Authorization</strong><span>This principal may do this, here, now.</span></div>
            <div><strong>Execution</strong><span>Only the broker, only with a grant.</span></div>
          </div>
        </div>
      </section>

      <section className="section">
        <div className="container">
          <div className="section-head">
            <span className="eyebrow">Workflow</span>
            <h2>From engagement to evidence, one line of custody.</h2>
          </div>
          <ol className="steps">
            {STEPS.map(([t, d]) => (
              <li key={t} className="step">
                <div>
                  <h3>{t}</h3>
                  <p>{d}</p>
                </div>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section className="section">
        <div className="container">
          <div className="cta-band">
            <ScrollText size={26} color="var(--accent)" aria-hidden="true" />
            <h2>Explore the operator console</h2>
            <p className="dim" style={{ maxWidth: '56ch' }}>
              The console currently runs on built-in sample data so you can walk every workflow. Live API integration follows in the next phase.
            </p>
            <Link to="/app" className="btn btn-primary btn-lg">
              Open the console <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      </section>
    </>
  )
}
