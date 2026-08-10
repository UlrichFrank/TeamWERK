import type { ReactNode } from 'react'
import { AuthContext, type AuthCtx, type User } from '../contexts/AuthContext'

const DEFAULT_USER: User = {
  id: 1,
  email: 'test@test.local',
  role: 'standard',
  clubFunctions: [],
  isParent: false,
}

function makeAuthCtx(capabilities: string[] = [], user: User = DEFAULT_USER): AuthCtx {
  return {
    user,
    loading: false,
    impersonating: null,
    mapsProvider: 'auto',
    setMapsProvider: () => {},
    capabilities,
    hasCapability: (cap: string) => capabilities.includes(cap),
    navRoutes: [],
    passwordChangeRecommended: false,
    dismissPasswordChangeHint: () => {},
    login: async () => {},
    logout: async () => {},
    startImpersonation: async () => {},
    stopImpersonation: async () => {},
  }
}

/** Minimaler Auth-Provider für Komponententests, die nur Capabilities brauchen. */
export function WithCapabilities({ caps, children }: { caps: string[]; children: ReactNode }) {
  return <AuthContext.Provider value={makeAuthCtx(caps)}>{children}</AuthContext.Provider>
}
