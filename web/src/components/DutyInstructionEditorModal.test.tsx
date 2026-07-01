import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import DutyInstructionEditorModal from './DutyInstructionEditorModal'
import { DUTY_INSTRUCTION_TEMPLATE } from '../lib/dutyInstructionTemplate'

vi.mock('../lib/api', () => ({ api: { put: vi.fn(() => Promise.resolve({ data: {} })) } }))

describe('DutyInstructionEditorModal', () => {
  test('prefills example on empty instruction and disables save', () => {
    render(
      <DutyInstructionEditorModal
        dutyTypeId={1}
        dutyTypeName="Kasse"
        currentMarkdown=""
        onClose={() => {}}
        onSaved={() => {}}
      />,
    )
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    expect(textarea.value).toBe(DUTY_INSTRUCTION_TEMPLATE)
    const save = screen.getByRole('button', { name: /^Speichern$/ })
    expect(save).toBeDisabled()
  })

  test('save enables once the textarea changes', () => {
    render(
      <DutyInstructionEditorModal
        dutyTypeId={2}
        dutyTypeName="Aufbau"
        currentMarkdown=""
        onClose={() => {}}
        onSaved={() => {}}
      />,
    )
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    fireEvent.change(textarea, { target: { value: 'geändert' } })
    const save = screen.getByRole('button', { name: /^Speichern$/ })
    expect(save).not.toBeDisabled()
  })

  test('uses existing markdown instead of template when instruction present', () => {
    render(
      <DutyInstructionEditorModal
        dutyTypeId={3}
        dutyTypeName="Wischer"
        currentMarkdown="## Bestehende Anleitung"
        onClose={() => {}}
        onSaved={() => {}}
      />,
    )
    const textarea = screen.getByRole('textbox') as HTMLTextAreaElement
    expect(textarea.value).toBe('## Bestehende Anleitung')
  })
})
