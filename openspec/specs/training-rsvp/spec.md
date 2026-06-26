# training-rsvp Specification

## Purpose

Diese Spezifikation beschreibt die Capability `training-rsvp`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Training-RSVP-Route

Die Training-RSVP-Funktionalität SHALL über `/termine` erreichbar sein.
Die RSVP-API-Endpunkte (`/api/training-sessions/{id}/respond`) bleiben unverändert.

#### Scenario: Spieler gibt RSVP über /termine ab
- **WHEN** ein User mit Rolle `spieler` die `/termine`-Seite aufruft
- **THEN** werden Trainings des eigenen Teams mit RSVP-Buttons angezeigt
- **THEN** führt ein RSVP-Klick intern zu `POST /api/training-sessions/{id}/respond`

#### Scenario: Trainer-Detailseite für Training über /termine
- **WHEN** ein Trainer auf eine Trainingskarte klickt
- **THEN** wird er zu `/termine/training/:id` navigiert
- **THEN** zeigt die Seite die RSVP-Tabelle + Anwesenheit

---

### Requirement: Spieler können für sich selbst zu-/absagen

Ein User mit `role='spieler'` SHALL für sich selbst eine RSVP-Antwort auf eine Trainingssession abgeben können (confirmed/declined/maybe). Eine Antwort kann jederzeit geändert werden (Upsert).

#### Scenario: Spieler sagt zu
- **WHEN** ein Spieler POST `/api/training-sessions/{id}/respond` mit `status='confirmed'` aufruft
- **THEN** wird eine `training_responses`-Row mit `member_id=<Spielers Mitglied>` und `responded_by=<user_id>` angelegt oder aktualisiert

#### Scenario: Spieler ändert Antwort
- **WHEN** ein Spieler dieselbe Session erneut mit `status='declined'` und `reason='Krank'` beantwortet
- **THEN** wird die bestehende Response aktualisiert (Upsert auf UNIQUE(training_id, member_id))

#### Scenario: Spieler ohne Mitglied-Verknüpfung kann nicht antworten
- **WHEN** ein User mit `role='spieler'` antwortet, dessen Account keinem `member`-Eintrag zugeordnet ist (`members.user_id` fehlt)
- **THEN** antwortet das System mit HTTP 422 und einem erklärenden Fehler

---

### Requirement: Elternteile können für ihre Kinder antworten

Ein User mit `role='elternteil'` SHALL über die `family_links`-Beziehung für seine Kinder eine RSVP-Antwort abgeben können.

#### Scenario: Elternteil sagt für Kind ab
- **WHEN** ein Elternteil POST `/api/training-sessions/{id}/respond` mit `member_id=<Kind>` und `status='declined'` aufruft
- **THEN** wird eine `training_responses`-Row für das Kind angelegt, `responded_by` ist die User-ID des Elternteils

#### Scenario: Elternteil kann nicht für fremde Kinder antworten
- **WHEN** ein Elternteil eine `member_id` angibt, die nicht in `family_links` mit ihrem Account verknüpft ist
- **THEN** antwortet das System mit HTTP 403

---

### Requirement: Privacy — Absage-Begründungen sind eingeschränkt sichtbar

Das System SHALL sicherstellen, dass das `reason`-Feld einer Absage nur für berechtigte Personen zurückgegeben wird.

#### Scenario: Fremder Spieler sieht keine Begründung
- **WHEN** Spieler A GET `/api/training-sessions/{id}` aufruft und Spieler B eine Absage mit Begründung hat
- **THEN** enthält die Response für Spieler B zwar `status='declined'`, aber `reason=null`

#### Scenario: Trainer sieht alle Begründungen
- **WHEN** ein Trainer GET `/api/training-sessions/{id}` aufruft
- **THEN** enthält jede Response das `reason`-Feld ungefiltert

#### Scenario: Spieler sieht eigene Begründung
- **WHEN** ein Spieler GET `/api/training-sessions/{id}` aufruft
- **THEN** ist `reason` in seiner eigenen Response sichtbar

#### Scenario: Elternteil sieht Begründung seines Kindes
- **WHEN** ein Elternteil GET `/api/training-sessions/{id}` aufruft
- **THEN** ist `reason` in der Response seines Kindes sichtbar (via `family_links`-Prüfung)

---

### Requirement: Öffentliche Response-Zusammenfassung

Alle authentifizierten User mit Zugriff auf eine Session SHALL eine Zusammenfassung der Antworten sehen können (Anzahl confirmed/declined/maybe sowie Namen + Status pro Teilnehmer).

#### Scenario: Response-Summary in Session-Liste
- **WHEN** ein Spieler GET `/api/training-sessions` aufruft
- **THEN** enthält jede Session `confirmed_count`, `declined_count`, `pending_count` sowie den eigenen RSVP-Status

#### Scenario: Vollständige Response-Liste in Session-Detail
- **WHEN** ein User GET `/api/training-sessions/{id}` aufruft
- **THEN** enthält die Antwort eine Liste aller Responses mit Name und Status (ohne Begründung für fremde Spieler)

---

### Requirement: Training-Response manuell ändern
Ein Spieler oder berechtigter Elternteil SHALL eine Training-Response (confirmed/declined/maybe) nur dann manuell ändern können, wenn die Response kein gesetztes `absence_id` hat. Ist `absence_id IS NOT NULL`, MUST die API die Änderung mit HTTP 403 ablehnen. Der Member MUST stattdessen die zugehörige Abwesenheit löschen.

#### Scenario: Manuelle Änderung ohne Abwesenheit
- **WHEN** ein Nutzer eine Training-Response ändert und `absence_id IS NULL`
- **THEN** wird die Änderung akzeptiert

#### Scenario: Manuelle Änderung bei auto-declined Response abgewiesen
- **WHEN** ein Nutzer versucht, eine Training-Response mit `absence_id IS NOT NULL` zu ändern
- **THEN** antwortet die API mit HTTP 403

#### Scenario: Trainer kann auto-declined nicht überschreiben
- **WHEN** ein Trainer versucht, eine Response mit `absence_id IS NOT NULL` für ein Kader-Member zu ändern
- **THEN** antwortet die API mit HTTP 403

