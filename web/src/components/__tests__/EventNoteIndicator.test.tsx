import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import EventNoteIndicator from '../EventNoteIndicator'

describe('EventNoteIndicator', () => {
  test('rendert nichts bei leerem Hinweis', () => {
    const { container } = render(<EventNoteIndicator variant="icon" note="   " />)
    expect(container.firstChild).toBeNull()
  })

  test('icon-Variante zeigt title-Tooltip mit vollem Text, aber nicht als Klartext', () => {
    render(<EventNoteIndicator variant="icon" note="Halle gesperrt" />)
    const el = screen.getByLabelText('Hinweis vorhanden')
    expect(el).toHaveAttribute('title', 'Hinweis: Halle gesperrt')
    expect(screen.queryByText('Halle gesperrt')).toBeNull()
  })

  test('inline-Variante zeigt vollen Hinweistext', () => {
    render(<EventNoteIndicator variant="inline" note="Bringt Hallenschuhe mit" />, { wrapper: MemoryRouter })
    expect(screen.getByText('Bringt Hallenschuhe mit')).toBeInTheDocument()
  })

  test('inline-Variante macht einen Dokument-Link klickbar', () => {
    const url = `${window.location.origin}/dokumente/datei/12`
    render(<EventNoteIndicator variant="inline" note={`Turnierplan: ${url}`} />, { wrapper: MemoryRouter })
    expect(screen.getByRole('link', { name: url })).toHaveAttribute('href', url)
    expect(screen.getByText(/Turnierplan:/)).toBeInTheDocument()
  })
})
