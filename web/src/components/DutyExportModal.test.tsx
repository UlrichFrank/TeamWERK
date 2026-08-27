import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import DutyExportModal from './DutyExportModal'

const mockGet = vi.fn((..._args: unknown[]) => Promise.resolve({ data: new Blob(['x']) }))
vi.mock('../lib/api', () => ({
  api: { get: (...args: unknown[]) => mockGet(...args) },
}))

describe('DutyExportModal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockGet.mockResolvedValue({ data: new Blob(['Datum;Dienst\n']) })
    // jsdom kennt createObjectURL nicht — der Download-Pfad braucht beides.
    URL.createObjectURL = vi.fn(() => 'blob:test')
    URL.revokeObjectURL = vi.fn()
  })

  test('ist mit dem angezeigten Monat vorbelegt und exportiert genau diesen Zeitraum', async () => {
    render(
      <DutyExportModal isOpen monthStart="2026-09-01" monthEnd="2026-09-30" onClose={() => {}} />,
    )
    expect((screen.getByLabelText('Von') as HTMLInputElement).value).toBe('2026-09-01')
    expect((screen.getByLabelText('Bis') as HTMLInputElement).value).toBe('2026-09-30')

    fireEvent.click(screen.getByRole('button', { name: /Herunterladen/ }))
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith(
      '/duty-slots/export?from=2026-09-01&to=2026-09-30',
      { responseType: 'blob' },
    ))
  })

  test('geänderter Zeitraum landet in der Anfrage', async () => {
    render(
      <DutyExportModal isOpen monthStart="2026-09-01" monthEnd="2026-09-30" onClose={() => {}} />,
    )
    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-09-15' } })
    fireEvent.change(screen.getByLabelText('Bis'), { target: { value: '2027-01-31' } })
    fireEvent.click(screen.getByRole('button', { name: /Herunterladen/ }))
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith(
      '/duty-slots/export?from=2026-09-15&to=2027-01-31',
      { responseType: 'blob' },
    ))
  })

  test('verdrehter Zeitraum sperrt den Download, statt einen 400 zu provozieren', () => {
    render(
      <DutyExportModal isOpen monthStart="2026-09-01" monthEnd="2026-09-30" onClose={() => {}} />,
    )
    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-10-05' } })
    expect(screen.getByRole('button', { name: /Herunterladen/ })).toBeDisabled()
    expect(screen.getByText(/muss vor/)).toBeInTheDocument()
    expect(mockGet).not.toHaveBeenCalled()
  })

  test('403 wird als Berechtigungsfehler angezeigt und der Dialog bleibt offen', async () => {
    const onClose = vi.fn()
    mockGet.mockRejectedValue({ isAxiosError: true, response: { status: 403 } })
    render(
      <DutyExportModal isOpen monthStart="2026-09-01" monthEnd="2026-09-30" onClose={onClose} />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Herunterladen/ }))
    await waitFor(() => expect(screen.getByText(/Keine Berechtigung/)).toBeInTheDocument())
    expect(onClose).not.toHaveBeenCalled()
  })
})
