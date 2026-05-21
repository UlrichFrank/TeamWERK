import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './contexts/AuthContext'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import RequestMembershipPage from './pages/RequestMembershipPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import AppShell from './components/AppShell'
import MembersPage from './pages/MembersPage'
import MemberDetailPage from './pages/MemberDetailPage'
import ProfilePage from './pages/ProfilePage'
import DutyBoardPage from './pages/DutyBoardPage'
import DutyAccountsPage from './pages/DutyAccountsPage'
import DutySlotsPage from './pages/DutySlotsPage'
import AdminClubPage from './pages/AdminClubPage'
import AdminTeamsPage from './pages/AdminTeamsPage'
import AdminUsersPage from './pages/AdminUsersPage'
import AdminDutyTypesPage from './pages/AdminDutyTypesPage'
import MembershipRequestsPage from './pages/MembershipRequestsPage'
import SpielplanPage from './pages/SpielplanPage'
import SpieltagDetailPage from './pages/SpieltagDetailPage'
import AdminGameTemplatePage from './pages/AdminGameTemplatePage'
import AdminSeasonsPage from './pages/AdminSeasonsPage'
import AdminKaderPage from './pages/AdminKaderPage'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="flex items-center justify-center h-screen">Laden…</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          {/* Public */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/join" element={<RequestMembershipPage />} />
          <Route path="/passwort-vergessen" element={<ForgotPasswordPage />} />
          <Route path="/reset-password" element={<ResetPasswordPage />} />

          {/* Protected */}
          <Route path="/" element={<PrivateRoute><AppShell /></PrivateRoute>}>
            <Route index element={<Navigate to="/mitglieder" replace />} />
            <Route path="mitglieder" element={<MembersPage />} />
            <Route path="mitglieder/:id" element={<MemberDetailPage />} />
            <Route path="profil" element={<ProfilePage />} />
            <Route path="dienstboerse" element={<DutyBoardPage />} />
            <Route path="dienstkonten" element={<DutyAccountsPage />} />
            <Route path="dienste" element={<DutySlotsPage />} />
            <Route path="anfragen" element={<MembershipRequestsPage />} />
            <Route path="admin/verein" element={<AdminClubPage />} />
            <Route path="admin/teams" element={<AdminTeamsPage />} />
            <Route path="admin/kader" element={<AdminKaderPage />} />
            <Route path="admin/nutzer" element={<AdminUsersPage />} />
            <Route path="admin/diensttypen" element={<AdminDutyTypesPage />} />
            <Route path="spielplan" element={<SpielplanPage />} />
            <Route path="spielplan/:gameId" element={<SpieltagDetailPage />} />
            <Route path="admin/spielplan-template" element={<AdminGameTemplatePage />} />
            <Route path="admin/saisons" element={<AdminSeasonsPage />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
