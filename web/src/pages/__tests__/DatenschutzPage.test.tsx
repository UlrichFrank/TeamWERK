import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route, Navigate } from 'react-router-dom'
import DatenschutzPage from '../DatenschutzPage'
import LoginPage from '../LoginPage'
import { AuthContext, type AuthCtx } from '../../contexts/AuthContext'

// Minimaler Ctx ohne eingeloggten User — simuliert öffentlichen Zugriff.
const anonymousCtx: AuthCtx = {
  user: null,
  loading: false,
  impersonating: null,
  mapsProvider: 'auto',
  setMapsProvider: () => {},
  capabilities: [],
  hasCapability: () => false,
  navRoutes: [],
  passwordChangeRecommended: false,
  dismissPasswordChangeHint: () => {},
  login: async () => {},
  logout: async () => {},
  startImpersonation: async () => {},
  stopImpersonation: async () => {},
}

describe('DatenschutzPage', () => {
  test('rendert auch ohne eingeloggten User (Public-Route)', () => {
    render(
      <AuthContext.Provider value={anonymousCtx}>
        <MemoryRouter initialEntries={['/datenschutz']}>
          <DatenschutzPage />
        </MemoryRouter>
      </AuthContext.Provider>,
    )
    expect(screen.getByRole('heading', { level: 1, name: /Datenschutzerklärung/i })).toBeInTheDocument()
  })

  test('enthält Matomo-Absatz mit Pflicht-Inhalten', () => {
    render(
      <AuthContext.Provider value={anonymousCtx}>
        <MemoryRouter><DatenschutzPage /></MemoryRouter>
      </AuthContext.Provider>,
    )
    expect(screen.getByRole('heading', { name: /Matomo/i })).toBeInTheDocument()
    const matomoSection = screen.getByRole('heading', { name: /Matomo/i }).parentElement?.parentElement
    expect(matomoSection?.textContent ?? '').toMatch(/anonym/i)
    expect(matomoSection?.textContent ?? '').toMatch(/IP/i)
    expect(matomoSection?.textContent ?? '').toMatch(/Do Not Track/i)
    expect(matomoSection?.textContent ?? '').toMatch(/mittwald/i)
    expect(matomoSection?.textContent ?? '').toMatch(/keine Cookies/i)
  })

  test('erwähnt anonyme Behandlung von Kinder-Accounts', () => {
    render(
      <AuthContext.Provider value={anonymousCtx}>
        <MemoryRouter><DatenschutzPage /></MemoryRouter>
      </AuthContext.Provider>,
    )
    expect(screen.getByText(/Kinder-Accounts/i)).toBeInTheDocument()
  })

  test('Route /datenschutz ist ohne Auth erreichbar (kein Redirect auf /login)', () => {
    // Simuliert Route-Konfig: /datenschutz ist Public, /login auch. Wir bestätigen,
    // dass auf /datenschutz die DatenschutzPage gerendert wird, nicht LoginPage.
    render(
      <AuthContext.Provider value={anonymousCtx}>
        <MemoryRouter initialEntries={['/datenschutz']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/datenschutz" element={<DatenschutzPage />} />
            <Route path="*" element={<Navigate to="/login" replace />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    )
    expect(screen.queryByRole('heading', { level: 1, name: /Datenschutzerklärung/i })).toBeInTheDocument()
    expect(screen.queryByText('Anmelden')).not.toBeInTheDocument()
  })
})
