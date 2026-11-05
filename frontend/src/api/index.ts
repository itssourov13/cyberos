import { createContext, createElement, useContext, type ReactNode } from 'react'
import type { CyberosApi } from './client'
import { liveApi } from './live'
import { previewApi } from './preview'

export { ApiError } from './client'
export type { CyberosApi } from './client'

/**
 * Live is the default. Preview fixtures are used only when explicitly
 * selected at build/dev time with `VITE_CYBEROS_API_MODE=preview`; a failing
 * backend never silently degrades to fixtures.
 */
const defaultApi: CyberosApi = import.meta.env.VITE_CYBEROS_API_MODE === 'preview' ? previewApi : liveApi

const ApiContext = createContext<CyberosApi>(defaultApi)

export function ApiProvider({ api = defaultApi, children }: { api?: CyberosApi; children: ReactNode }) {
  return createElement(ApiContext.Provider, { value: api }, children)
}

export const useApi = () => useContext(ApiContext)
