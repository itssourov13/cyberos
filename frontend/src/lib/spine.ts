import type { SpineStage } from '@/api/types'

/** Illustrative all-pass stage list for explanatory diagrams. Not a real evaluation. */
export const SPINE_ALL_PASS: Pick<SpineStage, 'id' | 'outcome'>[] = [
  { id: 'scope', outcome: 'pass' },
  { id: 'decision', outcome: 'pass' },
  { id: 'grant', outcome: 'pass' },
  { id: 'broker', outcome: 'pass' },
  { id: 'execution', outcome: 'pass' },
  { id: 'evidence', outcome: 'pass' },
]

export const SPINE_ALL_PENDING: Pick<SpineStage, 'id' | 'outcome'>[] = SPINE_ALL_PASS.map((s) => ({ ...s, outcome: 'pending' }))
