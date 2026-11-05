import { useEffect, useState } from 'react'
import type { StageOutcome, SpineStageId } from '@/api/types'
import { SpineFlow } from '@/components/domain/SpineFlow'
import { STAGE_ORDER } from '@/components/domain/spine-stages'

const SCRIPT: { allow: boolean; label: string }[] = [
  { allow: true, label: 'recon.tls.scan on portal.northwind.example' },
  { allow: false, label: 'auth.spray from an untrusted pack' },
]

/** Marketing illustration only: a scripted walk through the spine. It evaluates nothing. */
export function LiveSpine() {
  const reduce = typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  const [run, setRun] = useState(0)
  const [step, setStep] = useState(reduce ? STAGE_ORDER.length : 0)

  const end = SCRIPT[run].allow ? STAGE_ORDER.length : 2

  useEffect(() => {
    if (reduce) return
    const t = window.setTimeout(
      () => {
        if (step >= end) {
          setStep(0)
          setRun((r) => (r + 1) % SCRIPT.length)
        } else setStep((s) => s + 1)
      },
      step >= end ? 2600 : 760,
    )
    return () => window.clearTimeout(t)
  }, [step, reduce, end])

  const script = SCRIPT[run]
  const denyAt = 1 // decision
  const stages = STAGE_ORDER.map((s, i) => {
    let outcome: StageOutcome = 'pending'
    if (i < step) outcome = script.allow ? 'pass' : i < denyAt ? 'pass' : i === denyAt ? 'deny' : 'skipped'
    if (!script.allow && step > denyAt && i > denyAt) outcome = 'skipped'
    return { id: s.id as SpineStageId, outcome }
  })
  const active = step < end ? step : undefined
  const done = step >= end

  return (
    <div className="hero-spine" aria-label="Illustration of the execution spine">
      <SpineFlow stages={stages} activeIndex={active} />
      <div className="caption" aria-live="off">
        <span className="mono" style={{ color: 'var(--accent)' }}>
          {script.label}
        </span>
        <span>
          {done
            ? script.allow
              ? '→ executed, evidence signed and recorded'
              : '→ denied at Decision, nothing was dispatched'
            : active !== undefined
              ? `→ ${STAGE_ORDER[active].blurb}`
              : ''}
        </span>
      </div>
    </div>
  )
}
