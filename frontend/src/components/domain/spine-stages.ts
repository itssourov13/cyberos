import { Crosshair, Cpu, Fingerprint, KeyRound, Scale, ShieldCheck, type LucideIcon } from 'lucide-react'
import type { SpineStageId } from '@/api/types'

export const STAGE_ORDER: { id: SpineStageId; label: string; icon: LucideIcon; blurb: string }[] = [
  { id: 'scope', label: 'Scope', icon: Crosshair, blurb: 'Is the target inside an authorised scope and time window?' },
  { id: 'decision', label: 'Decision', icon: Scale, blurb: 'Does policy, pack trust and risk allow this capability?' },
  { id: 'grant', label: 'Grant', icon: KeyRound, blurb: 'A short-lived grant bound to principal, capability and target.' },
  { id: 'broker', label: 'Broker', icon: ShieldCheck, blurb: 'The tool broker verifies the grant before anything is dispatched.' },
  { id: 'execution', label: 'Execution', icon: Cpu, blurb: 'An attested worker runs the tool inside its sandbox.' },
  { id: 'evidence', label: 'Evidence', icon: Fingerprint, blurb: 'The signed result is verified and recorded by the backend.' },
]
