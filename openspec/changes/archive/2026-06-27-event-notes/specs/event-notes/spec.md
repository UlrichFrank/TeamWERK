# event-notes Specification

## ADDED Requirements

### Requirement: Terminbezogenes Hinweisfeld für Trainings und Spiele

Das System SHALL ein freitextliches **Hinweisfeld** an jedem Trainings-
und Spiel-Event bereitstellen. Das Feld SHALL ein einzelner String mit
maximal 200 Zeichen sein, persistiert in `training_sessions.note` bzw.
`games.note` mit `CHECK (length(note) <= 200)`. Es SHALL für alle
`event_type`-Varianten der Tabelle `games` (`heim`, `auswärts`, `generisch`)
gelten. Dienste (`duty_slots`) und Abwesenheiten (`member_absences`) sind
**nicht** Teil dieser Capability.

#### Scenario: Hinweisfeld an einem Training persistieren

- **WHEN** ein berechtigter Trainer `PUT /api/trainings/{id}/note` mit
  Body `{"note": "Halle gesperrt, wir joggen am See"}` aufruft
- **THEN** schreibt das Backend den Text in `training_sessions.note`
- **AND** liefert HTTP 200

#### Scenario: Hinweisfeld an einem generischen Event persistieren

- **WHEN** ein Vorstandsmitglied `PUT /api/games/{id}/note` für ein Game
  mit `event_type='generisch'` aufruft
- **THEN** wird der Text in `games.note` persistiert
- **AND** die Route liefert HTTP 200

#### Scenario: Zu langer Hinweistext wird abgelehnt

- **WHEN** `PUT /api/trainings/{id}/note` oder `PUT /api/games/{id}/note`
  einen Body mit `len(note) > 200` empfängt
- **THEN** liefert das Backend HTTP 400
- **AND** weder `training_sessions.note`/`games.note` noch
  `pending_event_notes_push` werden verändert

### Requirement: Berechtigungen zum Setzen des Hinweistexts

Das System SHALL den Hinweistext an einem Training nur für **Trainer:innen
des betroffenen Teams**, **Vorstand** und **Admin** beschreibbar machen. An
einem Spiel oder generischen Event SHALL der Hinweistext für **Vorstand**,
**Trainer:innen eines beteiligten Teams**, **sportliche_leitung** und
**Admin** beschreibbar sein. Andere Nutzer SHALL HTTP 403 erhalten.

#### Scenario: Trainer eines fremden Teams darf Training-Hinweis nicht setzen

- **WHEN** ein Trainer-User, dessen `member_club_functions` keine
  `trainer`-Zuordnung zum Team des Trainings enthält, `PUT
  /api/trainings/{id}/note` aufruft
- **THEN** liefert das Backend HTTP 403
- **AND** `training_sessions.note` bleibt unverändert

#### Scenario: Standard-User darf Spiel-Hinweis nicht setzen

- **WHEN** ein User ohne Vorstands-, Trainer- oder sportliche_leitung-
  Funktion `PUT /api/games/{id}/note` aufruft
- **THEN** liefert das Backend HTTP 403

### Requirement: Debounced Push-Notification bei Hinweis-Änderung

Das System SHALL **5 Minuten nach der letzten Änderung** eines Hinweistexts
eine Push-Notification an alle Mitglieder und Eltern der betroffenen Teams
versenden. Jeder neue `PUT`-Aufruf SHALL den 5-Minuten-Timer für diesen
Termin **zurücksetzen**, sodass mehrfache Korrekturen nur **einen** Push
auslösen. Ein Push SHALL **nie** für ein Event in der Vergangenheit
versendet werden (`event_date < today`). Ein Push SHALL **nie** ohne
Hinweistext versendet werden.

Die Debounce-Queue SHALL in der Tabelle `pending_event_notes_push (ref_type,
ref_id, note_text, notify_after, updated_by)` persistiert werden, mit
Primary Key `(ref_type, ref_id)`. Der Scheduler-Job SHALL minütlich laufen
und fällige Rows (`notify_after <= now`) abarbeiten. Die Row SHALL nach der
Verarbeitung **immer** gelöscht werden, unabhängig davon, ob ein Push
abgesetzt wurde.

#### Scenario: Erster Hinweistext erzeugt pending-Row mit notify_after = now+5min

- **WHEN** ein berechtigter Nutzer `PUT /api/{trainings|games}/{id}/note`
  mit nicht-leerem `note` aufruft
- **THEN** existiert in `pending_event_notes_push` eine Row mit
  `(ref_type, ref_id) = ('training'|'game', id)`, `note_text = note`,
  `notify_after ≈ now + 5 Minuten`

#### Scenario: Zweiter Edit innerhalb von 5 Minuten setzt Timer zurück

- **GIVEN** eine pending-Row mit `notify_after = t_0 + 5min`
- **WHEN** zur Zeit `t_1 < t_0 + 5min` ein weiterer `PUT …/note`-Aufruf
  erfolgt
- **THEN** wird `notify_after` auf `t_1 + 5min` aktualisiert
- **AND** `note_text` auf den neuen Text aktualisiert

#### Scenario: Leerer Hinweistext entfernt pending-Row ohne Push

- **GIVEN** eine pending-Row für ein Event
- **WHEN** ein berechtigter Nutzer `PUT …/note` mit Body `{"note": ""}`
  aufruft
- **THEN** wird die pending-Row gelöscht
- **AND** es wird kein Push versendet

#### Scenario: Scheduler versendet Push für zukünftiges Event und löscht Row

- **GIVEN** eine pending-Row mit `notify_after <= now` für ein Event mit
  `event_date >= today`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird `notify.Send` für `teamMembersAndParents(team_ids)` mit
  `category` `'trainings'` bzw. `'games'`, dem Hinweistext als Body und
  der Detail-URL als `url`-Argument aufgerufen
- **AND** die pending-Row wird gelöscht

#### Scenario: Scheduler unterdrückt Push für vergangenes Event

- **GIVEN** eine pending-Row mit `notify_after <= now` für ein Event mit
  `event_date < today`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die pending-Row wird trotzdem gelöscht (Aufräumen)

#### Scenario: Scheduler ignoriert noch-nicht-fällige Rows

- **GIVEN** eine pending-Row mit `notify_after > now`
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die Row bleibt unverändert in der Tabelle

#### Scenario: Scheduler verarbeitet pending-Row eines bereits gelöschten Events sauber

- **GIVEN** eine pending-Row, deren referenziertes Event in der Zwischenzeit
  gelöscht wurde (z. B. weil der DELETE-Handler die Cleanup-Logik nicht
  ausgeführt hat oder ein Race lief)
- **WHEN** der Scheduler-Tick läuft
- **THEN** wird **kein** Push versendet
- **AND** die pending-Row wird gelöscht

### Requirement: Hinweistext im iCal-Feed

Das System SHALL den Hinweistext eines Events in das `DESCRIPTION`-Feld der
iCal-Repräsentation übernehmen (Endpoint `GET /api/calendar/feed`). Damit
sehen externe Kalender-Apps (Apple Calendar, Google Calendar, Outlook) den
Hinweis automatisch.

#### Scenario: Training mit Hinweis erscheint mit DESCRIPTION im Feed

- **WHEN** ein User mit zugeordnetem Training, das einen nicht-leeren
  `note` hat, `GET /api/calendar/feed` aufruft
- **THEN** enthält der zurückgegebene iCal-Body für dieses Training eine
  Zeile `DESCRIPTION:<note>` (escaped per `escapeText`)

#### Scenario: Game mit Hinweis erscheint mit DESCRIPTION im Feed

- **WHEN** ein User mit zugeordnetem Game, das einen nicht-leeren `note`
  hat, `GET /api/calendar/feed` aufruft
- **THEN** enthält der iCal-Body für dieses Game eine Zeile
  `DESCRIPTION:<note>`

### Requirement: Live-UI-Update via SSE-Broadcast

Das System SHALL beim Setzen eines Hinweistexts unmittelbar (nicht
debounced) `h.hub.Broadcast("event-note")` aufrufen. Das Frontend SHALL
in allen Seiten und Komponenten, die Hinweise anzeigen (`DashboardPage`,
`KalenderPage`, `TerminePage`, `TermineDetailPage`, `EventInfoModal`), den
Event-Typ `'event-note'` im `useLiveUpdates`-Hook abonnieren und den
relevanten Daten-Reload auslösen.

#### Scenario: Anderer offener Browser sieht Hinweis sofort

- **GIVEN** Browser A und Browser B sind beide auf `/kalender` eingeloggt
- **WHEN** Browser A einen Hinweis an einem Training setzt
- **THEN** empfängt Browser B den SSE-Event `event-note`
- **AND** lädt seinen Termin-Datenstand neu, sodass das Indikator-Symbol
  und der Text ohne Reload sichtbar werden

### Requirement: Anzeige des Hinweises im Frontend

Das System SHALL einen vorhandenen Hinweistext (`note.trim() !== ""`) an
jeder Stelle, an der der Termin gerendert wird, mit einem
`<AlertTriangle>`-Icon (`lucide-react`) in `text-brand-danger` markieren.
Die Anzeigeform SHALL sich am verfügbaren Platz orientieren:

- **Kompakt** (Dashboard-Termin-Row, Kalender-Tag-Tile): nur Icon, voller
  Text über `title`-Attribut als Browser-Tooltip.
- **Breit** (`EventInfoModal`, Termin-Card in `/termine`, Detailseite
  `/termine/:id`): Icon plus voller Hinweistext in einer eigenen Zeile.

Berechtigte SHALL im `EventInfoModal` und auf der Detailseite einen
**Inline-Editor** (Textarea + 200-Zeichen-Counter + Speichern-Button)
zum Anlegen/Ändern des Hinweises erhalten.

#### Scenario: Kalender-Tile zeigt Icon ohne Text

- **WHEN** ein Termin im Kalender-Tag-Tile gerendert wird und einen
  nicht-leeren Hinweis hat
- **THEN** rendert die Tile ein `<AlertTriangle>`-Icon
  (`text-brand-danger`)
- **AND** das Button-Element trägt `title="Hinweis: <text>"`
- **AND** der Hinweistext selbst wird **nicht** als sichtbarer Text in
  der Tile gerendert

#### Scenario: Termin-Card in /termine zeigt Icon plus vollen Text

- **WHEN** eine Termin-Card in `/termine` gerendert wird und einen
  nicht-leeren Hinweis hat
- **THEN** rendert die Card unter dem `MapsLink` eine eigene Zeile mit
  `<AlertTriangle>`-Icon und dem vollständigen Hinweistext

#### Scenario: Berechtigter Trainer sieht Inline-Editor im EventInfoModal

- **WHEN** ein Trainer eines beteiligten Teams das `EventInfoModal` für
  ein Training öffnet
- **THEN** ist eine Textarea + Speichern-Button sichtbar, mit dem 200-
  Zeichen-Counter
- **AND** Speichern ruft `PUT /api/trainings/{id}/note`

#### Scenario: Standard-User sieht im EventInfoModal nur die Anzeige

- **WHEN** ein Standard-User ohne Schreibrecht das `EventInfoModal` öffnet
- **THEN** ist (bei vorhandenem Hinweis) Icon + Text sichtbar
- **AND** kein Editor und kein Edit-Button sichtbar
