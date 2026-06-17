import { useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useRegisterSW } from 'virtual:pwa-register/react'
import { AuthProvider, useAuth, hasFunction } from './contexts/AuthContext'
import { PersonContactProvider } from './contexts/PersonContactContext'
import { VersionProvider, useVersion } from './contexts/VersionContext'
import { UpdateBanner } from './components/UpdateBanner'
import { reloadWithSwActivation } from './lib/reload'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import RequestMembershipPage from './pages/RequestMembershipPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import AppShell from './components/AppShell'
import DashboardPage from './pages/DashboardPage'
import MembersPage from './pages/MembersPage'
import MemberDetailPage from './pages/MemberDetailPage'
import ProfilePage from './pages/ProfilePage'
import ChildProfilePage from './pages/ChildProfilePage'
import DutyPage from './pages/DutyPage'
import AdminSettingsPage from './pages/AdminSettingsPage'
import AdminUsersPage from './pages/AdminUsersPage'
import AdminDutyTypesPage from './pages/AdminDutyTypesPage'
import KalenderPage from './pages/KalenderPage'
import AdminDutyTemplatesPage from './pages/AdminDutyTemplatesPage'
import AdminDutyTemplateDetailPage from './pages/AdminDutyTemplateDetailPage'
import AdminKaderPage from './pages/AdminKaderPage'
import MitfahrgelegenheitenPage from './pages/MitfahrgelegenheitenPage'
import DocumentsPage from './pages/DocumentsPage'
import TerminePage from './pages/TerminePage'
import TermineDetailPage from './pages/TermineDetailPage'
import MeinTeamPage from './pages/MeinTeamPage'
import AdminVenuesPage from './pages/AdminVenuesPage'
import ChatPage from './pages/ChatPage'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="flex items-center justify-center h-screen">Laden…</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

// `roles` ist polymorph: 'admin' wird gegen die System-Rolle (users.role) geprüft,
// alle anderen Strings (z.B. 'trainer', 'vorstand') gegen die Vereinsfunktionen des Users.
// Die System-Rolle 'standard' wird nicht direkt benannt — sie ist der Default und ergibt
// sich aus der Abwesenheit von 'admin'. Siehe docs/berechtigungen.md.
export function RoleRoute({ roles, children }: { roles: string[]; children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="flex items-center justify-center h-screen">Laden…</div>
  const allowed = user && roles.some(r => r === 'admin' ? user.role === 'admin' : hasFunction(user, r))
  if (!allowed) return <Navigate to="/" replace />
  return <>{children}</>
}

// Sentinel für SW-Updates ohne bekannten Hash (z.B. wenn nur useRegisterSW
// triggert, ohne dass SSE eine neue Version gemeldet hat).
const SW_BANNER_SENTINEL = '__sw__'

function AppUpdateBanner() {
  const { updateAvailable: sseUpdateAvailable, latestVersion } = useVersion()
  const [swUpdateAvailable, setSwUpdateAvailable] = useState(false)
  const [dismissedVersion, setDismissedVersion] = useState<string | null>(null)

  useRegisterSW({ onNeedRefresh() { setSwUpdateAvailable(true) } })

  const shouldShow = sseUpdateAvailable || swUpdateAvailable
  if (!shouldShow) return null

  // currentBanneredVersion: der Hash, der diesen Banner triggert. Bevorzugt
  // der vom Server zuletzt gemeldete; Sentinel wenn nur SW-Pfad aktiv.
  const currentBanneredVersion = sseUpdateAvailable
    ? (latestVersion ?? SW_BANNER_SENTINEL)
    : SW_BANNER_SENTINEL
  if (dismissedVersion === currentBanneredVersion) return null

  return (
    <UpdateBanner
      onReload={reloadWithSwActivation}
      onDismiss={() => setDismissedVersion(currentBanneredVersion)}
    />
  )
}

export default function App() {
  return (
    <AuthProvider>
      <VersionProvider>
        <PersonContactProvider>
          <BrowserRouter>
          <AppUpdateBanner />
          <Routes>
            {/* Public */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/join" element={<RequestMembershipPage />} />
            <Route path="/passwort-vergessen" element={<ForgotPasswordPage />} />
            <Route path="/reset-password" element={<ResetPasswordPage />} />

            {/* Protected */}
            <Route path="/" element={<PrivateRoute><AppShell /></PrivateRoute>}>
              <Route index element={<DashboardPage />} />
              <Route path="mitglieder" element={<RoleRoute roles={['admin','vorstand']}><MembersPage /></RoleRoute>} />
              <Route path="mitglieder/:id" element={<RoleRoute roles={['admin','vorstand']}><MemberDetailPage /></RoleRoute>} />
              <Route path="profil" element={<ProfilePage />} />
              <Route path="profil/kind/:memberId" element={<ChildProfilePage />} />
              <Route path="dokumente" element={<DocumentsPage />} />
              <Route path="dokumente/:folderId" element={<DocumentsPage />} />
              <Route path="dienste" element={<DutyPage />} />
              <Route path="mitfahrgelegenheiten" element={<MitfahrgelegenheitenPage />} />
              <Route path="anfragen" element={<RoleRoute roles={['admin','vorstand']}><AdminUsersPage /></RoleRoute>} />
              <Route path="einstellungen" element={<RoleRoute roles={['admin','vorstand']}><AdminSettingsPage /></RoleRoute>} />
              <Route path="kader" element={<RoleRoute roles={['admin','vorstand','trainer','sportliche_leitung']}><AdminKaderPage /></RoleRoute>} />
              <Route path="nutzer" element={<RoleRoute roles={['admin','vorstand']}><AdminUsersPage /></RoleRoute>} />
              <Route path="diensttypen" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTypesPage /></RoleRoute>} />
              <Route path="kalender" element={<KalenderPage />} />
              <Route path="termine" element={<TerminePage />} />
              <Route path="termine/:type/:id" element={<TermineDetailPage />} />
              <Route path="mein-team" element={<MeinTeamPage />} />
              <Route path="chat" element={<ChatPage />} />
              <Route path="trainings" element={<Navigate to="/termine" replace />} />
              <Route path="trainings/:id" element={<Navigate to="/termine" replace />} />
              <Route path="dienstplan-vorlagen" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTemplatesPage /></RoleRoute>} />
              <Route path="dienstplan-vorlagen/:id" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTemplateDetailPage /></RoleRoute>} />
              <Route path="veranstaltungsorte" element={<RoleRoute roles={['admin','vorstand']}><AdminVenuesPage /></RoleRoute>} />
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
        </PersonContactProvider>
      </VersionProvider>
    </AuthProvider>
  )
}
