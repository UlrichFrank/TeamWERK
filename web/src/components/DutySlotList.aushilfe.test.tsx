import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from './DutySlotList'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// Eingetragene aus dem erweiterten Kader tragen im Slot bewusst KEIN
// „Erw. Kader“-Kennzeichen — auf Mobile überlagerte es die übrigen Namen.

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: 'Alice', role: 'standard' } }),
}))
vi.mock('../lib/api', () => ({ api: { post: vi.fn(), delete: vi.fn(), get: vi.fn().mockResolvedValue({ data: {} }), put: vi.fn() } }))

function slot(assignees: BoardSlot['assignees']): BoardSlot {
  return {
    id: 7, duty_type: 'Bewirtung', duty_type_id: 3, has_instruction: false, event_time: '15:00',
    hours_value: 3, slots_total: 3, vacancies: 1, claimed_by_me: false, assignees, comment_count: 0,
  }
}

describe('DutySlotList — kein Aushilfe-Kennzeichen am Slot', () => {
  test('auch die Aushilfe trägt kein Kennzeichen, ihr Name bleibt sichtbar', () => {
    render(
      <MemoryRouter>
        <PersonContactProvider>
        <DutySlotList
          slots={[slot([
            { user_id: 10, name: 'Sabine Roth', aushilfe: false },
            { user_id: 11, name: 'Jonas Keller', aushilfe: true },
          ])]}
          isPast={false} canEdit={false} onReload={() => {}}
        />
        </PersonContactProvider>
      </MemoryRouter>,
    )
    expect(screen.queryByText('Erw. Kader')).not.toBeInTheDocument()
    expect(screen.getByText('Jonas Keller')).toBeInTheDocument()
    expect(screen.getByText('Sabine Roth')).toBeInTheDocument()
  })

  test('ohne Aushilfe kein Kennzeichen', () => {
    render(
      <MemoryRouter>
        <PersonContactProvider>
        <DutySlotList slots={[slot([{ user_id: 10, name: 'Sabine Roth' }])]} isPast={false} canEdit={false} onReload={() => {}} />
        </PersonContactProvider>
      </MemoryRouter>,
    )
    expect(screen.queryByText('Erw. Kader')).not.toBeInTheDocument()
  })
})
