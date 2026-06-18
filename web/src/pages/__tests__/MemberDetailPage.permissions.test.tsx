/**
 * MemberDetailPage inline gate: isAdmin = admin || vorstand
 * Steuert die Tabs "Datenschutz", "Familie", "Admin" (nur bei vorhandenem Mitglied).
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 *
 * Hinweis: Nicht-vorstand-Personas landen per RoleRoute auf "/" (nie auf dieser Page).
 * Dieser Test prüft die defensive Logik der Page-Komponente selbst.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import MemberDetailPage from '../MemberDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const MEMBER_FIXTURE = {
  id: 1,
  first_name: 'Max',
  last_name: 'Mustermann',
  date_of_birth: '2000-01-01',
  status: 'aktiv',
  pass_number: 'P001',
  jersey_number: 7,
  position: 'MF',
  gender: 'm',
  phone: '',
  phone2: '',
  phone_emergency: '',
  iban: '',
  bic: '',
  account_holder: '',
  address: '',
  zip: '',
  city: '',
  notes: '',
  sepa_mandat: false,
  sepa_mandat_date: '',
  sepa_mandat_url: '',
  dsgvo_verarbeitung: false,
  dsgvo_weitergabe: false,
  club_functions: [],
  has_photo: false,
}

const ALLOWED_IDS = ['admin', 'vorstand', 'vorstand_elternteil']

describe('MemberDetailPage — isAdmin-Gate: "Datenschutz"-Tab', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<MemberDetailPage />, persona.id, {
      route: '/mitglieder/1',
      initialEntries: ['/mitglieder/1'],
      mocks: [
        { url: /members\/1$/, data: MEMBER_FIXTURE },
        { url: /members\/1\/change-drafts/, data: [] },
        { url: /invitations/, data: [] },
      ],
    })
    await flushAsync()

    const tab = screen.queryByRole('button', { name: 'Datenschutz' })
    if (ALLOWED_IDS.includes(persona.id)) {
      expect(tab, `"Datenschutz"-Tab muss für ${persona.id} sichtbar sein`).not.toBeNull()
    } else {
      expect(tab, `"Datenschutz"-Tab darf für ${persona.id} NICHT sichtbar sein`).toBeNull()
    }
  })
})
