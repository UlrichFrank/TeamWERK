import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import DashboardPage from '../DashboardPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { relativeTime } from '../../lib/relativeTime'

// Dashboard abonniert Live-Updates/Chat-SSE; im Test neutralisieren (wie
// DashboardPage.nachrichten.test.tsx).
vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../hooks/useChatEvents', () => ({ useChatEvents: vi.fn() }))

const BASE_DASHBOARD = {
  currentSeason: null,
  meineTermine: [],
  meineDienste: null,
  carpoolingConfirmed: [],
  carpoolingOpenGroups: [],
}

// Keine Fake-Timer: flushAsync wartet über einen echten setTimeout(0) auf die
// Effekt-getriebenen Fetches — mit vi.useFakeTimers() liefe das ins Leere.
// createdAt wird deshalb relativ zur echten Uhrzeit gebildet und die erwartete
// Zeitangabe direkt über denselben Helfer berechnet, den die Komponente nutzt.
function hoursAgo(h: number): string {
  return new Date(Date.now() - h * 60 * 60 * 1000).toISOString()
}

describe('DashboardPage — Geschehen-Section (Event-Log)', () => {
  test('zeigt Eintrag mit Titel, Text und relativer Zeit; Eintrag mit url ist anklickbar', async () => {
    const createdAt = hoursAgo(2)
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        {
          url: '/dashboard',
          data: {
            ...BASE_DASHBOARD,
            events: [
              {
                id: 1,
                category: 'games',
                title: 'Spiel gegen Ludwigsburg abgesagt',
                body: 'Halle gesperrt',
                url: '/termine/5',
                createdAt,
              },
            ],
          },
        },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Geschehen')).toBeInTheDocument()
    const title = screen.getByText('Spiel gegen Ludwigsburg abgesagt')
    expect(screen.getByText('Halle gesperrt')).toBeInTheDocument()
    expect(screen.getByText(relativeTime(createdAt))).toBeInTheDocument()

    // Anklickbar: der Eintrag ist als Link ins Ziel gerendert.
    const link = title.closest('a')
    expect(link).not.toBeNull()
    expect(link).toHaveAttribute('href', '/termine/5')
  })

  test('Eintrag mit leerer url wird dargestellt, ist aber nicht anklickbar', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        {
          url: '/dashboard',
          data: {
            ...BASE_DASHBOARD,
            events: [
              {
                id: 2,
                category: 'games',
                title: 'Termin abgesagt',
                body: 'Der Termin existiert nicht mehr',
                url: '',
                createdAt: hoursAgo(1),
              },
            ],
          },
        },
      ],
    })
    await flushAsync()

    const title = screen.getByText('Termin abgesagt')
    expect(title.closest('a')).toBeNull()
  })

  test('Leerzustand ohne Einträge', async () => {
    renderAsPersona(<DashboardPage />, 'spieler', {
      mocks: [
        { url: '/dashboard', data: { ...BASE_DASHBOARD, events: [] } },
      ],
    })
    await flushAsync()

    expect(screen.getByText('Geschehen')).toBeInTheDocument()
    expect(screen.getByText('Keine Ereignisse.')).toBeInTheDocument()
  })
})
