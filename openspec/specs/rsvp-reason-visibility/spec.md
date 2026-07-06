# rsvp-reason-visibility Specification

## Purpose

Regelt die Sichtbarkeit des freien Textes, den ein Nutzer beim Absagen oder „Vielleicht"-Antworten eines Termins (Spiel oder Training) über das Modal aus `rsvp-reason-modal` eingegeben hat. Das Modal fordert die Eingabe, aber ohne diese Capability wäre die Eingabe „write-only" und für den eingebenden Nutzer selbst nicht mehr nachprüfbar. Ziel: konsistente Regel „Trainer alles, Mitglied nur eigenes, Elternteil zusätzlich Kind" über Listen- und Detail-Views hinweg, für Spiele und Trainings gleich.

## Requirements

### Requirement: My-Reason auf Termin-Liste

Das System SHALL im JSON-Response von `GET /api/games/my` und `GET /api/training-sessions/my` pro Termin ein Feld `my_reason` (String) liefern, **wenn** der anfragende Nutzer für diesen Termin explizit mit `declined` oder `maybe` geantwortet hat **und** dabei einen nicht-leeren Grund angegeben hat. Wenn der Nutzer nur eine implizite Default-RSVP hat (`my_rsvp_is_default=true`) oder keine Antwort abgegeben hat, MUST das Feld weggelassen werden (`omitempty`).

#### Scenario: RSVP mit Grund → Feld gefüllt

- **GIVEN** ein Nutzer hat auf `/api/games/{id}/respond` mit `{"status":"declined","reason":"Arbeit bis 20h"}` geantwortet
- **WHEN** er `GET /api/games/my` aufruft
- **THEN** enthält der Response für dieses Spiel `"my_reason": "Arbeit bis 20h"` und `"my_rsvp": "declined"`

#### Scenario: Default-RSVP ohne Antwort → Feld fehlt

- **GIVEN** ein Termin mit `rsvp_default_players="confirmed"` und der Nutzer hat nichts geantwortet
- **WHEN** er `GET /api/training-sessions/my` aufruft
- **THEN** enthält der Response `"my_rsvp": "confirmed"`, `"my_rsvp_is_default": true`, aber **kein** Feld `my_reason`

### Requirement: Kind-Reason auf Termin-Liste für Eltern

Das System SHALL im JSON-Response von `GET /api/games/my` und `GET /api/training-sessions/my` innerhalb des `children_rsvp`-Arrays pro Kind ein Feld `reason` (String) liefern, wenn das Kind explizit geantwortet hat und dabei einen nicht-leeren Grund angegeben hat. Fehlt eine explizite Antwort oder ein Grund, MUST das Feld weggelassen werden.

#### Scenario: Elternteil sieht Kind-Reason

- **GIVEN** ein Elternteil (`family_links`-Eintrag) und das verknüpfte Kind hat mit Reason „Krank" ein Training abgesagt
- **WHEN** der Elternteil `GET /api/training-sessions/my` aufruft
- **THEN** enthält `children_rsvp` einen Eintrag für dieses Kind mit `"rsvp": "declined"` und `"reason": "Krank"`

#### Scenario: Nicht-Elternteil ist unberührt

- **GIVEN** ein Nutzer ohne `family_links`-Einträge
- **WHEN** er `GET /api/games/my` aufruft
- **THEN** ist `children_rsvp` ein leeres Array oder das Feld fehlt (unverändertes Verhalten)

### Requirement: Attendance-Reason-Gate: Trainer sieht alle

Das System SHALL in `GET /api/games/{id}/attendances` und `GET /api/training-sessions/{id}/attendances` für Nutzer mit System-Rolle `admin` oder Vereinsfunktion `trainer` das Feld `reason` pro Zeile befüllen, wenn ein nicht-leerer Grund in `game_responses.reason` / `training_responses.reason` steht.

#### Scenario: Trainer sieht alle Reasons

- **GIVEN** ein Trainer-User, mehrere Kader-Mitglieder haben mit unterschiedlichen Reasons abgesagt
- **WHEN** er `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** enthält jede Zeile mit Reason-Text ein befülltes `reason`-Feld

### Requirement: Attendance-Reason-Gate: Mitglied sieht nur eigenes

Das System SHALL in denselben Endpoints das Feld `reason` **nur** für die eigene Zeile des anfragenden Nutzers befüllen (matching über `member_id`), wenn dieser weder Trainer noch Elternteil des Zeilen-Mitglieds ist. Für alle anderen Zeilen MUST `reason` `null` sein.

#### Scenario: Mitglied sieht eigenen Reason

- **GIVEN** ein regulärer Kader-Spieler ohne Trainer-Funktion, hat mit Reason „Verletzt" abgesagt
- **WHEN** er `GET /api/games/{id}/attendances` aufruft
- **THEN** enthält die eigene Zeile `"reason": "Verletzt"` und alle fremden Zeilen `"reason": null`

### Requirement: Attendance-Reason-Gate: Elternteil sieht Kind-Zeilen

Das System SHALL in denselben Endpoints das Feld `reason` zusätzlich für jede Zeile befüllen, deren `member_id` mit einem `family_links.member_id`-Eintrag des anfragenden Nutzers (`parent_user_id`) matcht.

#### Scenario: Elternteil sieht Kind-Reason

- **GIVEN** ein Elternteil (`family_links` verknüpft ihn mit member_id 42), Kind 42 hat einen Trainingstermin mit Reason „Klavierstunde" abgesagt
- **WHEN** der Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** enthält die Zeile mit `member_id=42` `"reason": "Klavierstunde"`, alle anderen Zeilen (außer eigener) `"reason": null`

### Requirement: Attendance-Reason-Gate: Fremd-Reason bleibt geschlossen

Das System MUST in denselben Endpoints für alle Zeilen, auf die weder Trainer-Rolle noch Eigen-Match noch Eltern-Kind-Match zutrifft, `reason` als `null` liefern — auch dann, wenn `game_responses.reason` / `training_responses.reason` einen nicht-leeren Wert enthält.

#### Scenario: Regressionstest gegen heutigen Leak (Games)

- **GIVEN** ein Nutzer mit Team-Access, aber keine Trainer-Funktion, kein Kind-Eintrag, kein Eigen-Match; die Attendance-Liste enthält eine Zeile eines anderen Mitglieds mit Reason „Grippe"
- **WHEN** der Nutzer `GET /api/games/{id}/attendances` aufruft
- **THEN** enthält die betreffende Zeile `"reason": null` (nicht „Grippe")

### Requirement: Frontend rendert Reason nur, wenn im Payload

Das Frontend SHALL in `TerminePage` unter dem RSVP-Buttonblock einer Termin-Karte den Text aus `my_reason` anzeigen, wenn das Feld im Payload gesetzt ist. Fehlt das Feld, MUST kein Element gerendert werden.

#### Scenario: Karte zeigt Reason wenn Feld gesetzt

- **GIVEN** eine Termin-Karte, deren Payload `"my_rsvp":"declined","my_reason":"Arbeit"` enthält
- **WHEN** der Nutzer `/termine` öffnet
- **THEN** wird unterhalb der RSVP-Buttons eine kleine Zeile mit MessageCircle-Icon und dem Text „Arbeit" gerendert

#### Scenario: Karte bleibt sauber wenn Feld fehlt

- **GIVEN** eine Termin-Karte, deren Payload `"my_rsvp":"confirmed"` (kein `my_reason`) enthält
- **WHEN** der Nutzer `/termine` öffnet
- **THEN** wird kein zusätzliches Element unterhalb der Buttons gerendert

### Requirement: Frontend rendert Kind-Reason analog

Das Frontend SHALL in der Kind-Zeile innerhalb einer Termin-Karte den Text aus `children_rsvp[i].reason` anzeigen, wenn das Feld gesetzt ist. Fehlt das Feld, MUST kein Element gerendert werden.

#### Scenario: Kind-Zeile zeigt Reason

- **GIVEN** eine Termin-Karte für einen Elternteil, `children_rsvp[0]` = `{"name":"Anna","rsvp":"declined","reason":"Krank"}`
- **WHEN** der Elternteil `/termine` öffnet
- **THEN** wird unterhalb der Kind-Buttons für „Anna" der Text „Krank" mit MessageCircle-Icon gerendert
