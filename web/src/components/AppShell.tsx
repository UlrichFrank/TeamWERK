import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

const navItems = [
  { to: '/mitglieder', label: 'Mitglieder', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
  { to: '/profil', label: 'Mein Profil', roles: ['elternteil', 'spieler'] },
  { to: '/dienstboerse', label: 'Dienstbörse', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
  { to: '/dienstkonten', label: 'Dienstkonten', roles: ['admin', 'trainer', 'elternteil', 'spieler'] },
  { to: '/dienste', label: 'Dienst-Planung', roles: ['admin', 'trainer'] },
  { to: '/anfragen', label: 'Beitrittsanfragen', roles: ['admin', 'trainer'] },
  { to: '/admin/verein', label: 'Verein', roles: ['admin'] },
  { to: '/admin/teams', label: 'Teams', roles: ['admin'] },
  { to: '/admin/nutzer', label: 'Nutzer', roles: ['admin'] },
  { to: '/admin/diensttypen', label: 'Diensttypen', roles: ['admin'] },
]

export default function AppShell() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  const visible = navItems.filter(item => user && item.roles.includes(user.role))

  return (
    <div className="min-h-screen flex bg-gray-50">
      <aside className="w-56 bg-brand-blue text-white flex flex-col">
        <div className="px-4 py-5 border-b border-white/20 flex items-center gap-3">
          <img src="/logo.svg" alt="Team Stuttgart" className="h-8 w-8" />
          <span className="font-bold text-lg">VereinsWerk</span>
        </div>
        <nav className="flex-1 py-4">
          {visible.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `block px-4 py-2 text-sm hover:bg-white/10 ${isActive ? 'bg-white/20 font-medium' : ''}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="px-4 py-4 border-t border-white/20 text-xs">
          <div className="truncate mb-2">{user?.email}</div>
          <button
            onClick={handleLogout}
            className="text-white/70 hover:text-white"
          >
            Abmelden
          </button>
        </div>
      </aside>
      <main className="flex-1 p-8 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
