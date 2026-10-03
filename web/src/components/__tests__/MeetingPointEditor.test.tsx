import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import MeetingPointEditor from '../MeetingPointEditor'

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api)
})
afterEach(() => {
  mock.restore()
})

describe('MeetingPointEditor', () => {
  test('sendet Uhrzeit und getrimmten Ort, meldet das Ergebnis', async () => {
    const onSaved = vi.fn()
    mock.onPut('/games/7/meeting').reply(200, { meet_time: '13:30', meet_date: '2026-10-11', meet_place: 'Parkplatz' })

    render(<MeetingPointEditor gameId={7} initialTime={null} initialPlace="" onSaved={onSaved} />)
    fireEvent.change(screen.getByLabelText('Treffzeit'), { target: { value: '13:30' } })
    fireEvent.change(screen.getByLabelText('Treffpunkt (optional)'), { target: { value: ' Parkplatz ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ meet_time: '13:30', meet_place: 'Parkplatz' })
    expect(screen.getByRole('button', { name: 'Entfernen' })).toBeInTheDocument()
  })

  test('zeigt den übersetzten Fehler des Backends', async () => {
    mock.onPut('/games/7/meeting').reply(400, { error: 'meet_after_start' })

    render(<MeetingPointEditor gameId={7} initialTime={null} initialPlace="" />)
    fireEvent.change(screen.getByLabelText('Treffzeit'), { target: { value: '19:00' } })
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    expect(await screen.findByText('Die Treffzeit muss vor dem Anwurf liegen')).toBeInTheDocument()
  })

  test('Entfernen sendet leere Werte', async () => {
    const onSaved = vi.fn()
    mock.onPut('/games/7/meeting').reply(200, { meet_time: null, meet_date: null, meet_place: '' })

    render(<MeetingPointEditor gameId={7} initialTime="13:30" initialPlace="Bus" onSaved={onSaved} />)
    fireEvent.click(screen.getByRole('button', { name: 'Entfernen' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalledWith({ meet_time: null, meet_date: null, meet_place: '' }))
    expect(JSON.parse(mock.history.put[0].data)).toEqual({ meet_time: '', meet_place: '' })
    expect(screen.queryByRole('button', { name: 'Entfernen' })).not.toBeInTheDocument()
  })

  test('Ort ohne Uhrzeit lässt sich nicht speichern', () => {
    render(<MeetingPointEditor gameId={7} initialTime={null} initialPlace="" />)
    fireEvent.change(screen.getByLabelText('Treffpunkt (optional)'), { target: { value: 'Halle' } })
    expect((screen.getByRole('button', { name: 'Speichern' }) as HTMLButtonElement).disabled).toBe(true)
  })
})
