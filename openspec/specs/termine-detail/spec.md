# termine-detail Specification

## Purpose
TBD - created by archiving change eltern-rsvp-kinder-sichtbar. Update Purpose after archive.
## Requirements
### Requirement: Training-Detail zeigt vollständige Kaderliste für alle authentifizierten User

`GET /api/training-sessions/{id}/attendances` SHALL für alle authentifizierten User zugänglich sein, die entweder selbst Kader-Mitglied des Teams sind oder ein Kind im Kader haben. Bisher war dieser Endpoint Trainer-only.

#### Scenario: Spieler ruft Training-Detail ab

- **WHEN** ein User mit Rolle `spieler`, der Kader-Mitglied des betreffenden Teams ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder mit deren RSVP-Status

#### Scenario: Elternteil ruft Training-Detail ab

- **WHEN** ein Elternteil, dessen Kind Kader-Mitglied ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 200
- **THEN** enthält die Antwort alle Kader-Mitglieder mit deren RSVP-Status

#### Scenario: Fremder User wird abgelehnt

- **WHEN** ein User, der weder Kader-Mitglied noch Elternteil eines Kader-Mitglieds ist, `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** antwortet der Server mit HTTP 403

---

### Requirement: `present`-Feld nur für Trainer sichtbar

Das Feld `present` in der Attendances-Response SHALL nur für Trainer und Admins einen Wert enthalten. Für Spieler und Eltern ist `present` immer `null`. Für Trainer-Zeilen (`is_trainer=true`) ist `present` unabhängig vom Aufrufer immer `null`, da Trainer keine Anwesenheits-Erfassung haben.

#### Scenario: Nicht-Trainer erhält kein present-Flag

- **WHEN** ein Spieler oder Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `present` für alle Einträge `null`

#### Scenario: Trainer erhält present-Flags für Spieler-Zeilen

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** enthält `present` in Spieler-/Erweiterter-Kader-Zeilen den tatsächlich gespeicherten Wert (true/false/null)

#### Scenario: Trainer-Zeile hat immer present=null

- **WHEN** ein Trainer/Admin die Response abruft und darin eine Zeile mit `is_trainer=true` enthält
- **THEN** ist `present=null` in dieser Zeile

### Requirement: Kommentar-Sichtbarkeit auf der Training-Detail-Seite

`GET /api/training-sessions/{id}/attendances` SHALL ein Feld `reason` zurückgeben, gefiltert nach Rolle:
- Trainer/Admin: alle Kommentare aller Spieler
- Spieler: nur der eigene Kommentar
- Elternteil: nur Kommentare der eigenen Kinder

#### Scenario: Trainer sieht alle Kommentare

- **WHEN** ein Trainer `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` für alle Einträge mit vorhandenem Kommentar befüllt

#### Scenario: Spieler sieht nur eigenen Kommentar

- **WHEN** ein Spieler `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` nur für den eigenen Eintrag befüllt; alle anderen haben `reason: null`

#### Scenario: Elternteil sieht nur Kinder-Kommentare

- **WHEN** ein Elternteil `GET /api/training-sessions/{id}/attendances` aufruft
- **THEN** ist `reason` nur für Einträge der eigenen Kinder befüllt; alle anderen haben `reason: null`

### Requirement: Termin-Detail-Tabelle zeigt drei benannte Sektionen in fester Reihenfolge

Die Termin-Detail-Tabelle (`TermineDetailPage.tsx`) SHALL die Teilnehmer in drei benannten Sektionen anzeigen (nicht-leere zuerst): **Trainer**, **Spieler**, **Erweiterter Kader**. Die Trainer-Sektion MUSS oberhalb der Spieler-Sektion stehen, die Spieler-Sektion oberhalb des Erweiterten Kaders. Jede Sektion wird durch dieselbe `border-t-2 border-brand-border`-Linie visuell abgetrennt. Leere Sektionen (0 Rows) werden weggelassen.

Diese Regel gilt NICHT im Multi-Team-Modus generischer Events (`event_type='generisch'` mit ≥2 Teams) — dort bleibt die Team-Gruppierung erhalten.

#### Scenario: Alle drei Sektionen sichtbar

- **WHEN** ein User einen Termin mit ≥1 Trainer, ≥1 Spieler und ≥1 erweitertem Kader-Mitglied öffnet
- **THEN** zeigt die Tabelle drei benannte Sektionen in der Reihenfolge Trainer / Spieler / Erweiterter Kader

#### Scenario: Kein Trainer im Kader

- **WHEN** der Kader keine Trainer hat
- **THEN** wird die Trainer-Sektion samt Kopfzeile weggelassen, Spieler-Sektion erscheint zuoberst mit sichtbarem Titel „Spieler"

#### Scenario: Kein erweiterter Kader

- **WHEN** kein Mitglied im erweiterten Kader steht
- **THEN** wird die Sektion „Erweiterter Kader" samt Kopfzeile weggelassen

---

### Requirement: Anwesenheit- und Aufstellung-Zelle bleiben für Trainer-Zeilen leer

`ParticipantRow` in der Termin-Detail-Tabelle SHALL für Zeilen mit `is_trainer=true` weder eine Checkbox noch einen Platzhalter (Strich) in der Anwesend- und Aufstellung-Spalte rendern. Die `<td>`-Zellen bleiben strukturell erhalten (leerer Inhalt), damit die Spaltenausrichtung mit Spieler-Zeilen erhalten bleibt.

#### Scenario: Trainer-Zeile hat leere Anwesend-Zelle

- **WHEN** ein Trainer in der Tabelle erscheint und die Anwesend-Spalte für den User sichtbar ist (Trainer/Admin, Termin vergangen)
- **THEN** wird in der Anwesend-Spalte der Trainer-Zeile keine Checkbox und kein Text gerendert

#### Scenario: Trainer-Zeile hat leere Aufstellung-Zelle

- **WHEN** ein Spiel-Termin die Aufstellung-Spalte zeigt
- **THEN** wird in der Aufstellung-Spalte der Trainer-Zeile keine Checkbox und kein Text gerendert

#### Scenario: Spalten bleiben vertikal ausgerichtet

- **WHEN** Trainer- und Spieler-Zeilen in derselben Tabelle stehen
- **THEN** liegt die Rückmeldung-Spalte in Trainer- und Spieler-Zeilen exakt untereinander (kein `colspan`-Shift)

### Requirement: Voreinstellungs-basierte Antworten werden dezent gerendert

Die Termin-Detail-Tabelle (`web/src/pages/TermineDetailPage.tsx`) SHALL Zeilen, deren `rsvp_status` aus der Session/Spiel-Voreinstellung stammt (`rsvp_is_default=true`), visuell von aktiven Antworten unterscheiden: Die Status-Anzeige (Icon + Text) wird mit `text-brand-text-subtle italic` gerendert. Aktive Antworten bleiben in `text-brand-text` (nicht kursiv).

Trainer-Zeilen behalten die bestehende Darstellung ihres virtuellen `confirmed`-Defaults (dieser Change ändert die Trainer-Rendering-Logik nicht).

#### Scenario: Stammkader-Spieler mit Default „standardmäßig zugesagt"
- **WHEN** eine Session `rsvp_default_players='confirmed'` hat und ein Spieler hat keine `training_responses`-Row
- **THEN** zeigt die Detail-Tabellenzeile die Statuszelle mit CSS-Klasse `italic` und Textfarbe `text-brand-text-subtle`

#### Scenario: Aktive Antwort wird nicht kursiv gerendert
- **WHEN** derselbe Spieler eine `training_responses`-Row mit `status='confirmed'` hat
- **THEN** wird die Statuszelle in `text-brand-text` (nicht kursiv) gerendert

#### Scenario: Erweiterter Kader mit Default „standardmäßig abgesagt"
- **WHEN** eine Session `rsvp_default_extended='declined'` hat und ein Erweiterte-Kader-Mitglied hat keine Response
- **THEN** wird die Statuszelle mit „Absage"-Icon in `text-brand-text-subtle italic` gerendert

---

### Requirement: RSVP-Voreinstellungs-Editor im Bearbeiten-Modal

Die Bearbeiten-Modals für Trainings-Session, Trainings-Serie und Spiel (`TrainingEditModal.tsx`, `GameEditModal.tsx`, Series-Bulk-Formular in `AdminTrainingsPage.tsx`) SHALL zwei separate Radio-Gruppen anbieten, überschrieben mit der Sektionsüberschrift „RSVP-Voreinstellung":

- **„Kader-Spieler"** mit den drei Optionen „Standardmäßig zugesagt", „Standardmäßig abgesagt", „Keine automatische Rückmeldung" (gebunden an `rsvp_default_players`).
- **„Erweiterter Kader"** mit denselben drei Optionen (gebunden an `rsvp_default_extended`).

Die Checkbox „Begründung bei Absage erforderlich" (`rsvp_require_reason`) und die Radio-Option „Standardmäßig abgesagt" SHALL **frei kombinierbar** sein: es gibt keine gegenseitige `disabled`-Kopplung und keinen Sperr-Tooltip. Beide Kontrollen sind jederzeit bedienbar, da eine Default-Absage ohne Nutzerhandlung entsteht (kein Grund erhebbar) und `rsvp_require_reason` nur aktive Absagen betrifft — die Einstellungen wirken auf disjunkte Gruppen.

Die alte Checkbox „Alle Spieler standardmäßig zugesagt (Opt-Out)" entfällt vollständig.

#### Scenario: Radio-Auswahl wird gespeichert
- **WHEN** ein Trainer im TrainingEditModal die Radio-Option „Standardmäßig abgesagt" unter „Erweiterter Kader" wählt und speichert
- **THEN** enthält der `PUT`-Payload `rsvp_default_extended: "declined"`

#### Scenario: Reason-Checkbox lässt `declined`-Radios aktiv
- **WHEN** die Checkbox „Begründung bei Absage erforderlich" gesetzt ist
- **THEN** sind die Radios „Standardmäßig abgesagt" in beiden Rollen weiterhin `enabled` (frei wählbar)

#### Scenario: Aktive `declined`-Auswahl lässt Reason-Checkbox aktiv
- **WHEN** eine der beiden Voreinstellungen auf „Standardmäßig abgesagt" gesetzt ist
- **THEN** ist die Checkbox „Begründung bei Absage erforderlich" weiterhin `enabled` (frei setzbar)

