## Context

Das `EventInfoModal` ist eine Props-only-Komponente (kein eigener API-Call). `KalenderPage.tsx` übergibt beim Klick auf einen Kalendereintrag das jeweilige `Game`- oder `Training`-Objekt. Die benötigten Daten (`end_date`, `teams`) sind im `Game`-Objekt von `KalenderPage` bereits vorhanden — sie werden bisher nur nicht an `EventInfoModal` weitergereicht, weil das Interface der Komponente diese Felder nicht deklariert.

Kurznamen werden in `KalenderPage` via `buildTeamShortNames(teams)` vorberechnet und in der `shortNames`-Map gespeichert.

## Goals / Non-Goals

**Goals:**
- `EventInfoModal` zeigt für `event_type === 'generisch'` bei vorhandenem `end_date` eine Datumspanne
- Label "Gegner" wird bei generischen Events zu "Event-Name"
- Alle Detailansichten (Heim, Auswärts, Generisch, Training) zeigen die betroffene(n) Mannschaft(en) als Kurznamen

**Non-Goals:**
- Kein Backend-Eingriff
- Keine Änderung an der Abwesenheits-Detailansicht
- Kein Umbau des Modal-Layouts oder der RSVP-Logik

## Decisions

**Erweiterung der EventInfoModal-Interfaces statt neuer Props:**  
`Game` bekommt `end_date?: string | null` und `teams?: Array<{ id: number; name: string }>` (mit vorberechneten Kurznamen statt roher Team-Objekte).  
`Training` bekommt `team_name?: string`.  
Alternative wäre eine generische `teamLabels?: string[]`-Prop — abgelehnt, weil die typisierten Interface-Erweiterungen klarer ausdrücken, was woher kommt.

**Kurznamen werden in KalenderPage vorberechnet:**  
`KalenderPage` bildet bereits `shortNames: Map<number, string>`. Beim Öffnen des Modals werden die Teams mit ihren Kurznamen gemappt (`game.teams.map(t => ({ id: t.id, name: shortNames.get(t.id) ?? t.name }))`). Damit bleibt `EventInfoModal` props-only und kennt keine shortName-Logik.

## Risks / Trade-offs

- `end_date` aus dem API-Response kommt als ISO-Timestamp (`"2026-09-10T00:00:00Z"`). Daher wird `.slice(0, 10)` verwendet — konsistent mit der bestehenden Praxis in `KalenderPage`.
- Wenn ein Game keine `teams` hat (leeres Array), entfällt die Teams-Zeile im Modal stillschweigend.
