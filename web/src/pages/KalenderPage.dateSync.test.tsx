import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import KalenderPage from './KalenderPage'

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../hooks/useCompactHeader', () => ({ useCompactHeader: () => false }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', isParent: false, clubFunctions: [] },
    hasCapability: () => false,
    logout: vi.fn(),
  }),
}))
vi.mock('../lib/useEscapeKey', () => ({ useEscapeKey: vi.fn() }))

// Mock minimal API responses
function seedEmptyData() {
  mockGet.mockImplementation((url: string) => {
    if (url.includes('/games')) return Promise.resolve({ data: [] })
    if (url.includes('/training-sessions')) return Promise.resolve({ data: [] })
    if (url.includes('/training-series')) return Promise.resolve({ data: [] })
    if (url.includes('/absences')) return Promise.resolve({ data: [] })
    if (url.includes('/teams')) return Promise.resolve({ data: [] })
    if (url.includes('/teams/names')) return Promise.resolve({ data: [] })
    if (url.includes('/seasons')) return Promise.resolve({ data: [] })
    if (url.includes('/venues')) return Promise.resolve({ data: [] })
    if (url.includes('/duty-templates')) return Promise.resolve({ data: [] })
    return Promise.resolve({ data: [] })
  })
}

function renderKalender(initialEntry: string = '/kalender') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <KalenderPage />
    </MemoryRouter>,
  )
}

// Der Kalender startet ohne ?date im aktuellen Monat — die Erwartungen unten
// ("August 2026") sind damit an die Wanduhr gebunden und kippten beim
// Monatswechsel. Nur `Date` wird gefälscht, damit Timer, waitFor und userEvent
// weiter auf echten Timern laufen.
const NOW = new Date(2026, 7, 15, 10, 0, 0) // 15.08.2026, lokale Zeit

describe('KalenderPage — date sync to URL', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(NOW)
    vi.clearAllMocks()
    seedEmptyData()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  test('renders calendar with current month on first mount', async () => {
    renderKalender()
    await waitFor(() => {
      const heading = screen.getByText(/August 2026/)
      expect(heading).toBeTruthy()
    })
  })

  test('clicking next month displays new month', async () => {
    renderKalender('/kalender')
    await waitFor(() => screen.getByText(/August 2026/))

    const nextButton = screen.getByLabelText('Nächster Monat')
    await userEvent.click(nextButton)

    // September is month 8 (0-indexed)
    await waitFor(() => {
      expect(screen.getByText(/September 2026/)).toBeTruthy()
    })
  })

  test('clicking prev month displays new month', async () => {
    renderKalender('/kalender')
    await waitFor(() => screen.getByText(/August 2026/))

    const prevButton = screen.getByLabelText('Vorheriger Monat')
    await userEvent.click(prevButton)

    // July is month 6 (0-indexed)
    await waitFor(() => {
      expect(screen.getByText(/Juli 2026/)).toBeTruthy()
    })
  })

  test('clicking prev month from January goes to December of previous year', async () => {
    renderKalender('/kalender?date=2026-01-15')
    await waitFor(() => screen.getByText(/Januar 2026/))

    const prevButton = screen.getByLabelText('Vorheriger Monat')
    await userEvent.click(prevButton)

    await waitFor(() => {
      expect(screen.getByText(/Dezember 2025/)).toBeTruthy()
    })
  })

  test('clicking next month from December goes to January of next year', async () => {
    renderKalender('/kalender?date=2026-12-15')
    await waitFor(() => screen.getByText(/Dezember 2026/))

    const nextButton = screen.getByLabelText('Nächster Monat')
    await userEvent.click(nextButton)

    await waitFor(() => {
      expect(screen.getByText(/Januar 2027/)).toBeTruthy()
    })
  })

  test('clicking "heute" returns to current month', async () => {
    // Start from a different month
    renderKalender('/kalender?date=2025-06-01')
    await waitFor(() => screen.getByText(/Juni 2025/))

    const todayButton = screen.getByRole('button', { name: /Heute/ })
    await userEvent.click(todayButton)

    // Current month is August 2026
    await waitFor(() => {
      expect(screen.getByText(/August 2026/)).toBeTruthy()
    })
  })

  test('mounting with ?date=YYYY-MM-DD shows that month', async () => {
    renderKalender('/kalender?date=2026-05-15')
    await waitFor(() => {
      expect(screen.getByText(/Mai 2026/)).toBeTruthy()
    })
  })

  test('mounting with ?date=2025-03-20 shows March 2025', async () => {
    renderKalender('/kalender?date=2025-03-20')
    await waitFor(() => {
      expect(screen.getByText(/März 2025/)).toBeTruthy()
    })
  })

  test('multiple month clicks update the display correctly', async () => {
    renderKalender('/kalender')
    await waitFor(() => screen.getByText(/August 2026/))

    const nextButton = screen.getByLabelText('Nächster Monat')
    await userEvent.click(nextButton)
    await userEvent.click(nextButton)

    // Verify we're in October 2026
    await waitFor(() => {
      expect(screen.getByText(/Oktober 2026/)).toBeTruthy()
    })
  })

  test('navigating away and back via browser history restores the month', async () => {
    const { rerender } = render(
      <MemoryRouter initialEntries={['/kalender?date=2026-05-01']}>
        <KalenderPage />
      </MemoryRouter>,
    )
    await waitFor(() => screen.getByText(/Mai 2026/))

    // Navigate to a different route
    rerender(
      <MemoryRouter initialEntries={['/kalender?date=2026-05-01', '/other']}>
        <KalenderPage />
      </MemoryRouter>,
    )

    // Navigate back
    rerender(
      <MemoryRouter initialEntries={['/kalender?date=2026-05-01']}>
        <KalenderPage />
      </MemoryRouter>,
    )

    // Should still show May 2026
    await waitFor(() => {
      expect(screen.getByText(/Mai 2026/)).toBeTruthy()
    })
  })
})
