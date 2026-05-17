import { createContext, useContext, useState, useEffect, ReactNode } from 'react'
import axios from 'axios'
import { setAccessToken } from '../lib/api'

interface User { id: number; email: string; role: string }
interface AuthCtx {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthCtx | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    axios.post('/api/auth/refresh', {}, { withCredentials: true })
      .then(res => {
        const token: string = res.data.access_token
        setAccessToken(token)
        const payload = JSON.parse(atob(token.split('.')[1]))
        setUser({ id: payload.uid, email: payload.email, role: payload.role })
      })
      .catch(() => setUser(null))
      .finally(() => setLoading(false))
  }, [])

  async function login(email: string, password: string) {
    const res = await axios.post('/api/auth/login', { email, password }, { withCredentials: true })
    const token: string = res.data.access_token
    setAccessToken(token)
    const payload = JSON.parse(atob(token.split('.')[1]))
    setUser({ id: payload.uid, email: payload.email, role: payload.role })
  }

  async function logout() {
    await axios.post('/api/auth/logout', {}, { withCredentials: true })
    setAccessToken(null)
    setUser(null)
  }

  return <AuthContext.Provider value={{ user, loading, login, logout }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be inside AuthProvider')
  return ctx
}
