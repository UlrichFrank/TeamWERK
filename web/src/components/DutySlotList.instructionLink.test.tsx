import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from './DutySlotList'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: 'Alice', role: 'standard' } }),
}))
vi.mock('../lib/api', () => ({ api: { post: vi.fn(), delete: vi.fn() } }))

function baseSlot(overrides: Partial<BoardSlot> = {}): BoardSlot {
  return {
    id: 100,
    duty_type: 'Kasse',
    duty_type_id: 42,
    has_instruction: false,
    event_time: '10:00',
    slots_total: 2,
    vacancies: 1,
    claimed_by_me: false,
    assignees: [],
    ...overrides,
  }
}

describe('DutySlotList — Anleitung link', () => {
  test('renders link to instruction page when has_instruction is true', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: true })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link', { name: 'Anleitung ansehen' })
    expect(link.getAttribute('href')).toBe('/dienste/anleitung/42')
  })

  test('renders a strikethrough icon button when has_instruction is false, click opens info modal', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: false })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    // No link — instead a button with an explanatory aria-label.
    expect(screen.queryByRole('link', { name: 'Anleitung ansehen' })).toBeNull()
    const btn = screen.getByRole('button', { name: 'Keine Anleitung vorhanden' })
    expect(btn).not.toBeNull()

    // Click opens an info modal.
    fireEvent.click(btn)
    expect(screen.getByText(/noch keine Anleitung/)).toBeTruthy()
  })
})
