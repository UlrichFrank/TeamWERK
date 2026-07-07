import { useState, useEffect, useCallback } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useRegisterSW } from 'virtual:pwa-register/react'
import { Menu, X, Eye, RefreshCw, ChevronDown, ChevronRight, ChevronLeft, AlertTriangle } from 'lucide-react'
import ChangelogModal from './ChangelogModal'
import TransitionalHostnameBanner from './TransitionalHostnameBanner'
import MaintenanceBanner from './MaintenanceBanner'
import { useAuth } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import { usePushSubscription } from '../hooks/usePushSubscription'
import { useChatEvents } from '../hooks/useChatEvents'
import { useVersion } from '../contexts/VersionContext'
import { reloadWithSwActivation } from '../lib/reload'
import { api, setMaintenanceHandler } from '../lib/api'
import {
  setChannelDimension,
  setTeamSlugDimension,
  setRoleDimension,
  slugifyTeam,
  trackPageview,
} from '../lib/telemetry'

interface NavModule {
  label: string
  items: { to: string; label: string; end?: boolean }[]
}

// Static layout descriptor — defines grouping, labels, and ordering.
// Visibility is determined server-side via GET /api/me → nav routes.
const navModules: NavModule[] = [
  {
    label: 'Nutzer',
    items: [
      { to: '/', label: 'Dashboard', end: true },
      { to: '/profil', label: 'Mein Profil' },
    ],
  },
  {
    label: 'Spielbetrieb',
    items: [
      { to: '/kalender', label: 'Kalender' },
      { to: '/termine', label: 'Termine' },
      { to: '/videos', label: 'Videos' },
      { to: '/anwesenheit', label: 'Anwesenheit' },
    ],
  },
  {
    label: 'Verein',
    items: [
      { to: '/mein-team', label: 'Mein Team' },
      { to: '/dokumente', label: 'Dokumente' },
      { to: '/dienste', label: 'Dienste' },
      { to: '/mitfahrgelegenheiten', label: 'Mitfahrten' },
      { to: '/chat', label: 'Nachrichten' },
      { to: '/spielberichte', label: 'Spielberichte' },
      { to: '/berichte/pruefen', label: 'Berichte prüfen' },
    ],
  },
  {
    label: 'Verwaltung',
    items: [
      { to: '/nutzer', label: 'Nutzerverwaltung' },
      { to: '/mitglieder', label: 'Mitglieder' },
      { to: '/kader', label: 'Kader' },
      { to: '/diensttypen', label: 'Diensttypen' },
      { to: '/dienstplan-vorlagen', label: 'Dienstplan-Vorlagen' },
      { to: '/veranstaltungsorte', label: 'Veranstaltungsorte' },
      { to: '/beitragslauf', label: 'Beitragslauf' },
      { to: '/tresor', label: 'Tresor' },
      { to: '/einstellungen', label: 'Einstellungen' },
      { to: '/wartung', label: 'Wartungsmodus' },
    ],
  },
]

// Label des Moduls, das die aktuelle Route enthält (für Akkordeon-Default + aktive Hervorhebung).
function activeModuleLabel(pathname: string): string | null {
  for (const m of navModules) {
    if (m.items.some(item => (item.end ? pathname === item.to : pathname.startsWith(item.to)))) {
      return m.label
    }
  }
  return null
}

// Akkordeon: genau ein offenes Modul ('' = alle zu). Beim Start das Modul der aktuellen
// Route öffnen, sonst den zuletzt gemerkten Zustand, sonst das erste Modul.
function initOpenModule(): string {
  const stored = localStorage.getItem('nav-open-module')
  if (stored !== null) return stored
  return activeModuleLabel(window.location.pathname) ?? navModules[0].label
}

interface ChildEntry { id: number; first_name: string; last_name: string }

export default function AppShell() {
  const { user, loading, logout, impersonating, stopImpersonation, navRoutes: navRouteList, passwordChangeRecommended, dismissPasswordChangeHint } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const isMobile = useMediaQuery('(max-width: 639px)')
  usePushSubscription()
  const { version, updateAvailable: sseUpdateAvailable } = useVersion()
  const [swUpdateAvailable, setSwUpdateAvailable] = useState(false)
  useRegisterSW({ onNeedRefresh() { setSwUpdateAvailable(true) } })
  const showUpdateBanner = sseUpdateAvailable || swUpdateAvailable
  const [canGoBack, setCanGoBack] = useState(() => (window.history.state?.idx ?? 0) > 0)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [showChangelog, setShowChangelog] = useState(false)
  const [openModule, setOpenModule] = useState<string>(initOpenModule)
  const [navChildren, setNavChildren] = useState<ChildEntry[]>([])
  const navRoutes = new Set(navRouteList)
  const [chatUnread, setChatUnread] = useState(0)
  // Kurzer Overlay-Hinweis, wenn ein Mutations-Request durch die
  // Maintenance-Middleware mit 503 abgewiesen wurde. Der Banner (persistent
  // oben) bleibt getrennt sichtbar — dieser Toast reagiert auf den konkreten
  // Klick, damit der Nutzer die Wartungssperre in Verbindung mit seiner
  // Aktion sieht.
  const [maintenanceToast, setMaintenanceToast] = useState(false)
  useEffect(() => {
    let timer: number | undefined
    setMaintenanceHandler(() => {
      setMaintenanceToast(true)
      window.clearTimeout(timer)
      timer = window.setTimeout(() => setMaintenanceToast(false), 5000)
    })
    return () => {
      setMaintenanceHandler(null)
      window.clearTimeout(timer)
    }
  }, [])
  // Matomo team_slug Custom Dimension. 'none' = kein Team; 'unknown' = Endpoint-Fehler; 'mixed' = mehrere Teams.
  const [teamSlug, setTeamSlug] = useState<string>('none')

  const loadChatUnread = useCallback(async () => {
    if (!user) return
    try {
      const [convs, bcs] = await Promise.all([
        api.get('/chat/conversations'),
        api.get('/chat/broadcasts'),
      ])
      const convUnread = (convs.data ?? []).reduce((s: number, c: { unreadCount?: number }) => s + (c.unreadCount ?? 0), 0)
      const bcUnread = (bcs.data ?? []).filter((b: { isRead?: boolean; isSent?: boolean }) => !b.isRead && !b.isSent).length
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
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    loadChatUnread()
    // Effekt soll nur bei Wechsel der Nutzer-Identität laufen (user?.id), nicht bei jeder user-Objektreferenz
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id, loadChatUnread])

  useChatEvents((event) => {
    if (event.startsWith('chat:new-message') || event === 'chat:new-broadcast' || event === 'chat:conversation-read') {
      loadChatUnread()
    }
  })

  useEffect(() => {
    if (!user) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setTeamSlug('none')
      return
    }
    let cancelled = false
    api.get('/teams/my')
      .then(r => {
        if (cancelled) return
        const teams = (r.data ?? []) as { name?: string }[]
        if (teams.length === 0) setTeamSlug('none')
        else if (teams.length === 1 && teams[0].name) setTeamSlug(slugifyTeam(teams[0].name))
        else setTeamSlug('mixed')
      })
      .catch(() => {
        if (!cancelled) setTeamSlug('unknown')
      })
    return () => { cancelled = true }
    // Wie der Profil-Loader oben: nur bei Wechsel der Nutzer-Identität neu laden.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id])

  useEffect(() => {
    if (loading || !user) return
    setChannelDimension()
    setTeamSlugDimension(teamSlug)
    setRoleDimension(user.role)
    trackPageview(window.location.href, document.title)
  }, [location.pathname, location.search, loading, user, teamSlug])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    setCanGoBack((window.history.state?.idx ?? 0) > 0)
  }, [location])

  useEffect(() => {
    const nav = navigator as Navigator & {
      setAppBadge?: (n?: number) => Promise<void>
      clearAppBadge?: () => Promise<void>
    }
    if (!('setAppBadge' in nav)) return
    if (chatUnread > 0) {
      nav.setAppBadge?.(chatUnread).catch(() => {})
    } else {
      nav.clearAppBadge?.().catch(() => {})
    }
  }, [chatUnread])

  useEffect(() => {
    if (user) return
    const nav = navigator as Navigator & { clearAppBadge?: () => Promise<void> }
    if ('clearAppBadge' in nav) {
      nav.clearAppBadge?.().catch(() => {})
    }
  }, [user])

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  // Akkordeon: das geklickte Modul öffnen und alle anderen schließen; ein erneuter Klick
  // auf das offene Modul klappt es zu ('').
  const toggleModule = (label: string) => {
    setOpenModule(prev => {
      const next = prev === label ? '' : label
      localStorage.setItem('nav-open-module', next)
      return next
    })
  }

  // Bei Navigation (auch per Direktlink/navigate) das Modul der aktiven Route aufklappen,
  // damit der aktive Eintrag sichtbar ist und die Akkordeon-Auswahl der Seite folgt.
  useEffect(() => {
    const active = activeModuleLabel(location.pathname)
    if (active) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Sync an die Route (Akkordeon folgt der aktiven Seite)
      setOpenModule(active)
      localStorage.setItem('nav-open-module', active)
    }
  }, [location.pathname])

  const closeSidebar = () => setSidebarOpen(false)

  const sidebar = (
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
            // While /api/me is loading, fall back to showing all items to avoid flash
            if (navRoutes.size === 0) return true
            return navRoutes.has(item.to)
          })
          if (visibleItems.length === 0) return null
          const isModuleActive = visibleItems.some(item => location.pathname.startsWith(item.to))
          const isOpen = openModule === mod.label
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
      <div className="px-4 py-3 border-t border-brand-black/10">
        <a
          href="/benutzerhandbuch.html"
          target="_blank"
          rel="noopener"
          onClick={closeSidebar}
          className="text-xs text-brand-black/40 hover:text-brand-black/70 transition-colors"
        >
          Anleitung
        </a>
      </div>
      {version && (
        <div className="px-4 py-3 border-t border-brand-black/10">
          <button
            onClick={() => setShowChangelog(true)}
            className="text-xs text-brand-black/40 hover:text-brand-black/70 transition-colors"
          >
            v {version}
          </button>
        </div>
      )}
    </aside>
  )

  return (
    <div className="h-screen overflow-hidden flex flex-col">
      <MaintenanceBanner />
      <TransitionalHostnameBanner />
      <div className="flex-1 min-h-0 overflow-hidden flex bg-brand-gray">
      {/* Desktop sidebar */}
      <div className="hidden sm:flex">
        {sidebar}
      </div>

      {/* Mobile sidebar overlay */}
      {isMobile && sidebarOpen && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/40"
            onClick={closeSidebar}
          />
          <div className="fixed inset-y-0 left-0 z-50 w-56">
            {sidebar}
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
          {canGoBack && (
            <button
              onClick={() => navigate(-1)}
              aria-label="Zurück"
              className="text-brand-black/60 hover:text-brand-black transition-colors flex items-center gap-0.5"
            >
              <ChevronLeft className="w-5 h-5" />
              <span className="text-sm">Zurück</span>
            </button>
          )}
          <span className="font-bold text-lg">TeamWERK</span>
        </header>

        {/* Update banner */}
        {showUpdateBanner && (
          <div className="bg-brand-yellow text-brand-black text-sm font-medium shrink-0">
            <div className="px-4 py-2 flex items-center gap-2">
              <RefreshCw className="w-4 h-4 shrink-0" />
              <span className="flex-1">Neue Version verfügbar</span>
              <button
                onClick={() => setShowChangelog(true)}
                className="text-xs text-brand-black/60 hover:text-brand-black transition-colors"
              >
                Details
              </button>
              <button
                onClick={reloadWithSwActivation}
                className="bg-brand-black text-brand-yellow rounded px-2 py-0.5 text-xs font-semibold hover:bg-brand-black/80 transition-colors"
              >
                Jetzt laden
              </button>
            </div>
          </div>
        )}

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
          {canGoBack && (
            <div className="hidden sm:block -mt-2 mb-3">
              <button
                onClick={() => navigate(-1)}
                className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
              >
                <ChevronLeft className="w-4 h-4" /> Zurück
              </button>
            </div>
          )}
          {passwordChangeRecommended && (
            <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0 text-brand-info" />
              <div className="flex-1">
                Dein Passwort ist kürzer als die empfohlenen 12 Zeichen.{' '}
                <button
                  onClick={() => { dismissPasswordChangeHint(); navigate('/profil') }}
                  className="underline font-medium hover:text-brand-info"
                >
                  Jetzt ändern
                </button>
              </div>
              <button
                onClick={dismissPasswordChangeHint}
                aria-label="Hinweis schließen"
                className="text-brand-text-muted hover:text-brand-text shrink-0"
              >
                <X className="w-4 h-4" />
              </button>
            </div>
          )}
          <Outlet />
        </main>
      </div>

      {showChangelog && <ChangelogModal onClose={() => setShowChangelog(false)} />}
      {maintenanceToast && (
        <div
          role="alert"
          className="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 max-w-md bg-brand-danger-light border border-brand-danger/30 text-brand-text text-sm rounded-lg shadow-lg px-4 py-3 flex items-start gap-2"
        >
          <AlertTriangle className="w-5 h-5 shrink-0 text-brand-danger" />
          <p>Wartungsmodus aktiv — Änderungen sind gerade nicht möglich. Bitte gleich noch einmal versuchen.</p>
        </div>
      )}
      </div>
    </div>
  )
}
