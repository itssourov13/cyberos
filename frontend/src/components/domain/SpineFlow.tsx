import { Check, Minus, X } from 'lucide-react'
import type { SpineStage, StageOutcome } from '@/api/types'
import { STAGE_ORDER } from './spine-stages'


const outcomeLabel: Record<StageOutcome, string> = { pass: 'passed', deny: 'denied', skipped: 'not reached', pending: 'pending' }

interface Props {
  stages: Pick<SpineStage, 'id' | 'outcome'>[]
  /** Index currently being evaluated (shows a pulse). */
  activeIndex?: number
  compact?: boolean
}

/** The signature visual: scope → decision → grant → broker → execution → evidence. */
export function SpineFlow({ stages, activeIndex, compact }: Props) {
  return (
    <ol className={compact ? 'spine spine-compact' : 'spine'} aria-label="Execution spine">
      {STAGE_ORDER.map((def, i) => {
        const stage = stages.find((s) => s.id === def.id)
        const outcome: StageOutcome = stage?.outcome ?? 'pending'
        const active = activeIndex === i
        const Icon = outcome === 'pass' ? Check : outcome === 'deny' ? X : outcome === 'skipped' ? Minus : def.icon
        return (
          <li key={def.id} className="spine-node" data-state={outcome} data-active={active || undefined}>
            <span className="spine-dot">
              <Icon size={outcome === 'pending' ? 17 : 18} strokeWidth={outcome === 'pending' ? 2 : 2.6} aria-hidden="true" />
            </span>
            <span className="spine-label">{def.label}</span>
            <span className="sr-only">{outcomeLabel[outcome]}</span>
          </li>
        )
      })}
    </ol>
  )
}
