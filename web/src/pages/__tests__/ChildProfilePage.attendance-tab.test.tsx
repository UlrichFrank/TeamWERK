/**
 * Die Anwesenheit eines verlinkten Kindes liegt auf dessen Kind-Detailseite
 * /profil/kind/:memberId — NICHT aggregiert im eigenen /profil der Eltern. Der Tab
 * erscheint dort genau dann, wenn das Kind die Vereinsfunktion `spieler` hat, und zeigt
 * (ohne Auswahl-Buttons) die Statistik dieses Kindes. Regel: openspec/specs/
 * attendance-statistics/spec.md, Requirement "Trainer- und Spieler-Sichten im Frontend".
 */
import { describe, test, expect, vi } from 'vitest'
import { Routes, Route } from 'react-router-dom'
import { screen, fireEvent } from '@testing-library/react'
import ChildProfilePage from '../ChildProfilePage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../components/AttendanceStatsView', () => ({
  default: ({ memberId }: { memberId: number }) => (
    <div data-testid="stats-view">stats:{memberId}</div>
  ),
}))

type MemberFixture = { id: number; first_name: string; last_name: string; club_functions?: string[] }

function renderChild(child: MemberFixture) {
  renderAsPersona(
    <Routes>
      <Route path="/profil/kind/:memberId" element={<ChildProfilePage />} />
    </Routes>,
    'trainer_elternteil',
    { route: `/profil/kind/${child.id}` },
  )
  const mock = getApiMock()
  mock.reset()
  mock.onGet(`/profile/kind/${child.id}`).reply(200, {
    member: child,
    parents: [],
    user_contact: { recovery_email: '', phones: [], visibility: {} },
  })
  mock.onGet('/profile/me').reply(200, { own_member: null, children: [child], parents: [] })
  mock.onAny().reply(200, [])
}

const KID_SPIELER: MemberFixture = { id: 10, first_name: 'Kai', last_name: 'Kind', club_functions: ['spieler'] }
const KID_NONPLAYER: MemberFixture = { id: 11, first_name: 'Nele', last_name: 'Nichtspieler', club_functions: [] }

describe('ChildProfilePage — Anwesenheit-Tab', () => {
  test('Spieler-Kind: Tab sichtbar, Klick zeigt Statistik genau dieses Kindes', async () => {
    renderChild(KID_SPIELER)
    await flushAsync()
    const tab = screen.getByRole('button', { name: 'Anwesenheit' })
    expect(tab).toBeInTheDocument()

    fireEvent.click(tab)
    await flushAsync()
    // forcedMemberId=member.id → keine Auswahl-Buttons, direkte Statistik.
    expect(screen.getByTestId('stats-view')).toHaveTextContent('stats:10')
    expect(screen.queryByRole('button', { name: 'Kai Kind' })).toBeNull()
  })

  test('Nicht-Spieler-Kind: Tab NICHT sichtbar', async () => {
    renderChild(KID_NONPLAYER)
    await flushAsync()
    expect(screen.queryByRole('button', { name: 'Anwesenheit' })).toBeNull()
  })
})
