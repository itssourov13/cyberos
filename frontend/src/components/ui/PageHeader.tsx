export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string
  description?: string
  actions?: import("react").ReactNode
}) {
  return (
    <div className="mb-6 flex flex-col gap-3 border-b border-[var(--color-border)] pb-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <h1 className="text-lg font-semibold tracking-tight text-[var(--color-fg)] md:text-xl">{title}</h1>
        {description && (
          <p className="mt-1.5 max-w-2xl text-[13px] leading-relaxed text-[var(--color-muted)]">{description}</p>
        )}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  )
}

export function EmptyState({ title, description }: { title: string; description?: string }) {
  return (
    <div className="ops-panel px-6 py-14 text-center">
      <h2 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--color-muted)]">{title}</h2>
      {description && (
        <p className="mx-auto mt-2 max-w-md text-[13px] leading-relaxed text-[var(--color-muted)]">{description}</p>
      )}
    </div>
  )
}

export function LoadingState({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center gap-3 py-12 text-[13px] text-[var(--color-muted)]" role="status">
      <span className="h-3.5 w-3.5 animate-pulse rounded-full bg-[var(--color-accent)]/50" />
      {label}
    </div>
  )
}

export function ErrorState({
  title = 'Request failed',
  reason,
  requestId,
}: {
  title?: string
  reason?: string
  requestId?: string
}) {
  return (
    <div
      className="rounded-[var(--radius-md)] border border-red-500/25 bg-red-500/5 px-5 py-6"
      role="alert"
    >
      <h2 className="text-sm font-semibold text-red-300">{title}</h2>
      {reason && <p className="ops-mono mt-2 text-red-200/80">Reason: {reason}</p>}
      {requestId && (
        <p className="ops-mono mt-1 text-[var(--color-muted)]">Request ID: {requestId}</p>
      )}
      <p className="mt-3 text-xs text-[var(--color-muted)]">No action was performed.</p>
    </div>
  )
}

export function MetricCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string | number
  hint?: string
}) {
  return (
    <div className="ops-panel p-4 transition-colors duration-150 hover:border-[var(--color-border-strong)]">
      <div className="text-[10px] font-medium uppercase tracking-[0.12em] text-[var(--color-muted)]">
        {label}
      </div>
      <div className="mt-2 text-2xl font-semibold tabular-nums tracking-tight text-[var(--color-fg)]">
        {value}
      </div>
      {hint && <div className="mt-1 text-[11px] text-[var(--color-muted)]">{hint}</div>}
    </div>
  )
}
