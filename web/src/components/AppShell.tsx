import { useState } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

interface NavModule {
  label: string
  items: { to: string; label: string; roles: string[] }[]
}

const navModules: NavModule[] = [
  {
    label: 'Mitglieder',
    items: [
      { to: '/mitglieder', label: 'Mitglieder', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
      { to: '/profil', label: 'Mein Profil', roles: ['elternteil', 'spieler'] },
    ],
  },
  {
    label: 'Dienste',
    items: [
      { to: '/dienstboerse', label: 'Dienstbörse', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
      { to: '/dienstkonten', label: 'Dienstkonten', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
      { to: '/dienste', label: 'Dienst-Planung', roles: ['admin', 'trainer'] },
    ],
  },
  {
    label: 'Administration',
    items: [
      { to: '/anfragen', label: 'Beitrittsanfragen', roles: ['admin', 'trainer'] },
      { to: '/admin/verein', label: 'Verein', roles: ['admin'] },
      { to: '/admin/teams', label: 'Teams', roles: ['admin'] },
      { to: '/admin/nutzer', label: 'Nutzer', roles: ['admin'] },
      { to: '/admin/diensttypen', label: 'Diensttypen', roles: ['admin'] },
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

  return (
    <div className="min-h-screen flex bg-brand-gray">
      <aside className="w-56 bg-brand-gray text-black flex flex-col">
        <div className="px-4 py-5 border-b border-black/10 flex items-center gap-3">
          <img src="/logo.svg" alt="Team Stuttgart" className="h-8 w-8" />
          <span className="font-bold text-lg">TeamWERK</span>
        </div>
        <nav className="flex-1 py-4">
          {navModules.map(mod => {
            const visibleItems = mod.items.filter(item => user && item.roles.includes(user.role))
            if (visibleItems.length === 0) return null
            const isModuleActive = visibleItems.some(item => location.pathname.startsWith(item.to))
            const isOpen = openModules[mod.label]
            return (
              <div key={mod.label}>
                <button
                  onClick={() => toggleModule(mod.label)}
                  className={`px-4 py-2 w-full text-left flex items-center justify-between text-xs font-semibold uppercase tracking-wider ${isModuleActive ? 'text-black' : 'text-black/40'}`}
                >
                  {mod.label}
                  <span>{isOpen ? '▾' : '▸'}</span>
                </button>
                {isOpen && visibleItems.map(item => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      `block pl-7 pr-4 py-2 text-sm transition-colors ${isActive ? 'bg-brand-yellow text-black font-medium' : 'text-black/60 hover:bg-black hover:text-brand-yellow'}`
                    }
                  >
                    {item.label}
                  </NavLink>
                ))}
              </div>
            )
          })}
        </nav>
        <div className="px-4 py-4 border-t border-black/10 text-xs">
          <div className="truncate mb-2 text-black/40">{user?.email}</div>
          <button onClick={handleLogout} className="text-black/40 hover:text-black transition-colors">
            Abmelden
          </button>
        </div>
      </aside>
      <main className="flex-1 p-8 overflow-auto bg-gray-50 rounded-tl-3xl rounded-bl-3xl border-l-4 border-brand-yellow">
        <Outlet />
      </main>
    </div>
  )
}
