import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import { PersonContactProvider } from './contexts/PersonContactContext'
import { VersionProvider } from './contexts/VersionContext'
import { VaultProvider } from './contexts/VaultContext'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import RequestMembershipPage from './pages/RequestMembershipPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import DatenschutzPage from './pages/DatenschutzPage'
import AppShell from './components/AppShell'
import DashboardPage from './pages/DashboardPage'
import MembersPage from './pages/MembersPage'
import MemberDetailPage from './pages/MemberDetailPage'
import ProfilePage from './pages/ProfilePage'
import ChildProfilePage from './pages/ChildProfilePage'
import DutyPage from './pages/DutyPage'
const DutyInstructionPage = lazy(() => import('./pages/DutyInstructionPage'))
import AdminSettingsPage from './pages/AdminSettingsPage'
import BeitragslaufPage from './pages/admin/BeitragslaufPage'
import TresorPage from './pages/admin/TresorPage'
import WartungsmodusPage from './pages/admin/WartungsmodusPage'
import AdminUsersPage from './pages/AdminUsersPage'
import AdminDutyTypesPage from './pages/AdminDutyTypesPage'
import KalenderPage from './pages/KalenderPage'
import AdminDutyTemplatesPage from './pages/AdminDutyTemplatesPage'
import AdminDutyTemplateDetailPage from './pages/AdminDutyTemplateDetailPage'
import AdminKaderPage from './pages/AdminKaderPage'
import MitfahrgelegenheitenPage from './pages/MitfahrgelegenheitenPage'
import DocumentsPage from './pages/DocumentsPage'
import DocumentFileLinkPage from './pages/DocumentFileLinkPage'
import FileViewerPage from './pages/FileViewerPage'
import SepaMandatViewerPage from './pages/SepaMandatViewerPage'
import TerminePage from './pages/TerminePage'
import TermineDetailPage from './pages/TermineDetailPage'
import MeinTeamPage from './pages/MeinTeamPage'
import AdminVenuesPage from './pages/AdminVenuesPage'
import ChatPage from './pages/ChatPage'
import MatchReportListPage from './pages/MatchReportListPage'
import MatchReportFormPage from './pages/MatchReportFormPage'
import MatchReportPendingListPage from './pages/MatchReportPendingListPage'
import VideosPage from './pages/VideosPage'
import VideoUploadPage from './pages/VideoUploadPage'
import VideoDetailPage from './pages/VideoDetailPage'
import TeamAnwesenheitPage from './pages/TeamAnwesenheitPage'
import ProfilAnwesenheitPage from './pages/ProfilAnwesenheitPage'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  const location = useLocation()
  if (loading) return <div className="flex items-center justify-center h-screen">Laden…</div>
  if (!user) {
    const next = location.pathname + location.search
    const target = next && next !== '/' ? `/login?next=${encodeURIComponent(next)}` : '/login'
    return <Navigate to={target} replace />
  }
  return <>{children}</>
}

// `roles` ist polymorph: System-Rollen werden gegen `user.role` geprüft
// (users.role), Vereinsfunktionen gegen `user.clubFunctions`. Die
// System-Rolle 'standard' wird nicht benannt — sie ist der Default und
// ergibt sich aus der Abwesenheit einer eskalierten Rolle.
// Siehe docs/berechtigungen.md.
const SYSTEM_ROLES = new Set(['admin', 'presseteam'])

export function RoleRoute({ roles, children }: { roles: string[]; children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="flex items-center justify-center h-screen">Laden…</div>
  const allowed = user && roles.some(r =>
    SYSTEM_ROLES.has(r) ? user.role === r : user.clubFunctions?.includes(r)
  )
  if (!allowed) return <Navigate to="/" replace />
  return <>{children}</>
}


export default function App() {
  return (
    <AuthProvider>
      <VersionProvider>
        <PersonContactProvider>
          <VaultProvider>
          <BrowserRouter>
          <Routes>
            {/* Public */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/join" element={<RequestMembershipPage />} />
            <Route path="/passwort-vergessen" element={<ForgotPasswordPage />} />
            <Route path="/reset-password" element={<ResetPasswordPage />} />
            <Route path="/datenschutz" element={<DatenschutzPage />} />

            {/* Protected */}
            <Route path="/" element={<PrivateRoute><AppShell /></PrivateRoute>}>
              <Route index element={<DashboardPage />} />
              <Route path="mitglieder" element={<RoleRoute roles={['admin','vorstand','kassierer']}><MembersPage /></RoleRoute>} />
              <Route path="mitglieder/:id" element={<RoleRoute roles={['admin','vorstand','kassierer']}><MemberDetailPage /></RoleRoute>} />
              <Route path="profil" element={<ProfilePage />} />
              <Route path="profil/anwesenheit" element={<ProfilAnwesenheitPage />} />
              <Route path="profil/kind/:memberId" element={<ChildProfilePage />} />
              <Route path="anwesenheit" element={<RoleRoute roles={['admin','trainer','sportliche_leitung']}><TeamAnwesenheitPage /></RoleRoute>} />
              <Route path="team/:id/anwesenheit" element={<RoleRoute roles={['admin','trainer','sportliche_leitung']}><TeamAnwesenheitPage /></RoleRoute>} />
              <Route path="dokumente" element={<DocumentsPage />} />
              <Route path="dokumente/datei/:fileId" element={<DocumentFileLinkPage />} />
              <Route path="dokumente/anzeigen/:fileId" element={<FileViewerPage />} />
              <Route path="dokumente/:folderId" element={<DocumentsPage />} />
              <Route path="mitglieder/:memberId/sepa-mandat/anzeigen" element={<RoleRoute roles={['admin','vorstand','kassierer']}><SepaMandatViewerPage /></RoleRoute>} />
              <Route path="dienste" element={<DutyPage />} />
              <Route path="dienste/anleitung/:typeId" element={
                <Suspense fallback={<p className="text-sm text-brand-text-muted">Lade Anleitung…</p>}>
                  <DutyInstructionPage />
                </Suspense>
              } />
              <Route path="mitfahrgelegenheiten" element={<MitfahrgelegenheitenPage />} />
              <Route path="anfragen" element={<RoleRoute roles={['admin','vorstand']}><AdminUsersPage /></RoleRoute>} />
              <Route path="einstellungen" element={<RoleRoute roles={['admin','vorstand','kassierer']}><AdminSettingsPage /></RoleRoute>} />
              <Route path="beitragslauf" element={<RoleRoute roles={['admin','vorstand','kassierer']}><BeitragslaufPage /></RoleRoute>} />
              <Route path="tresor" element={<RoleRoute roles={['admin','vorstand','kassierer']}><TresorPage /></RoleRoute>} />
              <Route path="kader" element={<RoleRoute roles={['admin','vorstand','trainer','sportliche_leitung']}><AdminKaderPage /></RoleRoute>} />
              <Route path="nutzer" element={<RoleRoute roles={['admin','vorstand']}><AdminUsersPage /></RoleRoute>} />
              <Route path="diensttypen" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTypesPage /></RoleRoute>} />
              <Route path="kalender" element={<KalenderPage />} />
              <Route path="termine" element={<TerminePage />} />
              <Route path="termine/:type/:id" element={<TermineDetailPage />} />
              <Route path="mein-team" element={<MeinTeamPage />} />
              <Route path="chat" element={<ChatPage />} />
              <Route path="spielberichte" element={<RoleRoute roles={['presseteam','admin']}><MatchReportListPage /></RoleRoute>} />
              <Route path="spielberichte/pruefen" element={<RoleRoute roles={['admin','medien','vorstand']}><MatchReportPendingListPage /></RoleRoute>} />
              <Route path="spielberichte/:id" element={<RoleRoute roles={['presseteam','admin','medien','vorstand']}><MatchReportFormPage /></RoleRoute>} />
              <Route path="videos" element={<VideosPage />} />
              <Route path="videos/upload" element={<RoleRoute roles={['admin','trainer','sportliche_leitung','vorstand']}><VideoUploadPage /></RoleRoute>} />
              <Route path="videos/:id" element={<VideoDetailPage />} />
              <Route path="trainings" element={<Navigate to="/termine" replace />} />
              <Route path="trainings/:id" element={<Navigate to="/termine" replace />} />
              <Route path="dienstplan-vorlagen" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTemplatesPage /></RoleRoute>} />
              <Route path="dienstplan-vorlagen/:id" element={<RoleRoute roles={['admin','vorstand']}><AdminDutyTemplateDetailPage /></RoleRoute>} />
              <Route path="veranstaltungsorte" element={<RoleRoute roles={['admin','vorstand']}><AdminVenuesPage /></RoleRoute>} />
              <Route path="wartung" element={<RoleRoute roles={['admin']}><WartungsmodusPage /></RoleRoute>} />
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
          </VaultProvider>
        </PersonContactProvider>
      </VersionProvider>
    </AuthProvider>
  )
}
