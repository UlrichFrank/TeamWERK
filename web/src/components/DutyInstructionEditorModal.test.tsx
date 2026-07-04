import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import DutyInstructionEditorModal from './DutyInstructionEditorModal'
import { DUTY_INSTRUCTION_TEMPLATE } from '../lib/dutyInstructionTemplate'

const mockGet = vi.fn((..._args: unknown[]) => Promise.resolve({ data: {} }))
const mockPut = vi.fn((..._args: unknown[]) => Promise.resolve({ data: {} }))
vi.mock('../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    put: (...args: unknown[]) => mockPut(...args),
  },
}))

describe('DutyInstructionEditorModal', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPut.mockClear()
  })

  test('lädt Volltext aus dem Detail-Pfad und prefills Template bei leerer Anleitung', async () => {
    mockGet.mockResolvedValue({ data: { id: 1, name: 'Kasse', instruction_md: '' } })
    render(
      <DutyInstructionEditorModal dutyTypeId={1} dutyTypeName="Kasse" onClose={() => {}} onSaved={() => {}} />,
    )
    // Detail-Route wurde abgerufen (nicht die Liste).
    await waitFor(() => expect(mockGet).toHaveBeenCalledWith('/duty-types/1/instruction'))
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    await waitFor(() => expect(textarea.value).toBe(DUTY_INSTRUCTION_TEMPLATE))
    const save = screen.getByRole('button', { name: /^Speichern$/ })
    expect(save).toBeDisabled()
  })

  test('save aktiviert sich nach einer Änderung', async () => {
    mockGet.mockResolvedValue({ data: { id: 2, name: 'Aufbau', instruction_md: '' } })
    render(
      <DutyInstructionEditorModal dutyTypeId={2} dutyTypeName="Aufbau" onClose={() => {}} onSaved={() => {}} />,
    )
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    await waitFor(() => expect(textarea).not.toBeDisabled())
    fireEvent.change(textarea, { target: { value: 'geändert' } })
    const save = screen.getByRole('button', { name: /^Speichern$/ })
    expect(save).not.toBeDisabled()
  })

  test('übernimmt bestehenden Volltext aus dem Detail-Pfad', async () => {
    mockGet.mockResolvedValue({ data: { id: 3, name: 'Wischer', instruction_md: '## Bestehende Anleitung' } })
    render(
      <DutyInstructionEditorModal dutyTypeId={3} dutyTypeName="Wischer" onClose={() => {}} onSaved={() => {}} />,
    )
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    await waitFor(() => expect(textarea.value).toBe('## Bestehende Anleitung'))
  })
})
