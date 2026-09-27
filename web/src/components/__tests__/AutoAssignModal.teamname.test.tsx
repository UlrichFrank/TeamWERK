import { describe, test, expect } from 'vitest'
import { screen } from '@testing-library/react'
import AutoAssignModal from '../AutoAssignModal'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

describe('AutoAssignModal — Teamname mit Nummer am Ende', () => {
  test('zwei gleichartige Kader sind als "… 1" und "… 2" unterscheidbar', async () => {
    renderAsPersona(<AutoAssignModal seasonId={1} onDone={() => {}} onClose={() => {}} />, 'vorstand', {
      mocks: [{
        url: '/kader',
        data: {
          items: [
            { id: 1, age_class: 'C-Jugend', gender: 'm', team_number: 1, bracket_years: [] },
            { id: 2, age_class: 'C-Jugend', gender: 'm', team_number: 2, bracket_years: [] },
            { id: 3, age_class: 'A-Jugend', gender: 'f', team_number: 1, bracket_years: [] },
          ],
        },
      }],
    })
    await flushAsync()
    expect(screen.getByText('C-Jugend männlich 1')).toBeInTheDocument()
    expect(screen.getByText('C-Jugend männlich 2')).toBeInTheDocument()
    expect(screen.getByText('A-Jugend weiblich')).toBeInTheDocument()
  })
})
