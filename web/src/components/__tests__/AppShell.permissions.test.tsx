/**
 * AppShell Sidebar-Navigations-Sichtbarkeit pro Persona.
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Sidebar-Navigations-Items"
 *
 * Backend-Äquivalent: internal/permissions/matrix_test.go
 */
import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen } from '@testing-library/react'
import AppShell from '../AppShell'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

vi.mock('../../hooks/usePushSubscription', () => ({
  usePushSubscription: vi.fn(),
}))
vi.mock('../../hooks/useChatEvents', () => ({
  useChatEvents: vi.fn(),
}))
vi.mock('../../hooks/useLiveUpdates', () => ({
  useLiveUpdates: vi.fn(),
}))
vi.mock('../../contexts/VersionContext', () => ({
  useVersion: () => ({ version: null, updateAvailable: false, latestVersion: null }),
  VersionProvider: ({ children }: { children: React.ReactNode }) => children,
}))

beforeAll(() => {
  vi.spyOn(window.localStorage.__proto__, 'getItem').mockReturnValue(null)
  vi.spyOn(window.localStorage.__proto__, 'setItem').mockImplementation(() => {})
})

// Items sichtbar für ALLE Personas (roles: [])
const ALWAYS_VISIBLE = ['Dashboard', 'Kalender', 'Termine', 'Mein Team', 'Dokumente', 'Dienste', 'Mitfahrten', 'Nachrichten']

// Verwaltungs-Items und ihre erlaubten Personas
const VERWALTUNG_ITEMS: { label: string; allowedIds: string[] }[] = [
  {
    label: 'Nutzerverwaltung',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Mitglieder',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Kader',
    allowedIds: [
      'admin', 'vorstand', 'vorstand_elternteil',
      'trainer', 'trainer_elternteil',
      'sportliche_leitung', 'sportliche_leitung_elternteil',
    ],
  },
  {
    label: 'Diensttypen',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Dienstplan-Vorlagen',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Veranstaltungsorte',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Einstellungen',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
]

// Personas für die KEIN Verwaltungs-Item sichtbar ist → kein Modul-Header
const NO_VERWALTUNG_IDS = ['vorstand_beisitzer', 'kassierer', 'spieler', 'elternteil']

describe('AppShell — Sidebar-Items pro Persona', () => {
  test.each(PERSONAS)('Persona $id: immer sichtbare Items vorhanden', async (persona) => {
    renderAsPersona(<AppShell />, persona.id, { route: '/' })
    await flushAsync()
    for (const label of ALWAYS_VISIBLE) {
      expect(screen.queryByText(label), `"${label}" für ${persona.id}`).not.toBeNull()
    }
  })

  test.each(PERSONAS)('Persona $id: "Mein Profil" (nicht für admin)', async (persona) => {
    renderAsPersona(<AppShell />, persona.id, { route: '/' })
    await flushAsync()
    if (persona.id === 'admin') {
      expect(screen.queryByText('Mein Profil')).toBeNull()
    } else {
      expect(screen.queryByText('Mein Profil')).not.toBeNull()
    }
  })

  describe('Verwaltungs-Modul sichtbar/versteckt', () => {
    test.each(PERSONAS)('Persona $id: Modul-Header "Verwaltung"', async (persona) => {
      renderAsPersona(<AppShell />, persona.id, { route: '/' })
      await flushAsync()
      const header = screen.queryByRole('button', { name: /Verwaltung/i })
      if (NO_VERWALTUNG_IDS.includes(persona.id)) {
        // Kein Item sichtbar → Header muss fehlen (§6.2 Drift-Schutz)
        expect(header, `Verwaltungs-Header für ${persona.id} sollte nicht gerendert werden`).toBeNull()
      } else {
        expect(header, `Verwaltungs-Header für ${persona.id} muss vorhanden sein`).not.toBeNull()
      }
    })

    for (const item of VERWALTUNG_ITEMS) {
      test.each(PERSONAS)(`Persona $id: "${item.label}"`, async (persona) => {
        renderAsPersona(<AppShell />, persona.id, { route: '/' })
        await flushAsync()
        const el = screen.queryByText(item.label)
        if (item.allowedIds.includes(persona.id)) {
          expect(el, `"${item.label}" muss für ${persona.id} sichtbar sein`).not.toBeNull()
        } else {
          expect(el, `"${item.label}" darf für ${persona.id} NICHT sichtbar sein`).toBeNull()
        }
      })
    }
  })
})
