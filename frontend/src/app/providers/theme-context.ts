import { createContext, useContext } from 'react'

export type ThemeName = 'dark' | 'light'

export interface ThemeCtx {
  theme: ThemeName
  toggle: () => void
}

export const ThemeContext = createContext<ThemeCtx>({ theme: 'dark', toggle: () => {} })

export const useTheme = () => useContext(ThemeContext)
