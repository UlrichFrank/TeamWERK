import { describe, test, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'
import SearchableSelect, { type SearchableSelectItem } from './SearchableSelect'

afterEach(cleanup)

function makeItems(n: number, likelyIds: number[] = []): SearchableSelectItem[] {
  return Array.from({ length: n }, (_, i) => ({
    value: String(i),
    label: `Person ${i}`,
    searchText: `person ${i}`,
    likely: likelyIds.includes(i),
  }))
}

describe('SearchableSelect', () => {
  test('zeigt wahrscheinliche Treffer getrennt durch Separator über den restlichen Treffern', () => {
    const items = [
      { value: '1', label: 'Anna Müller', searchText: 'anna müller', likely: true },
      { value: '2', label: 'Bob Schmidt', searchText: 'bob schmidt', likely: false },
      { value: '3', label: 'Carla Weber', searchText: 'carla weber', likely: false },
    ]
    render(<SearchableSelect items={items} value="" onChange={() => {}} />)
    fireEvent.focus(screen.getByRole('textbox'))

    expect(screen.getByText('Wahrscheinliche Treffer')).toBeTruthy()
    const options = screen.getAllByRole('button').map(b => b.textContent)
    expect(options).toEqual(['Anna Müller', 'Bob Schmidt', 'Carla Weber'])
  })

  test('tippen filtert die Liste', () => {
    const items = makeItems(5)
    render(<SearchableSelect items={items} value="" onChange={() => {}} />)
    const input = screen.getByRole('textbox')
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'Person 3' } })

    expect(screen.getByText('Person 3')).toBeTruthy()
    expect(screen.queryByText('Person 0')).toBeNull()
  })

  test('begrenzt die Liste auf 100 Elemente und zeigt einen Hinweis auf weitere Treffer', () => {
    const items = makeItems(150)
    render(<SearchableSelect items={items} value="" onChange={() => {}} />)
    fireEvent.focus(screen.getByRole('textbox'))

    const options = screen.getAllByRole('button')
    expect(options).toHaveLength(100)
    expect(screen.getByText(/und 50 weitere/)).toBeTruthy()
  })

  test('Auswahl eines Eintrags ruft onChange auf und schließt die Liste', () => {
    const items = makeItems(3)
    const onChange = vi.fn()
    render(<SearchableSelect items={items} value="" onChange={onChange} />)
    fireEvent.focus(screen.getByRole('textbox'))
    fireEvent.mouseDown(screen.getByText('Person 1'))

    expect(onChange).toHaveBeenCalledWith('1')
    expect(screen.queryByText('Person 0')).toBeNull()
  })

  test('clearLabel setzt die Auswahl zurück', () => {
    const items = makeItems(3)
    const onChange = vi.fn()
    render(<SearchableSelect items={items} value="1" onChange={onChange} clearLabel="– Keine Verknüpfung –" />)
    fireEvent.focus(screen.getByRole('textbox'))
    fireEvent.mouseDown(screen.getByText('– Keine Verknüpfung –'))

    expect(onChange).toHaveBeenCalledWith('')
  })
})
