import { useState, useEffect, useCallback } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { Menu, X, ChevronRight, ChevronDown, Eye } from 'lucide-react'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import { usePushSubscription } from '../hooks/usePushSubscription'
import { useChatEvents } from '../hooks/useChatEvents'
import { useVersionCheck } from '../hooks/useVersionCheck'
import { api } from '../lib/api'

interface NavModule {
  label: string
  items: { to: string; label: string; roles: string[]; excludeRoles?: string[]; end?: boolean }[]
}

const navModules: NavModule[] = [
  {
    label: 'Nutzer',
    items: [
      { to: '/', label: 'Dashboard', roles: [], end: true },
      { to: '/profil', label: 'Mein Profil', roles: [], excludeRoles: ['admin'] },
    ],
  },
  {
    label: 'Spielbetrieb',
    items: [
      { to: '/kalender', label: 'Kalender', roles: [] },
      { to: '/termine', label: 'Termine', roles: [] },
    ],
  },
  {
    label: 'Verein',
    items: [
      { to: '/mein-team', label: 'Mein Team', roles: [] },
      { to: '/dokumente', label: 'Dokumente', roles: [] },
      { to: '/dienste', label: 'Dienste', roles: [] },
      { to: '/mitfahrgelegenheiten', label: 'Mitfahrten', roles: [] },
      { to: '/chat', label: 'Nachrichten', roles: [] },
    ],
  },
  {
    label: 'Verwaltung',
    items: [
      { to: '/admin/nutzer', label: 'Nutzerverwaltung', roles: ['admin', 'vorstand'] },
      { to: '/mitglieder', label: 'Mitglieder', roles: ['admin', 'vorstand'] },
      { to: '/admin/kader', label: 'Kader', roles: ['admin', 'vorstand', 'trainer', 'sportliche_leitung'] },
      { to: '/admin/diensttypen', label: 'Diensttypen', roles: ['admin', 'vorstand'] },
      { to: '/admin/dienstplan-vorlagen', label: 'Dienstplan-Vorlagen', roles: ['admin', 'vorstand'] },
      { to: '/admin/veranstaltungsorte', label: 'Veranstaltungsorte', roles: ['admin', 'vorstand'] },
      { to: '/admin/einstellungen', label: 'Einstellungen', roles: ['admin', 'vorstand'] },
    ],
  },
]

function initOpenModules(): Record<string, boolean> {
  const state: Record<string, boolean> = {}
  for (const m of navModules) {
    const stored = localStorage.getItem(`nav-open-${m.label}`)
    state[m.label] = stored !== null ? stored === 'true' : true
  }
  return state
}

interface ChildEntry { id: number; first_name: string; last_name: string }

export default function AppShell() {
  const { user, logout, impersonating, stopImpersonation } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const isMobile = useMediaQuery('(max-width: 639px)')
  usePushSubscription()
  const { version } = useVersionCheck()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [openModules, setOpenModules] = useState<Record<string, boolean>>(initOpenModules)
  const [navChildren, setNavChildren] = useState<ChildEntry[]>([])
  const [chatUnread, setChatUnread] = useState(0)

  const loadChatUnread = useCallback(async () => {
    if (!user) return
    try {
      const [convs, bcs] = await Promise.all([
        api.get('/chat/conversations'),
        api.get('/chat/broadcasts'),
      ])
      const convUnread = (convs.data ?? []).reduce((s: number, c: any) => s + (c.unreadCount ?? 0), 0)
      const bcUnread = (bcs.data ?? []).filter((b: any) => !b.isRead && !b.isSent).length
      setChatUnread(convUnread + bcUnread)
    } catch {}
  }, [user])

  useEffect(() => {
    if (!user) return
    api.get('/profile/me').then(r => {
      const kids: ChildEntry[] = (r.data?.children ?? [])
        .slice()
        .sort((a: ChildEntry, b: ChildEntry) => a.first_name.localeCompare(b.first_name, 'de'))
      setNavChildren(kids)
    }).catch(() => {})
    loadChatUnread()
  }, [user?.id, loadChatUnread])

  useChatEvents((event) => {
    if (event.startsWith('chat:new-message') || event === 'chat:new-broadcast') {
      loadChatUnread()
    }
  })

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const toggleModule = (label: string) => {
    setOpenModules(prev => {
      const next = { ...prev, [label]: !prev[label] }
      localStorage.setItem(`nav-open-${label}`, String(next[label]))
      return next
    })
  }

  const closeSidebar = () => setSidebarOpen(false)

  const Sidebar = () => (
    <aside className="w-56 bg-brand-gray text-brand-black flex flex-col overflow-y-auto">
      <div className="px-4 py-5 border-b border-brand-black/10 flex items-center justify-between">
        <NavLink to="/" onClick={closeSidebar} className="flex items-center gap-3 hover:opacity-80 transition-opacity">
          <img src="/logo.svg" alt="Team Stuttgart" className="h-8 w-8" />
          <span className="font-bold text-lg">TeamWERK</span>
        </NavLink>
        {isMobile && (
          <button onClick={closeSidebar} aria-label="Schließen" className="text-brand-black/60 hover:text-brand-black transition-colors">
            <X className="w-5 h-5" />
          </button>
        )}
      </div>
      <nav className="flex-1 py-4">
        {navModules.map(mod => {
          const visibleItems = mod.items.filter(item => {
            if (!user) return false
            if (item.excludeRoles?.some(r => r === 'admin' ? user.role === 'admin' : hasFunction(user, r))) return false
            return item.roles.length === 0 || item.roles.some(r => r === 'admin' ? user.role === 'admin' : hasFunction(user, r))
          })
          if (visibleItems.length === 0) return null
          const isModuleActive = visibleItems.some(item => location.pathname.startsWith(item.to))
          const isOpen = openModules[mod.label]
          return (
            <div key={mod.label}>
              <button
                onClick={() => toggleModule(mod.label)}
                className={`px-4 py-2 w-full text-left flex items-center justify-between text-xs font-semibold uppercase tracking-wider ${isModuleActive ? 'text-brand-black' : 'text-brand-black/40'}`}
              >
                {mod.label}
                {isOpen
                  ? <ChevronDown className="w-4 h-4" />
                  : <ChevronRight className="w-4 h-4" />
                }
              </button>
              {isOpen && visibleItems.map(item => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  onClick={closeSidebar}
                  className={({ isActive }) =>
                    `flex items-center justify-between pl-7 pr-4 py-2 text-sm transition-colors ${isActive ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-black/60 hover:bg-brand-black hover:text-brand-yellow'}`
                  }
                >
                  <span>{item.label}</span>
                  {item.to === '/chat' && chatUnread > 0 && (
                    <span className="bg-brand-yellow text-brand-black text-xs font-bold rounded-full px-1.5 py-0.5 leading-none">
                      {chatUnread}
                    </span>
                  )}
                </NavLink>
              ))}
              {isOpen && mod.label === 'Nutzer' && navChildren.map(child => (
                <NavLink
                  key={`kind-${child.id}`}
                  to={`/profil/kind/${child.id}`}
                  onClick={closeSidebar}
                  className={({ isActive }) =>
                    `block pl-10 pr-4 py-2 text-sm transition-colors ${isActive ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-black/60 hover:bg-brand-black hover:text-brand-yellow'}`
                  }
                >
                  {child.first_name}
                </NavLink>
              ))}
            </div>
          )
        })}
      </nav>
      <div className="px-4 py-4 border-t border-brand-black/10 text-xs">
        <div className="truncate mb-2 text-brand-black/40">{user?.email}</div>
        <button onClick={handleLogout} className="text-brand-black/40 hover:text-brand-black transition-colors">
          Abmelden
        </button>
      </div>
      {version && (
        <div className="px-4 py-3 border-t border-brand-black/10">
          <span className="text-xs text-brand-black/40">v {version}</span>
        </div>
      )}
    </aside>
  )

  return (
    <div className="h-screen overflow-hidden flex bg-brand-gray">
      {/* Desktop sidebar */}
      <div className="hidden sm:flex">
        <Sidebar />
      </div>

      {/* Mobile sidebar overlay */}
      {isMobile && sidebarOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/40"
            onClick={closeSidebar}
          />
          <div className="fixed inset-y-0 left-0 z-50 w-56">
            <Sidebar />
          </div>
        </>
      )}

      <div className="flex-1 flex flex-col min-h-0 min-w-0">
        {/* Mobile header */}
        <header className="sm:hidden bg-brand-white border-b border-brand-black/10 px-4 py-4 flex items-center gap-3">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            aria-label="Menü öffnen"
            className="text-brand-black/60 hover:text-brand-black transition-colors"
          >
            <Menu className="w-6 h-6" />
          </button>
          <span className="font-bold text-lg">TeamWERK</span>
        </header>

        {/* Impersonation banner */}
        {impersonating && (
          <div className="bg-brand-yellow px-4 py-2 flex items-center gap-2 text-brand-black text-sm font-medium shrink-0">
            <Eye className="w-4 h-4 shrink-0" />
            <span className="flex-1">Admin-Vorschau: <strong>{impersonating.name}</strong></span>
            <button
              onClick={stopImpersonation}
              className="flex items-center gap-1 bg-brand-black text-brand-yellow rounded px-2 py-0.5 text-xs font-semibold hover:bg-brand-black/80 transition-colors"
            >
              <X className="w-3 h-3" />
              Beenden
            </button>
          </div>
        )}

        {/* Main content */}
        <main className="flex-1 px-4 py-4 sm:p-8 overflow-auto bg-brand-white sm:rounded-tl-3xl sm:rounded-bl-3xl sm:border-l-4 sm:border-brand-yellow">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
