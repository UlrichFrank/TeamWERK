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

Das System SHALL die automatische Zusage nur dann setzen, wenn die Voreinstellung des Spiels für die jeweilige Rolle des Mitglieds `'confirmed'` ist:

- `rsvp_default_players='confirmed'` greift ausschließlich für Mitglieder, die im regulären Kader (`kader_members`) eines der am Spiel beteiligten Teams eingetragen sind.
- `rsvp_default_extended='confirmed'` greift ausschließlich für Mitglieder, die nur über `kader_extended_members` beteiligt sind (nicht bereits im Stammkader).

Analog gilt `'declined'` als virtuelle Absage und `'none'` bedeutet keine Voreinstellung (Mitglied muss selbst antworten).

#### Scenario: Auto-Confirm greift nicht bei extended-only Mitglied, wenn `extended='none'`

- **WHEN** ein Spieler nur über `kader_extended_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_default_players='confirmed'` und `rsvp_default_extended='none'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: null` zurück

#### Scenario: Auto-Confirm greift für reguläres Mitglied bei `players='confirmed'`

- **WHEN** ein Spieler über `kader_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_default_players='confirmed'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück

#### Scenario: Extended-only-Mitglied wird bei `extended='confirmed'` autoconfirmed

- **WHEN** ein Spieler nur über `kader_extended_members` beteiligt ist
- **WHEN** das Spiel hat `rsvp_default_extended='confirmed'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück

### Requirement: Spiel-RSVP-Cutoff 18 Stunden vor Beginn

Das System SHALL `POST /api/games/{id}/respond` für Nutzer ohne Trainer-/Vorstand-/Admin-Berechtigung mit HTTP 422 ablehnen, sobald die aktuelle Zeit weniger als 18 Stunden vor dem Beginn des Spiels (`date` + `time` in Europe/Berlin) liegt. Der Cutoff sperrt jeden Statuswechsel — die erste Antwort, einen Wechsel zwischen `confirmed`/`declined`/`maybe`, und das Aktualisieren des `reason`-Feldes.

Die Fehlerantwort SHALL den Body `{"error":"rsvp_locked","message":"Spiel kann nur bis 18 Stunden vor Beginn umgesagt werden.","locks_at":"<RFC3339 UTC>"}` liefern.

#### Scenario: Spieler antwortet 2 Tage vor Spiel
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"confirmed"}` aufruft und das Spiel beginnt in 48 Stunden
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'confirmed'` für den Spieler gespeichert

#### Scenario: Spieler sagt 12 Stunden vor Spiel ab
- **WHEN** ein Spieler `POST /api/games/{id}/respond` mit `{"status":"declined"}` aufruft und das Spiel beginnt in 12 Stunden
- **THEN** antwortet der Server mit HTTP 422
- **THEN** enthält der Response-Body `error=rsvp_locked` und `locks_at` als RFC3339-UTC

#### Scenario: Spieler ändert Antwort 12 Stunden vor Spiel
- **WHEN** ein Spieler bereits `confirmed` ist und 12 Stunden vor Beginn auf `declined` wechseln will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** bleibt `game_responses.status = 'confirmed'` unverändert

#### Scenario: Spieler beantwortet Spiel erstmals 12 Stunden vor Beginn
- **WHEN** ein Spieler ohne bestehende Response 12 Stunden vor Beginn antworten will
- **THEN** antwortet der Server mit HTTP 422
- **THEN** existiert weiterhin keine Zeile in `game_responses` für diesen Spieler

#### Scenario: Elternteil antwortet 12 Stunden vor Spiel für Kind
- **WHEN** ein Elternteil 12 Stunden vor Beginn `POST` mit `{"member_id": <Kind>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Trainer pflegt Response 12 Stunden vor Spiel
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` 12 Stunden vor Beginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** ist `game_responses.status = 'declined'` gespeichert mit `responded_by = <Trainer-User-ID>`

#### Scenario: Vorstand pflegt Response nach Spielbeginn
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` 1 Stunde nach Spielbeginn `POST` mit `{"member_id": <Spieler>, "status":"declined"}` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Sportliche Leitung darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung` 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Admin darf nach Cutoff antworten
- **WHEN** ein Nutzer mit Systemrolle `admin` (ohne Vereinsfunktion) 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 204

#### Scenario: Kassierer darf nicht nach Cutoff antworten
- **WHEN** ein Nutzer mit ausschließlicher Vereinsfunktion `kassierer` 12 Stunden vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 422

#### Scenario: Absence-Lock hat Vorrang vor Cutoff
- **WHEN** ein Spieler mit gesetztem `game_responses.absence_id` 2 Tage vor Beginn `POST` aufruft
- **THEN** antwortet der Server mit HTTP 403 (Absence-Lock, **nicht** 422)

---

### Requirement: Game-Listing liefert rsvp_locks_at

Listing- und Detail-Endpoints für Spiele SHALL pro Spiel ein Feld `rsvp_locks_at` (RFC3339, UTC) liefern, das den Zeitpunkt benennt, ab dem reguläre Mitglieder keine RSVP-Änderung mehr vornehmen können.

#### Scenario: Eigene Spiele-Liste enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games/my` aufruft
- **THEN** enthält jedes Spiel-Objekt das Feld `rsvp_locks_at` als RFC3339-UTC-String

#### Scenario: Vorstand-Spiele-Liste enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games` aufruft
- **THEN** enthält jedes Spiel-Objekt das Feld `rsvp_locks_at`

#### Scenario: Spiel-Detail enthält rsvp_locks_at
- **WHEN** ein User `GET /api/games/{id}` aufruft
- **THEN** enthält die Response das Feld `rsvp_locks_at`

#### Scenario: rsvp_locks_at = start - 18h
- **WHEN** ein Spiel am 30.06.2026 um 18:00 Uhr Europe/Berlin startet
- **THEN** liefert die API `rsvp_locks_at = "2026-06-29T22:00:00Z"` (00:00 Berliner Sommerzeit am 30.06. = 22:00 UTC am 29.06.)

### Requirement: Trainer können auf Spiel-RSVP antworten

`POST /api/games/{id}/respond` SHALL akzeptieren, dass ein User mit Vereinsfunktion `trainer` für seine eigene `member_id` oder für die `member_id` eines anderen Trainers antwortet. Der `status` MUSS einer von `confirmed | declined | maybe` sein.

Trainer-Rows in `game_responses` werden NICHT in `confirmed_count` / `declined_count` / `maybe_count` gezählt (siehe `trainer-rsvp`-Capability).

#### Scenario: Trainer sagt für sich selbst ab

- **WHEN** ein Trainer `POST /api/games/{id}/respond` mit `{"status":"declined","reason":"Krank"}` aufruft
- **THEN** antwortet der Server mit HTTP 204
- **THEN** existiert eine Row in `game_responses` mit `member_id=<Trainers Member>`, `status='declined'`, `reason='Krank'`

#### Scenario: Trainer sagt für anderen Trainer zu

- **WHEN** Trainer A `POST …/respond` mit `member_id=<TrainerB>` und `status='confirmed'` aufruft
- **THEN** antwortet der Server mit HTTP 204

### Requirement: RSVP-Voreinstellung pro Rolle (Spiele)

Jedes Spiel SHALL für Stammkader-Spieler und den Erweiterten Kader **unabhängig** eine Voreinstellung tragen: `confirmed` („standardmäßig zugesagt"), `declined` („standardmäßig abgesagt") oder `none` („keine automatische Rückmeldung"). Die Spalten heißen `rsvp_default_players` und `rsvp_default_extended` (TEXT NOT NULL DEFAULT `'none'` mit `CHECK` auf die drei Werte). Trainer haben KEINE Voreinstellungs-Spalte und werden weiterhin hart als `confirmed` behandelt.

Die Voreinstellung wird virtuell angewendet: fehlt zu einem Mitglied eine `game_responses`-Row, liefert die API den passenden Default-Status. Es werden dabei KEINE Rows in `game_responses` erzeugt.

#### Scenario: `my_rsvp` reflektiert Default für Stammkader-Spieler
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` hat und ein Spieler ist im Stammkader eines beteiligten Teams ohne `game_responses`-Eintrag
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp='confirmed'`

#### Scenario: `my_rsvp` bleibt null bei `extended='none'` für Extended-only-Mitglied
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` und `rsvp_default_extended='none'` hat
- **AND** ein Mitglied ist nur über `kader_extended_members` beteiligt und hat keine Response
- **THEN** liefert `GET /api/games/my` `my_rsvp=null`

#### Scenario: `my_rsvp` = `'declined'` bei `extended='declined'` für Extended-only-Mitglied
- **WHEN** ein Spiel `rsvp_default_extended='declined'` hat und ein Erweitertes-Kader-Mitglied hat keine Response
- **THEN** liefert `GET /api/games/my` `my_rsvp='declined'`

#### Scenario: Aktive Response überschreibt Default
- **WHEN** dasselbe Spiel `rsvp_default_players='confirmed'` hat und ein Stammkader-Spieler hat `game_responses.status='maybe'`
- **THEN** liefert `GET /api/games/my` `my_rsvp='maybe'`

---

### Requirement: Header-Zähler bezieht Voreinstellungen ein (Spiele)

`GET /api/games/{id}`, `GET /api/games` und `GET /api/games/my` SHALL in `confirmed_count`, `declined_count` und `maybe_count` Mitglieder mit virtuellem Default-Status ihrer Rolle mitzählen — nach der Formel `COALESCE(game_responses.status, game.rsvp_default_<role>)`, wobei `'none'` nirgends mitzählt. Trainer bleiben (unverändert) aus allen drei Zählern ausgeschlossen.

#### Scenario: Zähler bei `players='confirmed'` ohne Responses
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` hat und 3 Stammkader-Spieler ohne Response existieren
- **THEN** enthält der Spiel-Response `confirmed_count=3`, `declined_count=0`

#### Scenario: Zähler bei `extended='declined'` ohne Responses
- **WHEN** ein Spiel `rsvp_default_extended='declined'` hat und 2 Erweiterte-Kader-Mitglieder ohne Response existieren
- **THEN** enthält der Spiel-Response `declined_count=2`

### Requirement: Trainer-Default `confirmed` in `GET /api/games/my`

`GET /api/games/my` SHALL für einen aufrufenden User, der über `kader_trainers` Trainer eines am Spiel beteiligten Teams ist und **keine** eigene `game_responses`-Row hat, `my_rsvp='confirmed'` als virtuellen Default liefern (Priorität: explizite Response > Stammkader-Default > Erweitert-Default > Trainer-`confirmed` > `null`). Ohne Beziehung zum Spiel bleibt `my_rsvp=null`.

#### Scenario: Trainer ohne Response sieht confirmed
- **WHEN** ein User Trainer eines beteiligten Teams eines Spiels ist und keine `game_responses`-Row hat
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp='confirmed'`

#### Scenario: Fremder Funktionsträger sieht keinen Default
- **WHEN** ein Vorstand (kein Trainer/Spieler/Erweiterter eines beteiligten Teams) das Spiel sieht und keine Response hat
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp=null`

