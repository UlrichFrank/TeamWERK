import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, act, fireEvent } from '@testing-library/react'
import MembersPage from '../MembersPage'
import { renderAsPersona } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
// PersonChip hängt am PersonContactProvider; für den Windowing-Test reicht der Name.
vi.mock('../../components/PersonChip', () => ({
  default: ({ name }: { name: string }) => <span>{name}</span>,
}))

const MANY = Array.from({ length: 200 }, (_, i) => ({
  id: i,
  first_name: `Vorname${i}`,
  last_name: `Nachname${i}`,
  status: 'aktiv',
  club_functions: ['spieler'],
  can: { edit: false, delete: false },
}))

vi.mock('../../lib/usePagination', () => ({
  usePagination: () => ({
    items: MANY, total: 200, currentPage: 1, totalPages: 1,
    loading: false, error: null,
    setSearch: vi.fn(), goToPage: vi.fn(), refresh: vi.fn(),
  }),
}))

// Layout simulieren (jsdom liefert 0): Fenster-Scroll-Modus, Wrapper-Position steuerbar.
const VIEWPORT = 300
const ROW_HEIGHT = 53
const topBox = { value: 0 }

beforeAll(() => {
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: VIEWPORT })
  const orig = HTMLElement.prototype.getBoundingClientRect
  HTMLElement.prototype.getBoundingClientRect = function () {
    if (this.querySelector('table')) {
      return { top: topBox.value, left: 0, right: 0, bottom: 0, width: 0, height: 0, x: 0, y: topBox.value, toJSON: () => {} } as DOMRect
    }
    return orig.call(this)
  }
})

describe('MembersPage — Windowing der Mitgliedertabelle', () => {
  test('rendert nur sichtbare Zeilen; Scrollen tauscht sie aus', () => {
    topBox.value = 0
    renderAsPersona(<MembersPage />, 'vorstand')

    expect(screen.getByText('Nachname0, Vorname0')).toBeInTheDocument()
    expect(screen.queryByText('Nachname150, Vorname150')).toBeNull()

    act(() => {
      topBox.value = -150 * ROW_HEIGHT
      fireEvent.scroll(window)
    })

    expect(screen.getByText('Nachname150, Vorname150')).toBeInTheDocument()
    expect(screen.queryByText('Nachname0, Vorname0')).toBeNull()
  })
})
