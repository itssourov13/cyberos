import { afterEach, describe, expect, it, vi } from 'vitest'
import { liveApi } from './live'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

/** Route table keyed by path (query string ignored). Unlisted paths fail the test. */
function stubRoutes(routes: Record<string, unknown>) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string) => {
      const path = input.replace('/api/v1', '').split('?')[0]!
      if (!(path in routes)) throw new Error(`unexpected request: ${path}`)
      return json(routes[path])
    }),
  )
}

afterEach(() => vi.unstubAllGlobals())

const projects = { projects: [{ ID: 'proj-a', Name: 'A' }] }

describe('liveApi workers', () => {
  it('only shows a worker online when active with a fresh heartbeat', async () => {
    const fresh = new Date().toISOString()
    const old = new Date(Date.now() - 60 * 60_000).toISOString()
    stubRoutes({
      '/projects': projects,
      '/projects/proj-a/workers': {
        workers: [
          { worker_id: 'w1', project_id: 'proj-a', host_id: 'h1', status: 'ACTIVE', last_seen_at: fresh },
          { worker_id: 'w2', project_id: 'proj-a', host_id: 'h2', status: 'ACTIVE', last_seen_at: old },
          { worker_id: 'w3', project_id: 'proj-a', host_id: 'h3', status: 'SUSPENDED', last_seen_at: fresh },
          { worker_id: 'w4', project_id: 'proj-a', host_id: 'h4', status: 'REVOKED', last_seen_at: fresh },
          { worker_id: 'w5', project_id: 'proj-a', host_id: 'h5', status: 'PENDING', last_seen_at: '0001-01-01T00:00:00Z' },
        ],
      },
    })
    const states = Object.fromEntries((await liveApi.workers()).map((w) => [w.id, w.state]))
    expect(states).toEqual({ w1: 'online', w2: 'stale', w3: 'suspended', w4: 'revoked', w5: 'stale' })
  })
})

describe('liveApi assignments', () => {
  it('maps backend states and never claims a result is verified', async () => {
    stubRoutes({
      '/projects': projects,
      '/projects/proj-a/assignments': {
        assignments: [
          { assignment_id: 'a1', execution_id: 'ex1', worker_id: 'w1', status: 'RUNNING', lease_until: '2026-11-05T10:00:00Z', result_hash: 'abc' },
          { assignment_id: 'a2', execution_id: 'ex2', worker_id: 'w1', status: 'COMPLETED', lease_until: '', result_hash: 'def' },
          { assignment_id: 'a3', execution_id: 'ex3', worker_id: 'w2', status: 'EXPIRED', lease_until: '' },
        ],
      },
    })
    const rows = await liveApi.assignments()
    expect(rows.map((r) => r.state)).toEqual(['leased', 'completed', 'stale'])
    expect(rows.every((r) => r.resultVerified === false)).toBe(true)
    expect(rows[0]).toMatchObject({ id: 'a1', taskId: 'ex1', workerId: 'w1' })
  })
})

describe('liveApi audit', () => {
  it('maps entries to tones and surfaces no digest it does not have', async () => {
    stubRoutes({
      '/projects': projects,
      '/audit': {
        entries: [
          { id: 'e1', action: 'DECISION', at: '2026-11-05T01:00:00Z', actor: 'u1', outcome: 'DENIED', reason: 'out of scope', task_id: 't1' },
          { id: 'e2', action: 'GRANT_ISSUED', at: '2026-11-05T00:59:00Z', actor: '', outcome: 'SUCCEEDED', grant_id: 'g1' },
          { id: 'e3', action: 'PACK_TRUSTED', at: '2026-11-05T00:58:00Z', actor: 'op', outcome: 'SUCCEEDED' },
        ],
      },
    })
    const rows = await liveApi.audit()
    expect(rows.map((r) => r.tone)).toEqual(['danger', 'grant', 'trust'])
    expect(rows[0]!.subject).toBe('t1 · out of scope')
    expect(rows[1]!.actor).toBe('system')
    expect(rows.every((r) => r.digest === '')).toBe(true)
  })

  it('propagates a 403 for a viewer instead of faking an empty feed', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string) =>
        input.includes('/audit') ? json({ error: 'insufficient role for this operation' }, 403) : json(projects),
      ),
    )
    await expect(liveApi.audit()).rejects.toMatchObject({ code: 'forbidden' })
  })
})
