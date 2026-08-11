/**
 * Die Dienst-Vorschau in Schritt 4 des Anlege-Wizards muss dem entsprechen, was die
 * Regeneration real anlegt. Dafür braucht der Server die Teams des geplanten Events —
 * ohne sie filtert er bewusst nicht und zeigt auch team-eingeschränkte Einträge.
 *
 * Quelle: openspec/changes/duty-template-team-scope/specs/duty-template-team-scope/spec.md
 * — Requirement "Slot-Vorschau spiegelt die Team-Einschränkung".
 */
import { describe, test, expect } from 'vitest'
import { buildPreviewUrl } from '../dutyPreview'

const BASE = { templateId: 4, time: '14:00', date: '2026-09-19', endTime: '18:00' }

describe('buildPreviewUrl', () => {
  test('hängt die gewählten Kaderteams als team_ids an', () => {
    const url = buildPreviewUrl({ ...BASE, eventType: 'heim', teamIds: [11, 13] })
    expect(url).toContain('team_ids=11%2C13')
  })

  test('Heimspiel trägt date, aber kein end_time', () => {
    const url = buildPreviewUrl({ ...BASE, eventType: 'heim', teamIds: [11] })
    expect(url).toContain('date=2026-09-19')
    expect(url).not.toContain('end_time')
  })

  test('Auswärtsspiel trägt weder date noch end_time, aber die Teams', () => {
    const url = buildPreviewUrl({ ...BASE, eventType: 'auswärts', teamIds: [11] })
    expect(url).not.toContain('date=')
    expect(url).not.toContain('end_time')
    expect(url).toContain('team_ids=11')
  })

  test('generisches Event trägt end_time', () => {
    const url = buildPreviewUrl({ ...BASE, eventType: 'generisch', teamIds: [11] })
    expect(url).toContain('end_time=18%3A00')
  })

  test('ohne Teams bleibt der Parameter weg (Server filtert dann nicht)', () => {
    const url = buildPreviewUrl({ ...BASE, eventType: 'heim', teamIds: [] })
    expect(url).not.toContain('team_ids')
  })
})
