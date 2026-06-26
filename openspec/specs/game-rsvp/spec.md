# game-rsvp Specification

## Purpose

Diese Spezifikation beschreibt die Capability `game-rsvp`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Spieler kann zu einem Spiel RSVP abgeben

Das System SHALL es Spielern und Eltern ermöglichen, zu jedem Spiel ihres Teams
eine Rückmeldung (confirmed / declined / maybe) mit optionalem Grund abzugeben.

#### Scenario: Spieler sagt zu
- **WHEN** ein User mit Rolle `spieler` `POST /api/games/{id}/respond` mit `{"status": "confirmed"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist der Eintrag in `game_responses` mit `status = 'confirmed'` gespeichert

#### Scenario: Spieler sagt ab mit Grund
- **WHEN** ein User `POST /api/games/{id}/respond` mit `{"status": "declined", "reason": "Urlaub"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist der Grund in `game_responses.reason` gespeichert

#### Scenario: Elternteil sagt für Kind ab
- **WHEN** ein User mit Rolle `elternteil` `POST /api/games/{id}/respond` mit `{"member_id": 42, "status": "declined"}` aufruft
- **AND** member_id 42 ist via `family_links` mit dem User verknüpft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Elternteil ohne Verknüpfung wird abgelehnt
- **WHEN** ein User mit Rolle `elternteil` `POST /api/games/{id}/respond` mit einer `member_id` aufruft, die nicht zu seinen Kindern gehört
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: RSVP-Update überschreibt alten Eintrag
- **WHEN** ein User für dasselbe Spiel erneut `POST /api/games/{id}/respond` aufruft
- **THEN** wird der bestehende Eintrag via UPSERT aktualisiert

#### Scenario: Ungültiger Status wird abgelehnt
- **WHEN** `POST /api/games/{id}/respond` mit einem Status außerhalb von `confirmed/declined/maybe` aufgerufen wird
- **THEN** antwortet der Server mit HTTP 400

---

### Requirement: Kein Default-Status bei Spielen

Das System SHALL für Spiele keinen vorausgewählten RSVP-Status setzen.
Spieler müssen aktiv eine Auswahl treffen.

#### Scenario: Neues Spiel hat keinen RSVP-Status
- **WHEN** ein User die `/termine`-Seite aufruft und ein Spiel noch keine Rückmeldung hat
- **THEN** sind alle drei RSVP-Buttons (Zusagen/Vielleicht/Absagen) inaktiv (kein Button ist hervorgehoben)

---

### Requirement: User-gefilterte Spielliste mit RSVP-Daten

`GET /api/games/my` SHALL Spiele des eigenen Teams zurückgeben, inklusive
`my_rsvp`, `confirmed_count`, `declined_count`, `maybe_count` pro Spiel.

#### Scenario: Spieler sieht nur eigene Teamspiele
- **WHEN** ein User mit Rolle `spieler` `GET /api/games/my` aufruft
- **THEN** enthält die Antwort nur Spiele, bei denen sein Team über `game_teams` beteiligt ist

#### Scenario: RSVP-Counts in der Liste
- **WHEN** `GET /api/games/my` aufgerufen wird
- **THEN** enthält jedes Spiel-Objekt die Felder `confirmed_count`, `declined_count`, `maybe_count` und `my_rsvp`

#### Scenario: my_rsvp ist null wenn keine Antwort
- **WHEN** ein User noch keine Rückmeldung für ein Spiel abgegeben hat
- **THEN** ist `my_rsvp` im Response `null`

---

### Requirement: Trainer sieht Rückmeldungs-Übersicht pro Spiel

`GET /api/games/{id}/responses` SHALL für Trainer und Admins alle Rückmeldungen
zu einem Spiel zurückgeben (member_name, status, reason).

#### Scenario: Trainer ruft Übersicht ab
- **WHEN** ein User mit Rolle `trainer` oder `admin` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 200 und einer Liste aller Rückmeldungen

#### Scenario: Spieler kann Übersicht nicht abrufen
- **WHEN** ein User mit Rolle `spieler` `GET /api/games/{id}/responses` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Grund nur für eigene oder eigene Kinder sichtbar
- **WHEN** ein User mit Rolle `elternteil` die Detailseite aufruft
- **THEN** sind Gründe nur für seine eigenen Kinder sichtbar, nicht für andere Mitglieder

---

### Requirement: Spiel-Response manuell ändern
Ein Spieler oder berechtigter Elternteil SHALL eine Spiel-Response (confirmed/declined/maybe) nur dann manuell ändern können, wenn die Response kein gesetztes `absence_id` hat. Ist `absence_id IS NOT NULL`, MUST die API die Änderung mit HTTP 403 ablehnen. Der Member MUST stattdessen die zugehörige Abwesenheit löschen.

#### Scenario: Manuelle Änderung ohne Abwesenheit
- **WHEN** ein Nutzer eine Spiel-Response ändert und `absence_id IS NULL`
- **THEN** wird die Änderung akzeptiert

#### Scenario: Manuelle Änderung bei auto-declined Response abgewiesen
- **WHEN** ein Nutzer versucht, eine Spiel-Response mit `absence_id IS NOT NULL` zu ändern
- **THEN** antwortet die API mit HTTP 403

#### Scenario: Trainer kann auto-declined nicht überschreiben
- **WHEN** ein Trainer versucht, eine Response mit `absence_id IS NOT NULL` für ein Kader-Member zu ändern
- **THEN** antwortet die API mit HTTP 403

---

### Requirement: Auto-Confirm gilt nur für reguläre Kader-Mitglieder

Das System SHALL bei opt-out-Spielen (`rsvp_opt_out = 1`) die automatische Zusage (`my_rsvp: "confirmed"`) nur für Mitglieder setzen, die im regulären Kader (`kader_members`) eines der am Spiel beteiligten Teams eingetragen sind. Mitglieder, die ausschließlich über `kader_extended_members` an einem Spiel beteiligt sind, erhalten keinen Auto-Confirm und müssen explizit zusagen.

#### Scenario: Opt-out greift nicht bei extended-only Mitglied

- **WHEN** ein Spieler nur über `kader_extended_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: null` zurück

#### Scenario: Opt-out greift weiterhin für reguläres Mitglied

- **WHEN** ein Spieler über `kader_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück
