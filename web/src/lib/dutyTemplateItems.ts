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
