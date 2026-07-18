/**
 * Sichtbarkeit des Anwesenheit-Tabs in /profil ist an die Vereinsfunktion `spieler`
 * gekoppelt (eigenes Mitglied oder verlinktes Kind). Trainer/Vorstand ohne Spieler-
 * Funktion sehen den Tab nicht — Anwesenheit ist ein Spieler-Konzept, ihre Sicht ist
 * /team/{id}/anwesenheit. Regel liegt in openspec/specs/attendance-statistics/spec.md,
 * Requirement "Trainer- und Spieler-Sichten im Frontend".
 */
import type { ReactElement } from 'react'
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import ProfilePage from '../ProfilePage'
import { ProfilAnwesenheitContent } from '../ProfilAnwesenheitPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../components/AttendanceStatsView', () => ({
  default: ({ memberId }: { memberId: number }) => (
    <div data-testid="stats-view">stats:{memberId}</div>
  ),
}))

type MemberFixture = { id: number; first_name: string; last_name: string; club_functions?: string[] }

/**
 * Registriert /profile/me in einer definierten Form. Wird VOR flushAsync und
 * NACH renderAsPersona aufgerufen, weil renderAsPersona den MockAdapter neu erzeugt
 * (das Default-/profile/me würde sonst greifen bevor unser Extra registriert wäre).
 */
function setProfile(me: { own_member?: MemberFixture | null; children?: MemberFixture[] }) {
  const mock = getApiMock()
  mock.reset()
  mock.onGet('/profile/me').reply(200, {
    own_member: me.own_member ?? null,
    children: me.children ?? [],
    parents: [],
    phones: [],
    visibility: {},
  })
  mock.onAny().reply(200, [])
}

function renderProfile(
  ui: ReactElement,
  personaId: string,
  me: { own_member?: MemberFixture | null; children?: MemberFixture[] },
  route?: string,
) {
  renderAsPersona(ui, personaId, route ? { route } : {})
  setProfile(me)
}

const OWN_SPIELER: MemberFixture = { id: 1, first_name: 'Sam', last_name: 'Spieler', club_functions: ['spieler'] }
const OWN_TRAINER: MemberFixture = { id: 2, first_name: 'Thomas', last_name: 'Eisele', club_functions: ['trainer'] }
const KID_SPIELER: MemberFixture = { id: 10, first_name: 'Kai', last_name: 'Kind', club_functions: ['spieler'] }
const KID_NONPLAYER: MemberFixture = { id: 11, first_name: 'Nele', last_name: 'Nichtspieler', club_functions: [] }

describe('ProfilePage — Anwesenheit-Tab-Sichtbarkeit', () => {
  test('own=null, kids=[]: Tab NICHT sichtbar', async () => {
    renderProfile(<ProfilePage />, 'admin', {}, '/profil')
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Anwesenheit' })).toBeNull()
  })

  test('own=[trainer], kids=[]: Tab NICHT sichtbar (Thomas-Fall)', async () => {
    renderProfile(<ProfilePage />, 'trainer', { own_member: OWN_TRAINER }, '/profil')
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Anwesenheit' })).toBeNull()
  })

  test('own=[spieler], kids=[]: Tab sichtbar', async () => {
    renderProfile(<ProfilePage />, 'spieler', { own_member: OWN_SPIELER }, '/profil')
    await flushAsync()
    expect(screen.getByRole('button', { name: 'Anwesenheit' })).toBeInTheDocument()
  })

  test('own=[trainer] + kid=[spieler]: Tab sichtbar', async () => {
    renderProfile(<ProfilePage />, 'trainer_elternteil', {
      own_member: OWN_TRAINER,
      children: [KID_SPIELER],
    }, '/profil')
    await flushAsync()
    expect(screen.getByRole('button', { name: 'Anwesenheit' })).toBeInTheDocument()
  })

  test('own=[trainer] + kid=[] (Nicht-Spieler-Kind): Tab NICHT sichtbar', async () => {
    renderProfile(<ProfilePage />, 'trainer_elternteil', {
      own_member: OWN_TRAINER,
      children: [KID_NONPLAYER],
    }, '/profil')
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Anwesenheit' })).toBeNull()
  })
})

describe('ProfilAnwesenheitContent — Options-Filter', () => {
  test('own=[spieler] alleine: nur ein Kandidat → keine Buttons-Zeile, Stats für own', async () => {
    renderProfile(<ProfilAnwesenheitContent />, 'spieler', { own_member: OWN_SPIELER })
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Sam Spieler' })).toBeNull()
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:1')
  })

  test('own=[trainer] + kid=[spieler]: Options zeigen nur Kind, Default ist Kind', async () => {
    renderProfile(<ProfilAnwesenheitContent />, 'trainer_elternteil', {
      own_member: OWN_TRAINER,
      children: [KID_SPIELER],
    })
    await flushAsync()
    // Trainer-Vater darf sich NICHT auswählen können. Genau ein Kandidat übrig → keine Buttons.
    expect(screen.queryByRole('button', { name: 'Thomas Eisele' })).toBeNull()
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:10')
  })

  test('own=[spieler] + kid=[spieler]: beide Options sichtbar, Default ist own, Wechsel möglich', async () => {
    renderProfile(<ProfilAnwesenheitContent />, 'spieler', {
      own_member: OWN_SPIELER,
      children: [KID_SPIELER],
    })
    await flushAsync()
    expect(screen.getByRole('button', { name: 'Sam Spieler' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kai Kind' })).toBeInTheDocument()
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:1')

    fireEvent.click(screen.getByRole('button', { name: 'Kai Kind' }))
    await flushAsync()
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:10')
  })

  test('own=[trainer], kids=[] mit forcedMemberId=42: umgeht Spieler-Filter (Trainer-Drilldown)', async () => {
    renderProfile(<ProfilAnwesenheitContent forcedMemberId={42} />, 'trainer', {
      own_member: OWN_TRAINER,
    })
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Thomas Eisele' })).toBeNull()
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:42')
  })

  test('own=[trainer], kids=[] ohne forcedMemberId: leere Options → "Keine Anwesenheitsdaten"', async () => {
    renderProfile(<ProfilAnwesenheitContent />, 'trainer', { own_member: OWN_TRAINER })
    await flushAsync()
    expect(screen.queryByTestId('stats-view')).toBeNull()
    expect(screen.getByText(/Keine Anwesenheitsdaten/)).toBeInTheDocument()
  })
})
