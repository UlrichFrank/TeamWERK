import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import EventNoteEditor from '../EventNoteEditor'

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api)
})
afterEach(() => {
  mock.restore()
})

describe('EventNoteEditor', () => {
  test('Save-Roundtrip ruft PUT /trainings/{id}/note und onSaved', async () => {
    const onSaved = vi.fn()
    mock.onPut('/trainings/42/note').reply(200, {})

    render(<EventNoteEditor eventType="training" eventId={42} initialNote="" onSaved={onSaved} />)

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'Halle gesperrt' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Halle gesperrt'))
    expect(mock.history.put).toHaveLength(1)
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ note: 'Halle gesperrt' })
  })

  test('blockt Speichern bei > 200 Zeichen', () => {
    render(<EventNoteEditor eventType="game" eventId={1} initialNote="" />)
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'x'.repeat(201) } })

    const btn = screen.getByRole('button', { name: 'Speichern' }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    expect(screen.getByText('201/200')).toBeInTheDocument()
  })

  test('Speichern deaktiviert solange unverändert', () => {
    render(<EventNoteEditor eventType="game" eventId={1} initialNote="schon da" />)
    const btn = screen.getByRole('button', { name: 'Speichern' }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })
})
