/**
 * MembersPage inline gate: isAdmin = admin || vorstand
 * Steuert den "+ Neu"-Button (Mitglied anlegen).
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Inline-Gates auf Pages"
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import MembersPage from '../MembersPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const ALLOWED_IDS = ['admin', 'vorstand', 'vorstand_elternteil']

describe('MembersPage — isAdmin-Gate: "+ Neu"-Button', () => {
  test.each(PERSONAS)('Persona $id', (persona) => {
    renderAsPersona(<MembersPage />, persona.id)
    const btn = screen.queryByText('+ Neu')
    if (ALLOWED_IDS.includes(persona.id)) {
      expect(btn, `"+ Neu" muss für ${persona.id} sichtbar sein`).not.toBeNull()
    } else {
      expect(btn, `"+ Neu" darf für ${persona.id} NICHT sichtbar sein`).toBeNull()
    }
  })
})
