export function LogoMark({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" fill="none" aria-hidden="true">
      <rect width="64" height="64" rx="14" fill="var(--bg)" />
      <rect x="1" y="1" width="62" height="62" rx="13" stroke="var(--line-strong)" strokeWidth="2" />
      <path d="M32 12v40" stroke="var(--accent)" strokeWidth="3" strokeLinecap="round" opacity=".45" />
      <circle cx="32" cy="14" r="5" fill="var(--accent)" />
      <circle cx="32" cy="32" r="5" fill="var(--bg)" stroke="var(--accent)" strokeWidth="3" />
      <circle cx="32" cy="50" r="5" fill="var(--accent)" />
      <path d="M32 32h14" stroke="var(--grant)" strokeWidth="3" strokeLinecap="round" />
      <circle cx="48" cy="32" r="3" fill="var(--grant)" />
    </svg>
  )
}

export function Wordmark({ size = 28 }: { size?: number }) {
  return (
    <span className="brand">
      <LogoMark size={size} />
      <span className="brand-name">CYBEROS</span>
    </span>
  )
}
