import { describe, test, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import RpeScaleInfo from '../RpeScaleInfo'

describe('RpeScaleInfo', () => {
  test('ist standardmäßig eingeklappt', () => {
    render(<RpeScaleInfo />)
    const toggle = screen.getByRole('button', { name: /Was bedeutet die Skala/ })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    // Die Stufenbeschreibungen sind nicht im DOM, solange zugeklappt ist.
    expect(screen.queryByText(/sehr leicht/)).not.toBeInTheDocument()
  })

  test('klappt per Klick auf und zeigt alle Stufen', () => {
    render(<RpeScaleInfo />)
    fireEvent.click(screen.getByRole('button', { name: /Was bedeutet die Skala/ }))

    expect(screen.getByRole('button', { name: /Was bedeutet die Skala/ })).toHaveAttribute(
      'aria-expanded',
      'true',
    )
    for (const label of ['sehr leicht', 'leicht', 'mittel', 'hart', 'maximal']) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
    expect(screen.getByText(/kein richtig oder falsch/)).toBeInTheDocument()
  })

  test('klappt per zweitem Klick wieder zu', () => {
    render(<RpeScaleInfo />)
    const toggle = screen.getByRole('button', { name: /Was bedeutet die Skala/ })
    fireEvent.click(toggle)
    fireEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText(/sehr leicht/)).not.toBeInTheDocument()
  })
})
