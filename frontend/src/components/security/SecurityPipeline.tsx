const steps = [
  'Scope',
  'Auth',
  'CP',
  'Decision',
  'Grant',
  'Broker',
  'Host',
  'Worker',
  'Result',
  'Verify',
  'Audit',
]

/** Visualization only — frontend does not execute this chain. */
export function SecurityPipeline({ activeIndex = 4 }: { activeIndex?: number }) {
  return (
    <div className="ops-panel p-4">
      <div className="mb-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">
        Authorization → Execution pipeline · server-authoritative
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        {steps.map((s, i) => (
          <div key={s} className="flex items-center gap-1.5">
            <span
              className={`rounded-[var(--radius-sm)] border px-2 py-1 text-[11px] font-medium transition-colors ${
                i === activeIndex
                  ? 'border-[var(--color-accent)]/50 bg-[var(--color-accent)]/10 text-sky-100'
                  : 'border-[var(--color-border)] text-[var(--color-muted)]'
              }`}
            >
              {s}
            </span>
            {i < steps.length - 1 && (
              <span className="text-[10px] text-[var(--color-border-strong)]" aria-hidden>
                →
              </span>
            )}
          </div>
        ))}
      </div>
      <p className="mt-3 text-[11px] leading-relaxed text-[var(--color-muted)]">
        Integrity ≠ Signature ≠ Trust ≠ Eligibility ≠ Authorization ≠ Execution. The browser does not
        mint Grants or call the Broker.
      </p>
    </div>
  )
}

export function ConceptStrip() {
  const items = [
    { k: 'Integrity', v: 'VERIFIED' },
    { k: 'Signature', v: 'VALID' },
    { k: 'Trust', v: 'TRUSTED' },
    { k: 'Eligibility', v: 'ELIGIBLE' },
    { k: 'Authorization', v: 'ALLOWED' },
    { k: 'Execution', v: 'NOT STARTED' },
  ]
  return (
    <div className="grid gap-2 sm:grid-cols-3 lg:grid-cols-6">
      {items.map((it) => (
        <div key={it.k} className="ops-panel-elevated px-3 py-2.5">
          <div className="text-[10px] uppercase tracking-[0.12em] text-[var(--color-muted)]">{it.k}</div>
          <div className="ops-mono mt-1 text-sky-200/90">{it.v}</div>
        </div>
      ))}
    </div>
  )
}
