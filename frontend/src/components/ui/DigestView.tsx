import { useState } from 'react'
import { Copy, Check } from 'lucide-react'

export function DigestView({
  value,
  label,
  truncate = 16,
}: {
  value: string
  label?: string
  truncate?: number
}) {
  const [copied, setCopied] = useState(false)
  const short =
    value.length > truncate * 2 ? `${value.slice(0, truncate)}…${value.slice(-8)}` : value

  async function onCopy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 1400)
    } catch {
      /* ignore */
    }
  }

  return (
    <div className="inline-flex max-w-full items-center gap-1.5">
      {label && <span className="text-[10px] uppercase tracking-wider text-[var(--color-muted)]">{label}</span>}
      <code className="ops-mono truncate text-sky-200/90" title={value}>
        {short}
      </code>
      <button
        type="button"
        onClick={onCopy}
        className="rounded p-1 text-[var(--color-muted)] transition-colors hover:bg-[var(--color-surface-3)] hover:text-[var(--color-fg)]"
        aria-label="Copy value"
      >
        {copied ? <Check size={13} className="text-[var(--color-ok)]" /> : <Copy size={13} />}
      </button>
    </div>
  )
}

export function IDView({ value, label }: { value: string; label?: string }) {
  return <DigestView value={value} label={label} truncate={12} />
}
