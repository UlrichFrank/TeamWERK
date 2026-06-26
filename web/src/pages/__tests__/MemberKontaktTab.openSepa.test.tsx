/**
 * MemberKontaktTab — „Dokument öffnen" navigiert zur In-App-Viewer-Route
 * statt window.open. Schützt vor dem PWA-Standalone-Sackgassen-Bug.
 */
import { describe, test, expect, vi } from 'vitest'
import { fireEvent, screen, render } from '@testing-library/react'
import { MemoryRouter, Routes, Route, useLocation } from 'react-router-dom'
import { AuthContext, type AuthCtx, type User } from '../../contexts/AuthContext'
import MemberKontaktTab from '../../components/admin/MemberKontaktTab'

const FORM = {
  iban: '',
  account_holder: '',
  beitragsfrei: false,
  sepa_mandat: true,
  sepa_mandat_date: '2025-01-01',
  sepa_mandat_url: 'https://example.com/sepa.pdf',
}

const noop = async () => {}

const ADMIN_USER: User = {
  id: 1,
  email: 'a@b.de',
  role: 'admin',
  clubFunctions: [],
  isParent: false,
}

const ADMIN_CTX: AuthCtx = {
  user: ADMIN_USER,
  loading: false,
  impersonating: null,
  mapsProvider: 'auto',
  setMapsProvider: () => {},
  capabilities: ['manage_members'],
  hasCapability: () => true,
  navRoutes: [],
  passwordChangeRecommended: false,
  dismissPasswordChangeHint: () => {},
  login: async () => {},
  logout: async () => {},
  startImpersonation: async () => {},
  stopImpersonation: async () => {},
}

function LocationProbe() {
  const loc = useLocation()
  return <div data-testid="path">{loc.pathname}</div>
}

describe('MemberKontaktTab — Dokument öffnen', () => {
  test('navigiert auf /mitglieder/:id/sepa-mandat/anzeigen (kein window.open)', () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    render(
      <AuthContext.Provider value={ADMIN_CTX}>
        <MemoryRouter initialEntries={['/mitglieder/42']}>
          <Routes>
            <Route
              path="/mitglieder/:id"
              element={
                <MemberKontaktTab
                  memberId={42}
                  form={FORM}
                  isNew={false}
                  drafts={[]}
                  onFormChange={() => {}}
                  onDraftAccept={noop}
                  onDraftReject={noop}
                  onSave={noop}
                  saving={false}
                  saved={false}
                  error=""
                />
              }
            />
            <Route
              path="/mitglieder/:id/sepa-mandat/anzeigen"
              element={<LocationProbe />}
            />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    )

    fireEvent.click(screen.getByText('Dokument öffnen'))
    expect(screen.getByTestId('path').textContent).toBe('/mitglieder/42/sepa-mandat/anzeigen')
    expect(openSpy).not.toHaveBeenCalled()
    openSpy.mockRestore()
  })
})
