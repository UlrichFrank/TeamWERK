# dashboard-team-filter Specification

## Purpose

Diese Spezifikation beschreibt die Capability `dashboard-team-filter`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Team-Sichtbarkeit auf dem Dashboard

**Kontext:** Das Dashboard zeigt team-spezifische Daten nur für Nutzer mit Teamzugang. Die Logik zur Ermittlung der zugänglichen Teams wird vereinheitlicht.

**Bisheriges Verhalten:**
- `spieler`: Teams via `kader_members` + `members.user_id`
- `elternteil`: Teams via `family_links` + `kader_members`
- `trainer`: Teams via `kader_trainers` + `members.user_id` (**kein family_links-Pfad**)
- `admin`/`vorstand`: kein team-spezifischer Abschnitt

**Neues Verhalten:**
- Alle Rollen außer `admin`/`vorstand` verwenden `user_accessible_teams WHERE user_id = ? AND season_id = ?`
- `user_accessible_teams` deckt alle Pfade ab: `kader_members` (spieler), `family_links + kader_members` (elternteil), `kader_trainers` (trainer)
- Ein Trainer mit Kindern im Kader sieht die Spiele und Fahrtgemeinschaften dieser Teams
- `admin`/`vorstand` sehen weiterhin keine team-spezifischen Sections (Early-Return)

**Betroffene Dashboard-Sections:**
- Nächste Spiele (`queryNextGames`)
- Fahrtgemeinschaft-Hint (`queryCarpoolingHint`)
- Fahrzeug-Action (`vehicleAction`)
