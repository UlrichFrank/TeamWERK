import { createContext, useContext, useRef, useState, useEffect, ReactNode } from 'react'
import { api } from '../lib/api'
import { useAuth } from './AuthContext'

export interface PhoneEntry {
  label: string
  number: string
}

export interface PersonContact {
  name: string
  photo_url?: string
  phones?: PhoneEntry[]
  address?: string
  email?: string
  whatsapp_visible: boolean
}

type ContactState = PersonContact | 'loading' | 'error'

interface PersonContactCtx {
  get: (userId: number) => ContactState | undefined
  fetchContact: (userId: number) => void
}

const PersonContactContext = createContext<PersonContactCtx | null>(null)

export function PersonContactProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [cache, setCache] = useState<Map<number, ContactState>>(new Map())
  const prevUserId = useRef<number | undefined>(user?.id)

  useEffect(() => {
    if (user?.id !== prevUserId.current) {
      setCache(new Map())
      prevUserId.current = user?.id
    }
  }, [user?.id])

  function fetchContact(userId: number) {
    if (cache.has(userId)) return
    setCache(prev => new Map(prev).set(userId, 'loading'))
    api.get<PersonContact>(`/users/${userId}/contact`)
      .then(res => setCache(prev => new Map(prev).set(userId, res.data)))
      .catch(() => setCache(prev => new Map(prev).set(userId, 'error')))
  }

  return (
    <PersonContactContext.Provider value={{ get: (id) => cache.get(id), fetchContact }}>
      {children}
    </PersonContactContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components -- Hook-Export neben Provider; betrifft nur Dev-HMR
export function usePersonContact() {
  const ctx = useContext(PersonContactContext)
  if (!ctx) throw new Error('usePersonContact must be used within PersonContactProvider')
  return ctx
}
