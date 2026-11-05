export const cx = (...parts: (string | false | null | undefined)[]) => parts.filter(Boolean).join(' ')

/** Relative time, e.g. "12m ago" / "in 9m". */
export function relTime(iso: string, now = Date.now()): string {
  const diff = now - new Date(iso).getTime()
  const abs = Math.abs(diff)
  const m = Math.round(abs / 60_000)
  let text: string
  if (m < 1) text = 'just now'
  else if (m < 60) text = `${m}m`
  else if (m < 60 * 24) text = `${Math.round(m / 60)}h`
  else text = `${Math.round(m / 1440)}d`
  if (text === 'just now') return text
  return diff >= 0 ? `${text} ago` : `in ${text}`
}

export function fmtDateTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString('en-GB', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit', hour12: false })
}

export function fmtDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export const shortDigest = (d: string) => (d.length > 22 ? `${d.slice(0, 15)}…${d.slice(-6)}` : d)

export const pct = (n: number) => `${Math.round(n * 100)}%`
