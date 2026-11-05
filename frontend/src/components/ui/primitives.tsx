import { useState, type ReactNode } from 'react'
import { AlertTriangle, Check, Copy, Info, Inbox, OctagonAlert } from 'lucide-react'
import type { Tone } from '@/api/types'
import { cx, shortDigest } from '@/lib/format'
import { Sparkline } from './Charts'

export function Badge({ tone = 'neutral', dot, live, children }: { tone?: Tone; dot?: boolean; live?: boolean; children: ReactNode }) {
  return (
    <span className={cx('badge', tone !== 'neutral' && `badge-${tone}`, live && 'badge-live')}>
      {(dot || live) && <span className="dot" aria-hidden="true" />}
      {children}
    </span>
  )
}

export function Panel({ children, className }: { children: ReactNode; className?: string }) {
  return <section className={cx('panel', className)}>{children}</section>
}

export function PanelHead({ title, sub, actions }: { title: string; sub?: string; actions?: ReactNode }) {
  return (
    <header className="panel-head">
      <h3>{title}</h3>
      {sub && <span className="sub">{sub}</span>}
      <span className="spacer" />
      {actions}
    </header>
  )
}

export function PageHeader({ eyebrow, title, description, actions }: { eyebrow?: string; title: string; description?: string; actions?: ReactNode }) {
  return (
    <div className="page-head">
      <div className="col" style={{ gap: 6 }}>
        {eyebrow && <span className="eyebrow">{eyebrow}</span>}
        <h1>{title}</h1>
        {description && <p className="dim page-desc">{description}</p>}
      </div>
      <span className="spacer" />
      {actions && <div className="row-wrap">{actions}</div>}
    </div>
  )
}

export function Stat({ label, value, delta, spark, tone }: { label: string; value: ReactNode; delta?: string; spark?: number[]; tone?: Tone }) {
  return (
    <div className="panel stat">
      <span className="label">{label}</span>
      <span className="value" style={tone && tone !== 'neutral' ? { color: `var(--${tone})` } : undefined}>
        {value}
      </span>
      {delta && <span className="delta">{delta}</span>}
      {spark && (
        <span className="spark">
          <Sparkline values={spark} />
        </span>
      )}
    </div>
  )
}

const bannerIcon = { info: Info, warn: AlertTriangle, danger: OctagonAlert } as const

export function Banner({ tone = 'info', title, children }: { tone?: 'info' | 'warn' | 'danger'; title?: string; children?: ReactNode }) {
  const Icon = bannerIcon[tone]
  return (
    <div className={cx('banner', `banner-${tone}`)} role={tone === 'danger' ? 'alert' : 'note'}>
      <Icon size={16} aria-hidden="true" />
      <div className="col" style={{ gap: 2 }}>
        {title && <strong>{title}</strong>}
        {children && <span className="dim">{children}</span>}
      </div>
    </div>
  )
}

export function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="empty">
      <Inbox size={22} aria-hidden="true" />
      <strong className="dim">{title}</strong>
      {children && <span>{children}</span>}
    </div>
  )
}

export function Skeleton({ h = 14, w = '100%' }: { h?: number; w?: string | number }) {
  return <div className="skeleton" style={{ height: h, width: w }} aria-hidden="true" />
}

export function Meter({ value, tone }: { value: number; tone?: 'warn' | 'danger' }) {
  const v = Math.max(0, Math.min(1, value))
  return (
    <div className={cx('meter', tone)} role="meter" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(v * 100)}>
      <i style={{ width: `${v * 100}%` }} />
    </div>
  )
}

export function KV({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="kv">
      {items.map(([k, v]) => (
        <div key={k} style={{ display: 'contents' }}>
          <dt>{k}</dt>
          <dd>{v}</dd>
        </div>
      ))}
    </dl>
  )
}

export function Digest({ value }: { value: string }) {
  const [copied, setCopied] = useState(false)
  // Preview adapters emit `sample:` identifiers; they are placeholders, never cryptographic evidence.
  const isSample = value.startsWith('sample:')
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1400)
    } catch {
      /* clipboard unavailable */
    }
  }
  return (
    <span className={isSample ? 'digest digest-sample' : 'digest'} title={isSample ? `${value} — preview sample identifier, not a cryptographic digest` : value}>
      <span>{shortDigest(value)}</span>
      <button type="button" onClick={copy} aria-label="Copy digest">
        {copied ? <Check size={13} /> : <Copy size={13} />}
      </button>
    </span>
  )
}

export function Segmented<T extends string>({ options, value, onChange, label }: { options: { value: T; label: string }[]; value: T; onChange: (v: T) => void; label: string }) {
  return (
    <div className="seg" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={o.value === value} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

export function Tabs<T extends string>({ tabs, value, onChange }: { tabs: { value: T; label: string; count?: number }[]; value: T; onChange: (v: T) => void }) {
  return (
    <div className="tabs" role="tablist">
      {tabs.map((t) => (
        <button key={t.value} type="button" role="tab" className="tab" aria-selected={t.value === value} onClick={() => onChange(t.value)}>
          {t.label}
          {t.count !== undefined && <span className="muted"> · {t.count}</span>}
        </button>
      ))}
    </div>
  )
}
