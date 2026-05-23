import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './contexts/AuthContext'
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
import DutyPage from './pages/DutyPage'
import AdminClubPage from './pages/AdminClubPage'
import AdminUsersPage from './pages/AdminUsersPage'
import AdminDutyTypesPage from './pages/AdminDutyTypesPage'
import KalenderPage from './pages/KalenderPage'
import SpieltagDetailPage from './pages/SpieltagDetailPage'
import AdminDutyTemplatesPage from './pages/AdminDutyTemplatesPage'
import AdminDutyTemplateDetailPage from './pages/AdminDutyTemplateDetailPage'
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
            <Route index element={<DashboardPage />} />
            <Route path="mitglieder" element={<MembersPage />} />
            <Route path="mitglieder/:id" element={<MemberDetailPage />} />
            <Route path="profil" element={<ProfilePage />} />
            <Route path="dienste" element={<DutyPage />} />
            <Route path="anfragen" element={<Navigate to="/admin/nutzer" replace />} />
            <Route path="admin/verein" element={<AdminClubPage />} />
            <Route path="admin/kader" element={<AdminKaderPage />} />
            <Route path="admin/nutzer" element={<AdminUsersPage />} />
            <Route path="admin/diensttypen" element={<AdminDutyTypesPage />} />
            <Route path="kalender" element={<KalenderPage />} />
            <Route path="kalender/:gameId" element={<SpieltagDetailPage />} />
            <Route path="admin/dienstplan-vorlagen" element={<AdminDutyTemplatesPage />} />
            <Route path="admin/dienstplan-vorlagen/:id" element={<AdminDutyTemplateDetailPage />} />
            <Route path="admin/saisons" element={<AdminSeasonsPage />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
