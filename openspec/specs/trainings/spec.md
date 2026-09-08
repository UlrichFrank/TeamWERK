# trainings Specification

## Purpose

Diese Spezifikation beschreibt die Capability `trainings`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
### Requirement: Training-Series mit Venue
Training-Series SHALL einen optionalen Venue (venue_id FK) statt eines Freitext-Ortsfeldes haben.

#### Scenario: Series mit Venue anlegen
- **WHEN** Nutzer legt Training-Series mit Venue an
- **THEN** venue_id wird gespeichert; Response enthält venue-Objekt

#### Scenario: Series ohne Venue
- **WHEN** Nutzer legt Training-Series ohne Venue an
- **THEN** venue_id ist null; kein Maps-Link wird angezeigt

---

### Requirement: Training-Session mit Venue
Training-Sessions SHALL einen optionalen Venue (venue_id FK) statt eines Freitext-Ortsfeldes haben. Der Venue der Series wird als Default vorausgefüllt und kann je Session überschrieben werden.

#### Scenario: Session erbt Venue der Series
- **WHEN** Nutzer öffnet Formular für neue einzelne Training-Session aus einer Series
- **THEN** venue_id wird mit dem Venue der Series vorausgefüllt (sofern gesetzt)

#### Scenario: Session mit abweichendem Venue
- **WHEN** Nutzer wählt anderen Venue für eine einzelne Session
- **THEN** Session-venue_id überschreibt den Series-Venue für diese Session

#### Scenario: Venue in Response eingebettet
- **WHEN** GET /api/trainings oder Training-Detail aufgerufen wird
- **THEN** Response enthält venue-Objekt (oder null) für Series und Session

---

> **Replaced**: Das Freitext-Feld `location TEXT` wurde durch strukturierte `venue_id` FK ersetzt. Bestehende location-Texte werden nicht migriert; Orte müssen neu als Venues angelegt werden.

### Requirement: Der Kader ist der Besitzer eines Trainings

Das System SHALL `training_sessions.kader_id` und `training_series.kader_id` (beide
NOT NULL, `REFERENCES kader(id) ON DELETE RESTRICT`) als **alleinigen** Besitzer eines
Trainingstermins bzw. einer Trainingsserie führen. Jede Auflösung der Frage „wer gehört zu
diesem Training?" — Sichtbarkeit, Anwesenheitserfassung, Serien-Abmeldung,
Erinnerungen, SSE-Zielmenge — SHALL über `kader_id` erfolgen, nicht mehr über die
Kombination `(team_id, season_id)`.

Die **Selbst-RSVP** (`POST /api/training-sessions/{id}/respond` ohne `member_id`) ist
davon ausgenommen: sie prüft die Kaderzugehörigkeit im Bestand an keiner Stelle — weder
für Mannschaften noch für Übungsgruppen — und wird von diesem Change bewusst nicht
geändert (siehe `design.md — Bekannter Rest`). Das Antworten für **andere**
(`member_id` gesetzt) bleibt unverändert an Eltern- und Staff-Rechte gebunden.

Damit gilt ein einziger Pfad für Mannschaftstrainings und Übungsgruppentrainings.

#### Scenario: Training für eine Übungsgruppe anlegen
- **WHEN** ein Trainer einer Übungsgruppe `POST /api/training-sessions` für diese Gruppe
  sendet
- **THEN** wird der Termin mit `kader_id` der Gruppe und `team_id IS NULL` angelegt
  (HTTP 201)

#### Scenario: Mitglied sieht den Termin
- **WHEN** ein Mitglied einer Übungsgruppe seine Trainingsliste abruft
- **THEN** enthält sie die Termine dieser Gruppe

#### Scenario: Nicht-Mitglied sieht den Termin nicht
- **WHEN** ein Nutzer, der weder Mitglied noch Trainer noch Elternteil eines Mitglieds
  dieser Übungsgruppe ist, seine Trainingsliste abruft
- **THEN** fehlen die Termine der Gruppe

#### Scenario: Eltern sehen den Termin
- **WHEN** ein über `family_links` verknüpftes Elternteil eines Gruppenmitglieds die
  Trainingsliste abruft
- **THEN** enthält sie die Termine der Gruppe

#### Scenario: RSVP und Anwesenheit funktionieren
- **WHEN** ein Mitglied einer Übungsgruppe auf einen Termin antwortet und der Trainer
  anschließend die Anwesenheit erfasst
- **THEN** werden Antwort und Anwesenheit gespeichert (HTTP 200/201), wie bei einem
  Mannschaftstraining

#### Scenario: Erinnerungen erreichen die Gruppe
- **WHEN** ein Übungsgruppentermin in 24 bzw. 3 Stunden beginnt
- **THEN** erhalten Mitglieder, deren Eltern und Trainer der Gruppe dieselben
  Erinnerungen wie bei einem Mannschaftstermin

#### Scenario: Trainings vergangener Saisons bleiben aufgelöst
- **WHEN** die aktive Saison wechselt und ein Training der Vorsaison abgerufen wird
- **THEN** wird sein Besitzer weiterhin über `kader_id` aufgelöst — die Auflösung hängt an
  der Saison des Trainings, nicht an der aktiven

#### Scenario: Kader mit Trainings ist nicht löschbar
- **WHEN** ein Kader gelöscht werden soll, an dem Trainingstermine oder -serien hängen
- **THEN** antwortet das System mit HTTP 409 und weder Kader noch Trainings werden entfernt

#### Scenario: Bestandsverhalten für Mannschaften unverändert
- **WHEN** ein Mannschaftstraining angelegt, gelesen, beantwortet oder abgehakt wird
- **THEN** ist das Verhalten identisch zu vorher — inklusive der Sichtbarkeit für den
  erweiterten Kader und dessen Eltern

### Requirement: `team_id` eines Trainings ist eine Projektion, kein Datenfeld

Das System SHALL `training_sessions.team_id` und `training_series.team_id` als
**nullable** führen. Der Wert SHALL gesetzt sein, wenn der besitzende Kader zu einer
Mannschaft gehört (`kader.team_id IS NOT NULL`), und **NULL**, wenn der Besitzer eine
Übungsgruppe ist.

`team_id IS NULL` bedeutet ausdrücklich „gehört einer Übungsgruppe" und SHALL **nicht** als
Datenfehler behandelt oder durch einen Backfill gefüllt werden. Diese NULL trägt den
gesamten Ausschluss der Übungsgruppen aus den team-gebundenen Auswertungen; sie zu füllen
würde alle betroffenen Flächen gleichzeitig öffnen.

Konsumenten außerhalb von `internal/trainings` — Anwesenheitsstatistik
(`internal/attendance`), Abwesenheitskalender (`internal/absences`), iCal-Feed
(`internal/calendar`), Dashboard, Videos, Scheduler-Statistiken — SHALL weiterhin über
`team_id` auflösen und Übungsgruppen dadurch ohne zusätzlichen Filter ausschließen.

#### Scenario: Statistik ignoriert Übungsgruppen
- **WHEN** `GET /api/teams/{id}/attendance-stats` für eine Mannschaft abgerufen wird und
  Mitglieder dieser Mannschaft zusätzlich in einer Übungsgruppe trainieren
- **THEN** fließen ausschließlich die Mannschaftstrainings in die Statistik ein

#### Scenario: Kalender-Feed ignoriert Übungsgruppen
- **WHEN** ein Mitglied einer Übungsgruppe seinen iCal-Feed abruft
- **THEN** enthält er keine Übungsgruppentermine

#### Scenario: Abwesenheitskalender ignoriert Übungsgruppen
- **WHEN** der Abwesenheitskalender einer Mannschaft abgerufen wird
- **THEN** werden Übungsgruppentermine nicht berücksichtigt

#### Scenario: Live-Updates erreichen die Gruppe trotzdem
- **WHEN** ein Übungsgruppentermin angelegt, geändert oder gelöscht wird
- **THEN** erhalten die Mitglieder, Eltern und Trainer der Gruppe ein SSE-Update — die
  Zielmenge wird über `kader_id` aufgelöst, nicht über das NULL-`team_id`

### Requirement: Migration bestehender Trainings ist verlustfrei und laut

Das System SHALL beim Backfill jeden bestehenden Trainingstermin und jede Serie genau dem
Kader zuordnen, dessen `(team_id, season_id)` mit dem des Datensatzes übereinstimmt. Diese
Zuordnung ist eindeutig, weil `kader` je Saison höchstens eine Zeile pro Team führt.

Findet sich zu einem Datensatz **kein** Kader, SHALL die Migration **abbrechen** (`kader_id`
ist NOT NULL) statt den Datensatz stillschweigend zu verlieren oder mit NULL zu füllen.

#### Scenario: Bestandstermine behalten ihren Besitzer
- **WHEN** Migration `058` auf einer Bestandsdatenbank läuft
- **THEN** trägt jeder Termin danach den `kader_id` des Kaders seines
  `(team_id, season_id)`, und `team_id` bleibt unverändert gesetzt

#### Scenario: Waisen-Datensatz bricht die Migration ab
- **WHEN** ein Trainingstermin existiert, dessen `(team_id, season_id)` keinen Kader hat
- **THEN** schlägt die Migration fehl und die Datenbank bleibt auf dem alten Stand

### Requirement: RSVP setzt Zugehörigkeit zum Termin voraus

Das System SHALL eine Antwort auf einen Trainingstermin
(`POST /api/training-sessions/{id}/respond`) nur zulassen, wenn das **Ziel-Mitglied** der
Antwort zum Kader des Termins gehört. Zugehörig ist, wer in `kader_members`,
`kader_extended_members` oder `kader_trainers` des `training_sessions.kader_id` steht.
Andernfalls SHALL das System mit HTTP **403** antworten und keine Zeile in
`training_responses` schreiben.

Diese Prüfung SHALL für **beide** Formen der Antwort gelten: für die Antwort auf den
eigenen Namen (ohne `member_id`) ebenso wie für die Antwort für andere (`member_id`
gesetzt). Die bestehende Eltern-/Staff-Prüfung des Fremd-Zweigs bleibt unverändert
bestehen; die Kaderprüfung tritt additiv hinzu.

Die Auflösung SHALL über `kader_id` laufen und dadurch für Mannschaften und Übungsgruppen
denselben Pfad nehmen.

#### Scenario: Fremder darf nicht antworten
- **WHEN** ein Nutzer mit Mitglieds-Datensatz, der weder im Stammkader noch im erweiterten
  Kader noch unter den Trainern des Termins steht, auf diesen Termin antwortet
- **THEN** antwortet das System mit HTTP 403 und `training_responses` bleibt unverändert

#### Scenario: Fremder darf nicht auf einen Übungsgruppen-Termin antworten
- **WHEN** derselbe Fall auf einem Termin einer Übungsgruppe (`kader.kind='practice'`)
  auftritt
- **THEN** antwortet das System ebenfalls mit HTTP 403 — dieselbe Prüfung, derselbe Pfad

#### Scenario: Stammkader, erweiterter Kader und Trainer dürfen
- **WHEN** ein Mitglied des Stammkaders, ein Mitglied des erweiterten Kaders oder ein
  Trainer des Termin-Kaders antwortet
- **THEN** wird die Antwort gespeichert (HTTP 204)

#### Scenario: Staff darf nicht für Termin-Fremde antworten
- **WHEN** ein Vorstand oder Trainer eine Antwort für ein Mitglied setzt, das nicht zum
  Kader des Termins gehört
- **THEN** antwortet das System mit HTTP 403 — die Staff-Berechtigung erlaubt, für andere
  zu antworten, nicht, Termin-Fremde einzutragen

#### Scenario: Anzeige und Antwortrecht stimmen überein
- **WHEN** `GET /api/training-sessions` einen Termin mit `am_i_participant: true` ausweist
- **THEN** wird eine Antwort desselben Nutzers auf diesen Termin angenommen — beide Fragen
  werden von derselben Funktion beantwortet

### Requirement: Termin-Sichtbarkeit folgt der Kader-Eintragung, nicht der Vereinsfunktion

Das System SHALL einen Trainingstermin in `GET /api/training-sessions` für jeden Nutzer
listen, der zum Kader des Termins gehört. Die Zugehörigkeit als Trainer SHALL allein an der
Eintragung in `kader_trainers` hängen; die Vereinsfunktion `trainer` SHALL dafür **nicht**
zusätzlich verlangt werden.

Grund ist die Invariante des Requirements oben: das Antwortrecht hängt an der Eintragung.
Verlangte die Sichtbarkeit zusätzlich die Vereinsfunktion, entstünde ein Termin, den ein
Nutzer verwalten und beantworten darf, aber nicht sieht. Die Konstellation ist vorgesehen —
die Trainer-Auswahl einer Übungsgruppe bietet den Funktionsfilter als abwählbare Checkbox
an, ein Gruppenleiter ohne Vereinsfunktion ist damit anlegbar.

#### Scenario: Kader-Trainer ohne Vereinsfunktion sieht seinen Termin
- **WHEN** ein Nutzer ohne Vereinsfunktion als Trainer eines Kaders eingetragen ist und
  `GET /api/training-sessions` aufruft
- **THEN** enthält die Liste die Termine dieses Kaders mit `am_i_participant: true`

