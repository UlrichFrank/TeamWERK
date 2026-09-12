import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import PasswordChangeModal from '../PasswordChangeModal'

describe('PasswordChangeModal', () => {
  test('trägt role=dialog mit aria-labelledby auf den Titel', () => {
    render(<PasswordChangeModal onClose={() => {}} logout={() => {}} />)

    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    const labelledBy = dialog.getAttribute('aria-labelledby')
    expect(labelledBy).toBeTruthy()
    expect(document.getElementById(labelledBy!)).toHaveTextContent('Passwort ändern')
  })

  test('Fokus liegt initial im ersten Feld (Aktuelles Passwort)', () => {
    const { container } = render(<PasswordChangeModal onClose={() => {}} logout={() => {}} />)

    // Die Labels sind (bestehend, außerhalb dieses Changes) nicht per
    // htmlFor/id mit dem Feld verknüpft — deshalb hier über das
    // autoComplete-Attribut statt getByLabelText selektiert.
    const currentPasswordField = container.querySelector('input[autocomplete="current-password"]')
    expect(currentPasswordField).toHaveFocus()
  })
})
