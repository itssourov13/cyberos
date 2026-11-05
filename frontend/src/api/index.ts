import { createContext, createElement, useContext, type ReactNode } from 'react'
import type { CyberosApi } from './client'
import { previewApi } from './preview'

export { ApiError } from './client'
export type { CyberosApi } from './client'

/**
 * Phase 24D swaps this single line for the live `/api/v1` adapter.
 * Until then the console runs on the preview adapter and says so everywhere.
 */
const defaultApi: CyberosApi = previewApi

const ApiContext = createContext<CyberosApi>(defaultApi)

export function ApiProvider({ api = defaultApi, children }: { api?: CyberosApi; children: ReactNode }) {
  return createElement(ApiContext.Provider, { value: api }, children)
}

export const useApi = () => useContext(ApiContext)
