import { useState } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { Menu, X, ChevronRight, ChevronDown } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'

interface NavModule {
  label: string
  items: { to: string; label: string; roles: string[]; end?: boolean }[]
}

const navModules: NavModule[] = [
  {
    label: 'Mitglieder',
    items: [
      { to: '/', label: 'Dashboard', roles: [], end: true },
      { to: '/mitglieder', label: 'Mitglieder', roles: ['admin', 'vorstand', 'trainer'] },
      { to: '/profil', label: 'Mein Profil', roles: ['elternteil', 'spieler'] },
    ],
  },
  {
    label: 'Dienste',
    items: [
      { to: '/kalender', label: 'Kalender', roles: [] },
      { to: '/dienste', label: 'Dienste', roles: [] },
    ],
  },
  {
    label: 'Dienstplanung',
    items: [
      { to: '/admin/diensttypen', label: 'Diensttypen', roles: ['admin', 'vorstand'] },
      { to: '/admin/dienstplan-vorlagen', label: 'Dienstplan-Vorlagen', roles: ['admin', 'vorstand'] },
    ],
  },
  {
    label: 'Kaderplanung',
    items: [
      { to: '/admin/verein', label: 'Verein', roles: ['admin', 'vorstand'] },
      { to: '/admin/altersklassen', label: 'Altersklassen', roles: ['admin', 'vorstand'] },
      { to: '/admin/kader', label: 'Kader', roles: ['admin', 'vorstand'] },
      { to: '/admin/saisons', label: 'Saisons', roles: ['admin', 'vorstand'] },
    ],
  },
  {
    label: 'Administration',
    items: [
      { to: '/admin/nutzer', label: 'Nutzerverwaltung', roles: ['admin', 'vorstand'] },
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

export default function AppShell() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const isMobile = useMediaQuery('(max-width: 639px)')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [openModules, setOpenModules] = useState<Record<string, boolean>>(initOpenModules)

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
    <aside className="w-56 bg-brand-gray text-brand-black flex flex-col">
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
          const visibleItems = mod.items.filter(item => user && (item.roles.length === 0 || item.roles.includes(user.role)))
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
                    `block pl-7 pr-4 py-2 text-sm transition-colors ${isActive ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-black/60 hover:bg-brand-black hover:text-brand-yellow'}`
                  }
                >
                  {item.label}
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
    </aside>
  )

  return (
    <div className="min-h-screen flex bg-brand-gray">
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

      <div className="flex-1 flex flex-col">
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

        {/* Main content */}
        <main className="flex-1 px-4 py-4 sm:p-8 overflow-auto bg-brand-white sm:rounded-tl-3xl sm:rounded-bl-3xl sm:border-l-4 sm:border-brand-yellow">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
