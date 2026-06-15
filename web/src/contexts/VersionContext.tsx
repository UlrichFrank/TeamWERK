import { createContext, useContext, ReactNode } from 'react'
import { useVersionCheck } from '../hooks/useVersionCheck'

interface VersionCtx {
  version: string | null
  latestVersion: string | null
  updateAvailable: boolean
}

const VersionContext = createContext<VersionCtx | null>(null)

export function VersionProvider({ children }: { children: ReactNode }) {
  const value = useVersionCheck()
  return <VersionContext.Provider value={value}>{children}</VersionContext.Provider>
}

export function useVersion(): VersionCtx {
  const ctx = useContext(VersionContext)
  if (!ctx) throw new Error('useVersion must be used within VersionProvider')
  return ctx
}
