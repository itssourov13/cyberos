import type { ReactNode } from 'react'

export function DataTable({ headers, children }: { headers: string[]; children: ReactNode }) {
  return (
    <div className="ops-panel overflow-x-auto">
      <table className="w-full min-w-[640px] text-left text-[13px]">
        <thead className="border-b border-[var(--color-border)] bg-[var(--color-surface-2)] text-[10px] uppercase tracking-[0.12em] text-[var(--color-muted)]">
          <tr>
            {headers.map((h) => (
              <th key={h} className="px-3 py-2.5 font-medium">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--color-border)]">{children}</tbody>
      </table>
    </div>
  )
}

export function SearchInput({
  value,
  onChange,
  placeholder,
}: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
}) {
  return (
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder ?? 'Search…'}
      className="w-full max-w-xs rounded-[var(--radius-sm)] border border-[var(--color-border)] bg-[var(--color-surface-2)] px-3 py-1.5 text-[13px] outline-none transition-colors focus:border-[var(--color-border-focus)]"
      aria-label={placeholder ?? 'Search'}
    />
  )
}

export function SeverityBadge({ severity }: { severity: string }) {
  const s = severity.toUpperCase()
  const map: Record<string, string> = {
    CRITICAL: 'bg-red-600/15 text-red-300 border-red-500/35',
    HIGH: 'bg-orange-500/12 text-orange-300 border-orange-500/30',
    MEDIUM: 'bg-amber-500/12 text-amber-200 border-amber-500/25',
    LOW: 'bg-sky-500/12 text-sky-300 border-sky-500/25',
    INFO: 'bg-slate-500/12 text-slate-300 border-slate-500/25',
  }
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-[var(--radius-sm)] border px-2 py-0.5 text-[11px] font-medium ${map[s] ?? map.INFO}`}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" aria-hidden />
      {s}
    </span>
  )
}

export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  onConfirm,
  onCancel,
}: {
  open: boolean
  title: string
  body: ReactNode
  confirmLabel: string
  onConfirm: () => void
  onCancel: () => void
}) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/65 p-4 backdrop-blur-[2px]" role="dialog" aria-modal>
      <div className="w-full max-w-md rounded-[var(--radius-lg)] border border-orange-500/35 bg-[var(--color-surface-1)] p-5 shadow-2xl">
        <h2 className="text-sm font-semibold tracking-wide text-orange-100">{title}</h2>
        <div className="mt-3 text-[13px] text-[var(--color-muted)]">{body}</div>
        <div className="mt-5 flex justify-end gap-2">
          <button type="button" onClick={onCancel} className="ops-btn ops-btn-secondary">
            Cancel
          </button>
          <button type="button" onClick={onConfirm} className="ops-btn ops-btn-danger">
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
