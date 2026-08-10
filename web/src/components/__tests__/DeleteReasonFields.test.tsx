import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import DeleteReasonFields, { deletionPayload } from '../DeleteReasonFields'
import { WithCapabilities } from '../../test/authCtx'

function renderFields(caps: string[], onSilentChange = vi.fn()) {
  return render(
    <WithCapabilities caps={caps}>
      <DeleteReasonFields
        idPrefix="t"
        reason=""
        onReasonChange={() => {}}
        silent={false}
        onSilentChange={onSilentChange}
      />
    </WithCapabilities>,
  )
}

describe('DeleteReasonFields — Grundfeld', () => {
  test('ist unabhängig von Capabilities immer vorhanden', () => {
    renderFields([])
    expect(screen.getByLabelText(/Grund/)).toBeInTheDocument()
  })
})

describe('DeleteReasonFields — Capability-Gate für „Ohne Benachrichtigung löschen"', () => {
  test('Häkchen fehlt ohne suppress_event_notification', () => {
    // manage_games/manage_trainings dürfen das Häkchen NICHT freischalten:
    // beide hat der Trainer, das Stummschaltrecht liegt nur beim Vorstand.
    renderFields(['manage_games', 'manage_trainings', 'manage_duties'])
    expect(screen.queryByLabelText('Ohne Benachrichtigung löschen')).not.toBeInTheDocument()
  })

  test('Häkchen ist da mit suppress_event_notification', () => {
    renderFields(['suppress_event_notification'])
    const box = screen.getByLabelText('Ohne Benachrichtigung löschen')
    expect(box).toBeInTheDocument()
    expect((box as HTMLInputElement).type).toBe('checkbox')
  })

  test('Klick auf das Häkchen meldet true nach oben', () => {
    const onSilentChange = vi.fn()
    renderFields(['suppress_event_notification'], onSilentChange)
    fireEvent.click(screen.getByLabelText('Ohne Benachrichtigung löschen'))
    expect(onSilentChange).toHaveBeenCalledWith(true)
  })
})

describe('deletionPayload', () => {
  test('trimmt den Grund und sendet ihn zusammen mit silent', () => {
    expect(deletionPayload('  Halle gesperrt  ', true)).toEqual({ reason: 'Halle gesperrt', silent: true })
  })

  test('leeres Feld ergibt den leeren String — der Server behandelt ihn als „kein Grund"', () => {
    expect(deletionPayload('   ', false)).toEqual({ reason: '', silent: false })
  })
})
