import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
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
    render(<EventNoteIndicator variant="inline" note="Bringt Hallenschuhe mit" />)
    expect(screen.getByText('Bringt Hallenschuhe mit')).toBeInTheDocument()
  })
})
