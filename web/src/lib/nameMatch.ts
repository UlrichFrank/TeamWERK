interface NamedEntity {
  first_name: string
  last_name: string
}

function normalize(name: string): string {
  return name.trim().toLowerCase()
}

// Heuristik für "wahrscheinlicher Treffer": gleicher Nachname (typischerweise Familie)
// oder gleicher Vorname (typischerweise der eigene Account) — exakter Vergleich reicht,
// Fuzzy-Matching wäre für die kleine, manuell kuratierte Nutzerliste Overkill.
export function isLikelyNameMatch(candidate: NamedEntity, reference: NamedEntity): boolean {
  const cf = normalize(candidate.first_name)
  const cl = normalize(candidate.last_name)
  const rf = normalize(reference.first_name)
  const rl = normalize(reference.last_name)
  return (cl !== '' && cl === rl) || (cf !== '' && cf === rf)
}

// Treffer auf Vor- UND Nachnamen ist vermutlich das Mitglied selbst (bereits per
// "Nutzer verknüpfen" verbunden) — kein Kandidat für Erziehungsberechtigte.
export function isFullNameMatch(candidate: NamedEntity, reference: NamedEntity): boolean {
  const cf = normalize(candidate.first_name)
  const cl = normalize(candidate.last_name)
  const rf = normalize(reference.first_name)
  const rl = normalize(reference.last_name)
  return cf !== '' && cl !== '' && cf === rf && cl === rl
}

// Für Erziehungsberechtigte zählt nur der Nachname (Familienname) als Kriterium.
export function isLastNameMatch(candidate: NamedEntity, reference: NamedEntity): boolean {
  const cl = normalize(candidate.last_name)
  const rl = normalize(reference.last_name)
  return cl !== '' && cl === rl
}
