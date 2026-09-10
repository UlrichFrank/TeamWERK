import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import ChatPollCreateModal from './ChatPollCreateModal'

const mockPost = vi.fn((..._args: unknown[]) => Promise.resolve({ data: { id: 1 } }))
vi.mock('../lib/api', () => ({
  api: { post: (...args: unknown[]) => mockPost(...args) },
}))

function fillQuestion(text: string) {
  fireEvent.change(screen.getByPlaceholderText('Frage eingeben…'), { target: { value: text } })
}

function optionInputs(): HTMLInputElement[] {
  return [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    .map((n) => screen.queryByPlaceholderText(`Option ${n}`) as HTMLInputElement | null)
    .filter((el): el is HTMLInputElement => el !== null)
}

describe('ChatPollCreateModal', () => {
  beforeEach(() => {
    mockPost.mockReset()
    mockPost.mockResolvedValue({ data: { id: 1 } })
  })

  test('Senden ist deaktiviert, solange weniger als zwei Optionen ausgefüllt sind', () => {
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={() => {}} />)
    fillQuestion('Wer fährt am Samstag?')
    fireEvent.change(optionInputs()[0], { target: { value: 'Ich' } })
    // zweites Feld bleibt leer
    expect(screen.getByRole('button', { name: 'Umfrage erstellen' })).toBeDisabled()
  })

  test('Senden ist bei doppelten Optionen (Trim + Groß-/Kleinschreibung) deaktiviert', () => {
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={() => {}} />)
    fillQuestion('Pizza oder Nudeln?')
    const [opt1, opt2] = optionInputs()
    fireEvent.change(opt1, { target: { value: 'Pizza' } })
    fireEvent.change(opt2, { target: { value: ' pizza ' } })
    expect(screen.getByRole('button', { name: 'Umfrage erstellen' })).toBeDisabled()
  })

  test('höchstens 10 Optionsfelder — "Option hinzufügen" verschwindet danach', () => {
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={() => {}} />)
    const addButton = () => screen.queryByText('Option hinzufügen')
    expect(optionInputs()).toHaveLength(2)
    for (let i = 0; i < 8; i++) {
      fireEvent.click(addButton()!)
    }
    expect(optionInputs()).toHaveLength(10)
    expect(addButton()).not.toBeInTheDocument()
  })

  test('Entfernen (X) ist erst ab dem dritten Feld verfügbar', () => {
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={() => {}} />)
    expect(screen.queryByLabelText('Option 1 entfernen')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Option 2 entfernen')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Option hinzufügen'))
    expect(screen.getByLabelText('Option 3 entfernen')).toBeInTheDocument()
    fireEvent.click(screen.getByLabelText('Option 3 entfernen'))
    expect(optionInputs()).toHaveLength(2)
  })

  test('sendet nur nicht-leere, getrimmte Optionen in der Reihenfolge der Felder', async () => {
    const onCreated = vi.fn()
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={onCreated} />)
    fillQuestion(' Wer fährt am Samstag? ')
    const [opt1, opt2] = optionInputs()
    fireEvent.change(opt1, { target: { value: ' Ich ' } })
    fireEvent.change(opt2, { target: { value: 'Ich nicht' } })
    fireEvent.click(screen.getByText('Option hinzufügen'))
    // drittes Feld bleibt leer -> wird verworfen
    fireEvent.click(screen.getByText('Option hinzufügen'))
    fireEvent.change(optionInputs()[3], { target: { value: 'Nur Hinfahrt' } })

    const submitBtn = screen.getByRole('button', { name: 'Umfrage erstellen' })
    expect(submitBtn).not.toBeDisabled()
    fireEvent.click(submitBtn)

    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/chat/conversations/7/polls', {
      question: 'Wer fährt am Samstag?',
      options: ['Ich', 'Ich nicht', 'Nur Hinfahrt'],
      allowMultiple: false,
    }))
    await waitFor(() => expect(onCreated).toHaveBeenCalled())
  })

  test('Mehrfachantworten-Schalter wandert in den Request', async () => {
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={() => {}} />)
    fillQuestion('Wann passt es dir?')
    const [opt1, opt2] = optionInputs()
    fireEvent.change(opt1, { target: { value: 'Samstag' } })
    fireEvent.change(opt2, { target: { value: 'Sonntag' } })
    fireEvent.click(screen.getByLabelText('Mehrfachantworten erlauben'))
    fireEvent.click(screen.getByRole('button', { name: 'Umfrage erstellen' }))
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith('/chat/conversations/7/polls', {
      question: 'Wann passt es dir?',
      options: ['Samstag', 'Sonntag'],
      allowMultiple: true,
    }))
  })

  test('Fehler vom Server wird angezeigt, Modal bleibt offen', async () => {
    mockPost.mockRejectedValue({ isAxiosError: true, response: { data: { error: 'zu viele Optionen' } } })
    const onCreated = vi.fn()
    render(<ChatPollCreateModal convId={7} onClose={() => {}} onCreated={onCreated} />)
    fillQuestion('Frage?')
    const [opt1, opt2] = optionInputs()
    fireEvent.change(opt1, { target: { value: 'A' } })
    fireEvent.change(opt2, { target: { value: 'B' } })
    fireEvent.click(screen.getByRole('button', { name: 'Umfrage erstellen' }))
    await waitFor(() => expect(screen.getByText('zu viele Optionen')).toBeInTheDocument())
    expect(onCreated).not.toHaveBeenCalled()
  })
})
