/**
 * Einstellungen-Tabs sind capability-gesteuert (nie über role/clubFunctions direkt):
 *   Verein / Beiträge → manage_club / manage_fees → kassierer-like (kassierer + vorstand + admin)
 *   Saisons / Altersklassen → manage_seasons → vorstand-like (vorstand + admin)
 * Quelle: internal/policy/rules.go (Capabilities). Kassierer sieht nur Verein + Beiträge,
 * Vorstand/Admin alle vier Tabs.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import AdminSettingsPage from '../AdminSettingsPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

// manage_club + manage_fees
const KASSIERER_LIKE = ['admin', 'vorstand', 'vorstand_elternteil', 'kassierer']
// manage_seasons
const VORSTAND_LIKE = ['admin', 'vorstand', 'vorstand_elternteil']

const MOCKS = [
  { url: /\/club/, data: {} },
  { url: /\/seasons/, data: [] },
  { url: /\/fee-rates/, data: [] },
  { url: /age-class-rules/, data: [] },
]

function tabButton(label: string): HTMLElement | null {
  return screen.queryByRole('button', { name: label })
}

describe('AdminSettingsPage — capability-gesteuerte Tab-Sichtbarkeit', () => {
  test.each(PERSONAS)('Persona $id', async (persona) => {
    renderAsPersona(<AdminSettingsPage />, persona.id, { mocks: MOCKS })
    await flushAsync()

    const expectVerein = KASSIERER_LIKE.includes(persona.id)
    const expectBeitraege = KASSIERER_LIKE.includes(persona.id)
    const expectSaisons = VORSTAND_LIKE.includes(persona.id)
    const expectAltersklassen = VORSTAND_LIKE.includes(persona.id)

    expect(Boolean(tabButton('Verein')), `${persona.id}: Verein-Tab`).toBe(expectVerein)
    expect(Boolean(tabButton('Beiträge')), `${persona.id}: Beiträge-Tab`).toBe(expectBeitraege)
    expect(Boolean(tabButton('Saisons')), `${persona.id}: Saisons-Tab`).toBe(expectSaisons)
    expect(Boolean(tabButton('Altersklassen')), `${persona.id}: Altersklassen-Tab`).toBe(expectAltersklassen)
  })
})
