import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import TrainingDiaryEntryForm from '../TrainingDiaryEntryForm'

function todayISO() {
  return new Date().toISOString().slice(0, 10)
}

describe('TrainingDiaryEntryForm', () => {
  test('blendet das Freitextfeld nur bei "sonstiges" ein', () => {
    render(<TrainingDiaryEntryForm onSubmit={vi.fn()} />)
    expect(screen.queryByLabelText('Art des Trainings')).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Art'), { target: { value: 'sonstiges' } })
    expect(screen.getByLabelText('Art des Trainings')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Art'), { target: { value: 'kraft' } })
    expect(screen.queryByLabelText('Art des Trainings')).not.toBeInTheDocument()
  })

  test('blockiert das Speichern bei "sonstiges" ohne Freitext', () => {
    const onSubmit = vi.fn()
    render(<TrainingDiaryEntryForm onSubmit={onSubmit} />)
    fireEvent.change(screen.getByLabelText('Art'), { target: { value: 'sonstiges' } })

    expect(screen.getByRole('button', { name: 'Eintragen' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Eintragen' }))
    expect(onSubmit).not.toHaveBeenCalled()
  })

  test('sendet den Freitext mit, sobald er ausgefüllt ist', () => {
    const onSubmit = vi.fn()
    render(<TrainingDiaryEntryForm onSubmit={onSubmit} />)
    fireEvent.change(screen.getByLabelText('Art'), { target: { value: 'sonstiges' } })
    fireEvent.change(screen.getByLabelText('Art des Trainings'), { target: { value: 'Schwimmen' } })
    fireEvent.click(screen.getByRole('button', { name: 'Eintragen' }))

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ kind: 'sonstiges', kind_custom: 'Schwimmen' }),
    )
  })

  test('lässt kein Datum in der Zukunft zu', () => {
    render(<TrainingDiaryEntryForm onSubmit={vi.fn()} />)
    // Der Browser erzwingt max= selbst; der Test sichert das Attribut ab,
    // damit es beim Umbauen nicht verloren geht.
    expect(screen.getByLabelText('Datum')).toHaveAttribute('max', todayISO())
  })

  test('lehnt unplausible Dauern ab', () => {
    const onSubmit = vi.fn()
    render(<TrainingDiaryEntryForm onSubmit={onSubmit} />)
    fireEvent.change(screen.getByLabelText('Dauer (Minuten)'), { target: { value: '1000' } })

    expect(screen.getByRole('button', { name: 'Eintragen' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Eintragen' }))
    expect(onSubmit).not.toHaveBeenCalled()
  })

  test('übergibt die gewählte Intensität', () => {
    const onSubmit = vi.fn()
    render(<TrainingDiaryEntryForm onSubmit={onSubmit} />)
    fireEvent.click(screen.getByRole('button', { name: '8' }))
    fireEvent.click(screen.getByRole('button', { name: 'Eintragen' }))

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ rpe: 8 }))
  })
})
