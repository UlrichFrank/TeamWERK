import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from './DutySlotList'

// Teil von openspec/changes/dienst-dauer: die Dienstbörse zeigt eine Zeitspanne
// (Start–Ende) statt nur des Startzeitpunkts, gebildet aus event_time + hours_value.

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
    event_time: '08:00',
    hours_value: 1,
    slots_total: 2,
    vacancies: 1,
    claimed_by_me: false,
    assignees: [],
    ...overrides,
  }
}

describe('DutySlotList — Zeitspanne', () => {
  test('Slot mit Dauer rendert Start–Ende statt nur des Startzeitpunkts', () => {
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ event_time: '08:00', hours_value: 1 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.getByText('8:00–9:00')).toBeInTheDocument()
  })

  test('Slot ohne event_time rendert weiter den bisherigen Platzhalter', () => {
    render(
      <MemoryRouter>
        <DutySlotList slots={[baseSlot({ event_time: '', hours_value: 1 })]} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})
