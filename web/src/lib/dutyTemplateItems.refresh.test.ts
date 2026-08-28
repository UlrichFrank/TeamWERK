import { describe, test, expect } from 'vitest'
import { refreshItemsFromDutyTypes } from './dutyTemplateItems'

const TYPES = [
  {
    id: 5, default_anchor: 'start' as const, default_offset_minutes: -30, hours_value: 1.5,
    duration_mode: 'dynamisch' as const, end_anchor: 'start' as const, end_offset_minutes: 40,
    end_at_next_duty: true,
    audiences: ['eltern'],
  },
  {
    id: 8, default_anchor: 'end' as const, default_offset_minutes: 5, hours_value: 0.5,
    duration_mode: 'absolut' as const, end_anchor: 'end' as const, end_offset_minutes: 0,
    end_at_next_duty: false,
    audiences: [],
  },
]

function item(over: Partial<{
  duty_type_id: number
  anchor: 'start' | 'end'
  offset_minutes: number
  hours_value: number
  duration_mode: 'absolut' | 'dynamisch'
  end_anchor: 'start' | 'end'
  end_offset_minutes: number
  end_at_next_duty: boolean
  audiences: string[]
}> = {}) {
  return {
    duty_type_id: 5, anchor: 'start' as const, offset_minutes: -30, hours_value: 1.5,
    duration_mode: 'dynamisch' as const, end_anchor: 'start' as const, end_offset_minutes: 40,
    end_at_next_duty: true,
    audiences: ['eltern'],
    ...over,
  }
}

describe('refreshItemsFromDutyTypes', () => {
  test('holt die aktuelle Dauer zurück in die Zeile', () => {
    const { items, changed } = refreshItemsFromDutyTypes([item({ hours_value: 1.0 })], TYPES)
    expect(changed).toBe(1)
    expect(items[0].hours_value).toBe(1.5)
  })

  test('meldet 0, wenn schon alles übereinstimmt', () => {
    const before = [item()]
    const { items, changed } = refreshItemsFromDutyTypes(before, TYPES)
    expect(changed).toBe(0)
    // Unveränderte Zeilen behalten ihre Identität — kein unnötiger Re-Render.
    expect(items[0]).toBe(before[0])
  })

  test('frischt auch Anker, Versatz und Zielgruppe auf', () => {
    const { items } = refreshItemsFromDutyTypes(
      [item({ duty_type_id: 8, anchor: 'start', offset_minutes: 0, hours_value: 2, audiences: ['spieler'] })],
      TYPES,
    )
    expect(items[0]).toMatchObject({ anchor: 'end', offset_minutes: 5, hours_value: 0.5, audiences: [] })
  })

  test('lässt Zeilen ohne passenden Diensttyp unangetastet', () => {
    // duty_type_id 0 = noch nicht ausgewählt; 99 = Typ gelöscht.
    const before = [item({ duty_type_id: 0, hours_value: 9 }), item({ duty_type_id: 99, hours_value: 9 })]
    const { items, changed } = refreshItemsFromDutyTypes(before, TYPES)
    expect(changed).toBe(0)
    expect(items[0].hours_value).toBe(9)
    expect(items[1].hours_value).toBe(9)
  })

  test('zählt nur tatsächlich geänderte Zeilen', () => {
    const { changed } = refreshItemsFromDutyTypes([item(), item({ hours_value: 1.0 })], TYPES)
    expect(changed).toBe(1)
  })

  // openspec/changes/dienst-dauer-dynamisch, Aufgabe 7.2 — der wahrscheinlichste
  // Fehler laut design.md: ohne die drei neuen Felder im Vergleich UND in der
  // Übernahme bliebe eine Zeile nach dem Auffrischen auf 'absolut' stehen, obwohl
  // der Diensttyp längst auf 'dynamisch' umgestellt wurde (Modus und Dauer aus
  // zwei verschiedenen Ständen).
  test('ein von absolut auf dynamisch gewechselter Diensttyp wird vollständig übernommen', () => {
    const before = [item({
      duty_type_id: 8,
      anchor: 'end', offset_minutes: 5, hours_value: 0.5,
      duration_mode: 'absolut', end_anchor: 'end', end_offset_minutes: 0,
      audiences: [],
    })]
    // Der Diensttyp 8 ist inzwischen auf 'dynamisch' umgestellt worden.
    const typesNow = [
      TYPES[0],
      { ...TYPES[1], duration_mode: 'dynamisch' as const, end_anchor: 'start' as const, end_offset_minutes: 25 },
    ]
    const { items, changed } = refreshItemsFromDutyTypes(before, typesNow)
    expect(changed).toBe(1)
    expect(items[0]).toMatchObject({ duration_mode: 'dynamisch', end_anchor: 'start', end_offset_minutes: 25 })
  })

  // dienst-abloesung: das Kennzeichen gehört zu den Feldern, die eine Zeile per
  // Copy-on-pick vom Diensttyp übernimmt — sonst bliebe eine Vorlage nach dem
  // Umstellen des Diensttyps stumm beim alten Verhalten.
  test('frischt das Ablösungs-Kennzeichen auf', () => {
    const { items, changed } = refreshItemsFromDutyTypes([item({ end_at_next_duty: false })], TYPES)
    expect(changed).toBe(1)
    expect(items[0].end_at_next_duty).toBe(true)
  })

  test('ein abweichendes Kennzeichen allein zählt schon als Änderung', () => {
    const before = [item({ duty_type_id: 8, anchor: 'end', offset_minutes: 5, hours_value: 0.5,
      duration_mode: 'absolut', end_anchor: 'end', end_offset_minutes: 0,
      end_at_next_duty: true, audiences: [] })]
    const { items, changed } = refreshItemsFromDutyTypes(before, TYPES)
    expect(changed).toBe(1)
    expect(items[0].end_at_next_duty).toBe(false)
  })
})
