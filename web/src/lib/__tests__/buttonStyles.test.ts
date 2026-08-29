import { describe, expect, it } from 'vitest'
import {
  BTN_DANGER,
  BTN_PRIMARY,
  BTN_SECONDARY,
  BTN_SMALL,
  HEADER_CTRL,
  HEADER_CTRL_ICON,
  HEADER_DANGER,
  HEADER_FIELD,
  HEADER_GHOST,
  HEADER_H,
  HEADER_NEUTRAL,
  HEADER_PRIMARY,
  HEADER_SPLIT_CARET,
  HEADER_SPLIT_MAIN,
} from '../buttonStyles'

/**
 * Die Zusage der Capability `component-standards` ist, dass alle Bedienelemente
 * einer Kopfzeile gleich hoch sind. Geprüft wird sie hier auf den Klassen, nicht
 * an gerenderten Pixeln: jsdom rechnet kein Layout, `getBoundingClientRect()`
 * liefert dort überall 0. Die Pixel-Aussage hängt an Tailwind und gehört in die
 * Sichtprüfung bzw. den E2E-Lauf.
 */

const STRUCTURAL = { HEADER_CTRL, HEADER_CTRL_ICON, HEADER_SPLIT_MAIN, HEADER_SPLIT_CARET, HEADER_FIELD }
const COLOR_SETS = { HEADER_PRIMARY, HEADER_NEUTRAL, HEADER_DANGER, HEADER_GHOST }
const FORM = { BTN_PRIMARY, BTN_SECONDARY, BTN_DANGER, BTN_SMALL }

const tokens = (s: string) => s.split(/\s+/).filter(Boolean)

describe('Header-Controls', () => {
  it.each(Object.entries(STRUCTURAL))('%s trägt die gemeinsame Höhe', (_name, cls) => {
    for (const t of tokens(HEADER_H)) expect(tokens(cls)).toContain(t)
  })

  it.each(Object.entries(STRUCTURAL))('%s legt keine zweite Höhe fest', (_name, cls) => {
    const heights = tokens(cls).filter(t => /^(sm:)?h-/.test(t))
    expect(heights.sort()).toEqual(tokens(HEADER_H).sort())
  })

  // Der Grund für die Trennung Basis/Farbsatz: ein Maß im Farbsatz würde je nach
  // Zustand eine andere Größe ergeben — genau der Fehler, den dieser Change behebt.
  it.each(Object.entries(COLOR_SETS))('%s enthält kein Maß, nur Farbe', (_name, cls) => {
    const sizing = tokens(cls).filter(t => /^(sm:)?(h-|w-|p[xy]?-|text-(xs|sm|base|lg)$|rounded)/.test(t))
    expect(sizing).toEqual([])
  })

  it('Split-Hälften runden nur ihre Außenkante', () => {
    expect(tokens(HEADER_SPLIT_MAIN)).toContain('rounded-l-md')
    expect(tokens(HEADER_SPLIT_MAIN)).not.toContain('rounded-md')
    expect(tokens(HEADER_SPLIT_CARET)).toContain('rounded-r-md')
    expect(tokens(HEADER_SPLIT_CARET)).not.toContain('rounded-md')
  })

  // Ohne Rahmen wäre ein Control 2px flacher als seine Nachbarn — bei fixer Höhe
  // nicht mehr, aber die Kante muss trotzdem an derselben Stelle sitzen.
  it.each(Object.entries(STRUCTURAL))('%s zeichnet einen Rahmen', (_name, cls) => {
    expect(tokens(cls)).toContain('border')
  })
})

describe('Formular-Aktionen bleiben eine eigene Rolle', () => {
  it.each(Object.entries(FORM))('%s bemaßt sich über Padding, nicht über feste Höhe', (_name, cls) => {
    expect(tokens(cls).filter(t => /^(sm:)?h-/.test(t))).toEqual([])
    expect(tokens(cls).some(t => /^py-/.test(t))).toBe(true)
  })

  it.each(Object.entries({ ...FORM, ...STRUCTURAL }))(
    '%s hat einen einheitlichen Disabled-Zustand',
    (name, cls) => {
      if (name === 'HEADER_FIELD') return // Eingabefeld, kein Button
      expect(tokens(cls)).toContain('disabled:opacity-40')
      expect(tokens(cls)).toContain('disabled:cursor-not-allowed')
    },
  )
})
