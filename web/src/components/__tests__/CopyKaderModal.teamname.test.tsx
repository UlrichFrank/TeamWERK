import { describe, test, expect } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import CopyKaderModal from '../CopyKaderModal'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

// Der Kopierer arbeitet je Altersklasse+Geschlecht und legt in der Zielsaison
// genau eine Mannschaft an — der Dialog zeigt deshalb eine Zeile je Kombination.
describe('CopyKaderModal — eine Zeile je Kombination', () => {
  test('zwei C-Jugenden der Quellsaison erscheinen als eine Zeile mit Hinweis', async () => {
    renderAsPersona(<CopyKaderModal toSeasonId={2} toSeasonName="2027/28" onDone={() => {}} onClose={() => {}} />, 'vorstand', {
      mocks: [
        { url: '/seasons', data: [{ id: 1, name: '2026/27', is_active: false }, { id: 2, name: '2027/28', is_active: true }] },
        {
          url: '/kader',
          data: {
            items: [
              { id: 1, age_class: 'C-Jugend', gender: 'm', member_count: 12 },
              { id: 2, age_class: 'C-Jugend', gender: 'm', member_count: 10 },
              { id: 3, age_class: 'A-Jugend', gender: 'f', member_count: 8 },
            ],
          },
        },
      ],
    })
    await flushAsync()
    fireEvent.change(screen.getByRole('combobox'), { target: { value: '1' } })
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Weiter' }))
    await flushAsync()

    expect(screen.getAllByText('C-Jugend männlich')).toHaveLength(1)
    expect(screen.getByText('2 Mannschaften in der Quellsaison – es wird eine angelegt')).toBeInTheDocument()
    expect(screen.getByText('A-Jugend weiblich')).toBeInTheDocument()
    expect(screen.getAllByText(/Mannschaften in der Quellsaison/)).toHaveLength(1)
  })
})
