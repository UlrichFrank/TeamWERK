import { describe, test, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from './DutySlotList'
import { api } from '../lib/api'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: 'Alice', role: 'standard' } }),
}))
vi.mock('../lib/api', () => ({
  api: { post: vi.fn(), delete: vi.fn(), get: vi.fn(), put: vi.fn() },
}))

afterEach(() => {
  vi.clearAllMocks()
})

function baseSlot(overrides: Partial<BoardSlot> = {}): BoardSlot {
  return {
    id: 100,
    duty_type: 'Kuchen',
    duty_type_id: 42,
    has_instruction: false,
    event_time: '10:00',
    hours_value: 1,
    slots_total: 2,
    vacancies: 1,
    claimed_by_me: false,
    assignees: [],
    comment_count: 0,
    ...overrides,
  }
}

describe('DutySlotList — Kommentar-Icon', () => {
  test('erscheint nicht bei comment_count=0', () => {
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ comment_count: 0 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.queryByLabelText(/Kommentar.* ansehen/)).toBeNull()
  })

  test('erscheint bei comment_count>0 mit korrekter Pluralform', () => {
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ comment_count: 2 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.getByLabelText('2 Kommentare ansehen')).toBeInTheDocument()
  })

  test('Singular bei genau einem Kommentar', () => {
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ comment_count: 1 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.getByLabelText('1 Kommentar ansehen')).toBeInTheDocument()
  })
})

describe('DutySlotList — Kommentar-Modal (on-demand)', () => {
  test('lädt Kommentare erst beim Öffnen, nicht beim initialen Render', () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] })
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ comment_count: 1 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(api.get).not.toHaveBeenCalled()

    fireEvent.click(screen.getByLabelText('1 Kommentar ansehen'))
    expect(api.get).toHaveBeenCalledWith('/duty-slots/100/comments')
  })

  test('zeigt geladene Kommentare im Modal', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: [
        { user_id: 5, user_name: 'Bob', body: 'Marmorkuchen', created_at: '2026-06-10 09:00:00' },
      ],
    })
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ comment_count: 1 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByLabelText('1 Kommentar ansehen'))
    await waitFor(() => expect(screen.getByText('Marmorkuchen')).toBeInTheDocument())
    expect(screen.getByText('Bob')).toBeInTheDocument()
  })

  test('ohne eigene Zuteilung ist das Modal reiner Lesemodus (keine Textarea)', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] })
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ comment_count: 1, claimed_by_me: false })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByLabelText('1 Kommentar ansehen'))
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(screen.queryByLabelText('Dein Kommentar')).toBeNull()
  })

  test('mit eigener Zuteilung zeigt die Textarea und speichert per PUT', async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [] })
    vi.mocked(api.put).mockResolvedValue({ data: {} })
    const onReload = vi.fn()
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ comment_count: 0, claimed_by_me: true, my_assignment_id: 77, vacancies: 1 })]}
          isPast={false}
          canEdit={false}
          onReload={onReload}
        />
      </MemoryRouter>,
    )
    // Ohne vorhandene Kommentare ist das Icon nicht da — Öffnen läuft über das
    // ⋮-Menü ("Kommentieren").
    fireEvent.click(screen.getAllByLabelText('Aktionen')[0])
    fireEvent.click(screen.getAllByRole('button', { name: 'Kommentieren' })[0])
    await waitFor(() => expect(api.get).toHaveBeenCalledWith('/duty-slots/100/comments'))

    fireEvent.change(screen.getByLabelText('Dein Kommentar'), { target: { value: 'Käsekuchen' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith('/duty-assignments/77/comment', { body: 'Käsekuchen' }),
    )
    expect(onReload).toHaveBeenCalled()
  })
})

describe('DutySlotList — "Kommentieren" im ⋮-Menü', () => {
  test('fehlt ohne eigene Zuteilung', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ claimed_by_me: false, vacancies: 1 })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByLabelText('Aktionen'))
    expect(screen.queryByRole('button', { name: 'Kommentieren' })).toBeNull()
  })

  test('erscheint mit eigener Zuteilung, sowohl im Desktop- als auch im Mobile-Menü', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ claimed_by_me: true, my_assignment_id: 55, has_instruction: true, duty_type_id: 9 })]}
          isPast={false}
          canEdit
          onEdit={() => {}}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    // Zwei ActionMenu-Instanzen (Desktop `hidden sm:flex` + Mobile `sm:hidden`,
    // jsdom kennt keine Media Queries) — beide tragen denselben Eintrags-Satz.
    const triggers = screen.getAllByLabelText('Aktionen')
    expect(triggers).toHaveLength(2)
    for (const trigger of triggers) {
      fireEvent.click(trigger)
    }
    const kommentierenEntries = screen.getAllByRole('button', { name: 'Kommentieren' })
    expect(kommentierenEntries).toHaveLength(2)
  })
})

describe('DutySlotList — ⋮-Menü auch auf Desktop', () => {
  test('Desktop-Menü enthält Bearbeiten/Anleitung, Eintragen/Austragen bleiben eigene Buttons', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: true, duty_type_id: 3, claimed_by_me: false, vacancies: 1 })]}
          isPast={false}
          canEdit
          onEdit={() => {}}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    // Eintragen ist ein direkt sichtbarer Button außerhalb jedes Menüs.
    expect(screen.getAllByRole('button', { name: 'Eintragen' }).length).toBeGreaterThan(0)

    const triggers = screen.getAllByLabelText('Aktionen')
    expect(triggers.length).toBeGreaterThan(0)
    fireEvent.click(triggers[0])
    expect(screen.getAllByRole('button', { name: 'Bearbeiten' })[0]).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Anleitung' })[0]).toBeInTheDocument()
    // Eintragen/Austragen sind kein Menü-Eintrag im Desktop-Menü.
    expect(screen.queryAllByRole('button', { name: 'Eintragen' }).length).toBeGreaterThan(0)
  })
})
