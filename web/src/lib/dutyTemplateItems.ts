/**
 * Toggle-Helfer für `team_ids` eines Dienstplan-Vorlagen-Eintrags.
 *
 * Baut das Array nie neu aus den sichtbaren Optionen auf, sondern ergänzt bzw.
 * entfernt nur die eine ID — ein gespeichertes Team, das in der aktiven Saison
 * keinen Kader mehr hat, taucht als Checkbox gar nicht auf und würde sonst
 * beim nächsten Speichern stillschweigend verschwinden.
 */
export function toggleTeamID(
  current: number[] | null | undefined,
  teamID: number,
  checked: boolean,
): number[] {
  const ids = current ?? []
  if (checked) return ids.includes(teamID) ? ids : [...ids, teamID]
  return ids.filter(x => x !== teamID)
}

/** Die Felder, die eine Vorlagen-Zeile beim Auswählen des Diensttyps aus ihm kopiert. */
export interface DutyTypeDefaults {
  id: number
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  hours_value: number
  /** Dauer-Modus des Diensttyps (dienst-dauer-dynamisch); optional wie audiences —
   * defensiv gegen eine gecachte Alt-Antwort ohne die drei Felder. */
  duration_mode?: 'absolut' | 'dynamisch'
  end_anchor?: 'start' | 'end'
  end_offset_minutes?: number
  /** Ablösungs-Kennzeichen (dienst-abloesung); optional aus demselben Grund. */
  end_at_next_duty?: boolean
  audiences?: string[] | null
}

interface RefreshableItem {
  duty_type_id: number
  anchor: 'start' | 'end'
  offset_minutes: number
  hours_value: number
  duration_mode: 'absolut' | 'dynamisch'
  end_anchor: 'start' | 'end'
  end_offset_minutes: number
  end_at_next_duty: boolean
  audiences: string[]
}

/**
 * Holt für jede Vorlagen-Zeile die aktuellen Werte ihres Diensttyps zurück in
 * die Zeile — dieselben Felder, die beim Auswählen des Diensttyps ohnehin
 * hineinkopiert werden (Copy-on-pick).
 *
 * Nötig, weil die Zeile nach dem Auswählen eigenständig ist: eine spätere
 * Änderung am Diensttyp erreicht sie nicht, und der Dienst-Regen liest
 * ausschließlich die Zeile. Ohne diesen Weg müsste der Vorstand jede Vorlage
 * von Hand nachziehen, um eine geänderte Dauer wirksam zu machen.
 *
 * Gibt die neuen Zeilen und die Anzahl tatsächlich geänderter zurück; Zeilen
 * ohne passenden Diensttyp (noch nicht ausgewählt, Typ gelöscht) bleiben
 * unangetastet. Reine Formular-Operation — persistiert wird erst beim Speichern.
 */
export function refreshItemsFromDutyTypes<T extends RefreshableItem>(
  items: T[],
  dutyTypes: DutyTypeDefaults[],
): { items: T[]; changed: number } {
  let changed = 0
  const next = items.map(item => {
    const dt = dutyTypes.find(d => d.id === item.duty_type_id)
    if (!dt) return item
    const audiences = dt.audiences ?? []
    const durationMode = dt.duration_mode ?? 'absolut'
    const endAnchor = dt.end_anchor ?? 'end'
    const endOffsetMinutes = dt.end_offset_minutes ?? 0
    const endAtNextDuty = dt.end_at_next_duty ?? false
    const isSame =
      item.anchor === dt.default_anchor &&
      item.offset_minutes === dt.default_offset_minutes &&
      item.hours_value === dt.hours_value &&
      item.duration_mode === durationMode &&
      item.end_anchor === endAnchor &&
      item.end_offset_minutes === endOffsetMinutes &&
      item.end_at_next_duty === endAtNextDuty &&
      item.audiences.length === audiences.length &&
      item.audiences.every(a => audiences.includes(a))
    if (isSame) return item
    changed++
    return {
      ...item,
      anchor: dt.default_anchor,
      offset_minutes: dt.default_offset_minutes,
      hours_value: dt.hours_value,
      duration_mode: durationMode,
      end_anchor: endAnchor,
      end_offset_minutes: endOffsetMinutes,
      end_at_next_duty: endAtNextDuty,
      audiences,
    }
  })
  return { items: next, changed }
}
