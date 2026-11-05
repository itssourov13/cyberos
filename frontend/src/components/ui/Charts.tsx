import type { Tone } from '@/api/types'

export function Sparkline({ values, width = 96, height = 28, tone = 'ok' }: { values: number[]; width?: number; height?: number; tone?: Tone }) {
  if (values.length < 2) return null
  const max = Math.max(...values)
  const min = Math.min(...values)
  const span = max - min || 1
  const step = width / (values.length - 1)
  const pts = values.map((v, i) => `${(i * step).toFixed(1)},${(height - 3 - ((v - min) / span) * (height - 6)).toFixed(1)}`).join(' ')
  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} role="img" aria-label="Trend">
      <polyline points={pts} fill="none" stroke={`var(--${tone === 'neutral' ? 'text-3' : tone})`} strokeWidth="1.6" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}

export function Bars({ values, height = 132, unit = 'h' }: { values: number[]; height?: number; unit?: string }) {
  const max = Math.max(...values, 1)
  const w = 100 / values.length
  return (
    <svg viewBox={`0 0 100 ${height}`} preserveAspectRatio="none" width="100%" height={height} role="img" aria-label="Executions per hour">
      {[0.25, 0.5, 0.75].map((g) => (
        <line key={g} x1="0" x2="100" y1={height * g} y2={height * g} stroke="var(--line)" strokeWidth="0.4" vectorEffect="non-scaling-stroke" />
      ))}
      {values.map((v, i) => {
        const h = (v / max) * (height - 8)
        return (
          <rect key={i} x={i * w + w * 0.16} y={height - h} width={w * 0.68} height={h} rx="0.6" fill={i === values.length - 1 ? 'var(--accent)' : 'var(--accent-line)'}>
            <title>{`${v} executions · ${values.length - 1 - i}${unit} ago`}</title>
          </rect>
        )
      })}
    </svg>
  )
}
