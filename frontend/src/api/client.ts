import type { ApiClient } from './types'
import { mockApiClient } from './mock/client'

/**
 * API boundary. Default: MOCK adapter for Part 1.
 * Production will target CYBEROS Go /api/v1 without client-side authz.
 */
const useMock = import.meta.env.VITE_USE_MOCK !== 'false'

export const api: ApiClient = useMock ? mockApiClient : mockApiClient

export function isMockMode(): boolean {
  return useMock
}
