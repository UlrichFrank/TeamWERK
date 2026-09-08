## ADDED Requirements

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
