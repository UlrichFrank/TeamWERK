import { createContext, useContext, useState, useEffect, useRef, ReactNode } from 'react'
import axios from 'axios'
import { api, setAccessToken } from '../lib/api'

interface User { id: number; email: string; role: string; clubFunctions: string[]; isParent: boolean }
interface Impersonating { userId: number; name: string }
type MapsProvider = 'auto' | 'google' | 'apple'
interface AuthCtx {
  user: User | null
  loading: boolean
  impersonating: Impersonating | null
  mapsProvider: MapsProvider
  setMapsProvider: (p: MapsProvider) => void
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  startImpersonation: (userId: number, name: string) => Promise<void>
  stopImpersonation: () => Promise<void>
}

export type { MapsProvider, User, AuthCtx }


const WARN_MS = 25 * 60 * 1000
const LOGOUT_MS = 30 * 60 * 1000
const COUNTDOWN_SECS = 5 * 60
const IDLE_EVENTS = ['mousemove', 'keydown', 'click', 'touchstart', 'scroll'] as const

export const AuthContext = createContext<AuthCtx | null>(null)

function fmtCountdown(s: number): string {
  const m = Math.floor(s / 60)
  const sec = s % 60
  return `${m}:${sec.toString().padStart(2, '0')}`
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [impersonating, setImpersonating] = useState<Impersonating | null>(null)
  const [showWarning, setShowWarning] = useState(false)
  const [countdown, setCountdown] = useState(COUNTDOWN_SECS)
  const [mapsProvider, setMapsProvider] = useState<MapsProvider>('auto')

  async function loadMapsProvider() {
    try {
      const res = await api.get('/profile/me')
      const p = res.data.maps_provider
      if (p === 'google' || p === 'apple' || p === 'auto') setMapsProvider(p)
    } catch {
      // keep default 'auto'
    }
  }

  const warnTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const logoutTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const countdownInterval = useRef<ReturnType<typeof setInterval> | null>(null)

  function clearTimers() {
    if (warnTimer.current) { clearTimeout(warnTimer.current); warnTimer.current = null }
    if (logoutTimer.current) { clearTimeout(logoutTimer.current); logoutTimer.current = null }
    if (countdownInterval.current) { clearInterval(countdownInterval.current); countdownInterval.current = null }
  }

  async function logout() {
    clearTimers()
    setShowWarning(false)
    await axios.post('/api/auth/logout', {}, { withCredentials: true })
    setAccessToken(null)
    setUser(null)
  }

  function resetTimer() {
    clearTimers()
    setShowWarning(false)
    setCountdown(COUNTDOWN_SECS)

    warnTimer.current = setTimeout(() => {
      setShowWarning(true)
      setCountdown(COUNTDOWN_SECS)
      countdownInterval.current = setInterval(() => {
        setCountdown(c => Math.max(0, c - 1))
      }, 1000)
    }, WARN_MS)

    logoutTimer.current = setTimeout(logout, LOGOUT_MS)
  }

  useEffect(() => {
    axios.post('/api/auth/refresh', {}, { withCredentials: true })
      .then(res => {
        const token: string = res.data.access_token
        setAccessToken(token)
        const payload = JSON.parse(atob(token.split('.')[1]))
        setUser({ id: payload.uid, email: payload.email, role: payload.role, clubFunctions: payload.club_functions ?? [], isParent: payload.is_parent ?? false })
        loadMapsProvider()
      })
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Start idle timer only when logged in; clean up on logout or unmount
  useEffect(() => {
    if (!user) {
      clearTimers()
      setShowWarning(false)
      return
    }

    IDLE_EVENTS.forEach(e => window.addEventListener(e, resetTimer))
    resetTimer()

    return () => {
      IDLE_EVENTS.forEach(e => window.removeEventListener(e, resetTimer))
      clearTimers()
    }
  }, [user]) // eslint-disable-line react-hooks/exhaustive-deps

  async function login(email: string, password: string) {
    const res = await axios.post('/api/auth/login', { email, password }, { withCredentials: true })
    const token: string = res.data.access_token
    setAccessToken(token)
    const payload = JSON.parse(atob(token.split('.')[1]))
    setUser({ id: payload.uid, email: payload.email, role: payload.role, clubFunctions: payload.club_functions ?? [], isParent: payload.is_parent ?? false })
    loadMapsProvider()
  }

  async function startImpersonation(userId: number, name: string) {
    const res = await api.post(`/impersonate/${userId}`)
    const token: string = res.data.access_token
    setAccessToken(token)
    const payload = JSON.parse(atob(token.split('.')[1]))
    setUser({ id: payload.uid, email: payload.email, role: payload.role, clubFunctions: payload.club_functions ?? [], isParent: payload.is_parent ?? false })
    setImpersonating({ userId, name })
    loadMapsProvider()
  }

  async function stopImpersonation() {
    const res = await axios.post('/api/auth/refresh', {}, { withCredentials: true })
    const token: string = res.data.access_token
    setAccessToken(token)
    const payload = JSON.parse(atob(token.split('.')[1]))
    setUser({ id: payload.uid, email: payload.email, role: payload.role, clubFunctions: payload.club_functions ?? [], isParent: payload.is_parent ?? false })
    setImpersonating(null)
    loadMapsProvider()
  }

  return (
    <AuthContext.Provider value={{ user, loading, impersonating, mapsProvider, setMapsProvider, login, logout, startImpersonation, stopImpersonation }}>
      {children}
      {showWarning && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-2xl p-6 max-w-sm w-full mx-4">
            <h2 className="text-lg font-bold text-gray-900 mb-2">Sitzung läuft ab</h2>
            <p className="text-gray-600 mb-1">Sie werden automatisch abgemeldet.</p>
            <p className="text-4xl font-mono font-bold text-center my-4">{fmtCountdown(countdown)}</p>
            <div className="flex gap-3 mt-4">
              <button
                onClick={resetTimer}
                className="flex-1 bg-brand-yellow hover:bg-black hover:text-white text-black font-semibold py-2.5 px-4 rounded-lg transition-colors"
              >
                Angemeldet bleiben
              </button>
              <button
                onClick={logout}
                className="flex-1 bg-gray-100 hover:bg-gray-200 text-gray-700 font-semibold py-2.5 px-4 rounded-lg transition-colors"
              >
                Jetzt abmelden
              </button>
            </div>
          </div>
        </div>
      )}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be inside AuthProvider')
  return ctx
}
