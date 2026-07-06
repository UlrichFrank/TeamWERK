/**
 * AppShell Sidebar-Navigations-Sichtbarkeit pro Persona.
 * Quelle: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §"Sidebar-Navigations-Items"
 *
 * Backend-Äquivalent: internal/permissions/matrix_test.go
 */
import { describe, test, expect, vi, beforeAll } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import AppShell from '../AppShell'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { PERSONAS } from '../../test/personas'

// Die Sidebar ist ein Akkordeon: es ist immer genau EIN Modul offen (Default auf
// Route '/' ist "Nutzer"), die Items geschlossener Module sind nicht im DOM.
// Für Sichtbarkeits-Assertions muss daher zuerst das Modul des Items geöffnet
// werden. Der Modul-Header (ein <button>) ist unabhängig vom Auf-/Zu-Zustand
// immer gerendert, solange das Modul mindestens ein sichtbares Item hat.
async function queryItemInModule(moduleName: string, itemLabel: string): Promise<HTMLElement | null> {
  let el = screen.queryByText(itemLabel)
  if (el) return el
  const header = screen.queryByRole('button', { name: moduleName })
  if (!header) return null // Persona hat kein Item in diesem Modul → Header fehlt
  fireEvent.click(header)
  await flushAsync()
  el = screen.queryByText(itemLabel)
  return el
}

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

// Items sichtbar für ALLE Personas (roles: []) — mit ihrem Akkordeon-Modul,
// damit der Test das Modul vor der Assertion aufklappen kann.
const ALWAYS_VISIBLE: { label: string; module: string }[] = [
  { label: 'Dashboard', module: 'Nutzer' },
  { label: 'Kalender', module: 'Spielbetrieb' },
  { label: 'Termine', module: 'Spielbetrieb' },
  { label: 'Mein Team', module: 'Verein' },
  { label: 'Dokumente', module: 'Verein' },
  { label: 'Dienste', module: 'Verein' },
  { label: 'Mitfahrten', module: 'Verein' },
  { label: 'Nachrichten', module: 'Verein' },
]

// Verwaltungs-Items und ihre erlaubten Personas
const VERWALTUNG_ITEMS: { label: string; allowedIds: string[] }[] = [
  {
    label: 'Nutzerverwaltung',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil'],
  },
  {
    label: 'Mitglieder',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil', 'kassierer'],
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
    // Beitragslauf: kassierer-like (kassierer + vorstand + admin)
    label: 'Beitragslauf',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil', 'kassierer'],
  },
  {
    // Einstellungen: kassierer-like (Tabs werden in der Seite capability-gefiltert)
    label: 'Einstellungen',
    allowedIds: ['admin', 'vorstand', 'vorstand_elternteil', 'kassierer'],
  },
]

// Personas für die KEIN Verwaltungs-Item sichtbar ist → kein Modul-Header.
// Kassierer sieht Mitglieder/Beitragslauf/Einstellungen und ist daher NICHT mehr hier.
const NO_VERWALTUNG_IDS = ['vorstand_beisitzer', 'spieler', 'elternteil']

describe('AppShell — Sidebar-Items pro Persona', () => {
  test.each(PERSONAS)('Persona $id: immer sichtbare Items vorhanden', async (persona) => {
    renderAsPersona(<AppShell />, persona.id, { route: '/' })
    await flushAsync()
    for (const { label, module } of ALWAYS_VISIBLE) {
      const el = await queryItemInModule(module, label)
      expect(el, `"${label}" für ${persona.id}`).not.toBeNull()
    }
  })

  test.each(PERSONAS)('Persona $id: "Mein Profil" (nicht für admin)', async (persona) => {
    renderAsPersona(<AppShell />, persona.id, { route: '/' })
    await flushAsync()
    const el = await queryItemInModule('Nutzer', 'Mein Profil')
    if (persona.id === 'admin') {
      expect(el).toBeNull()
    } else {
      expect(el).not.toBeNull()
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
        const el = await queryItemInModule('Verwaltung', item.label)
        if (item.allowedIds.includes(persona.id)) {
          expect(el, `"${item.label}" muss für ${persona.id} sichtbar sein`).not.toBeNull()
        } else {
          expect(el, `"${item.label}" darf für ${persona.id} NICHT sichtbar sein`).toBeNull()
        }
      })
    }
  })
})
