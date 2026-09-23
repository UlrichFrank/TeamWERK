# Proposal

## Why

Trainer planen Kader, Aufstellung und Trainingsinhalte auf Basis der RSVPs. Ändert ein Spieler kurz vor dem Termin seine Antwort (z. B. Zusage → Absage drei Tage vor dem Spiel), erfährt der Trainer davon nur, wenn er zufällig die Teilnehmerliste erneut öffnet. Gerade in der letzten Woche vor einem Termin ist eine solche Umentscheidung planungsrelevant und soll aktiv gemeldet werden.

## What Changes

- Ändert ein Spieler (selbst oder über ein Elternteil / Staff) den Status einer **bereits abgegebenen** RSVP zu einem Spiel oder Training, und beginnt der Termin in **höchstens 7 Tagen**, erhalten die **Trainer des betroffenen Kaders** eine Push-Benachrichtigung.
- Die Meldung nennt Spieler, Termin, alten und neuen Status sowie — falls angegeben — den Grund (Trainer sehen den Grund laut `rsvp-reason-visibility` ohnehin).
- Keine Meldung bei: erster Antwort (kein vorheriger Status), unverändertem Status (nur Grund geändert), Termin > 7 Tage entfernt oder bereits begonnen, Antwort eines Trainers dieses Kaders für sich selbst.
- Der Auslöser selbst wird nicht benachrichtigt (ein Trainer, der für einen Spieler umsagt, bekommt keine Meldung über seine eigene Handlung).
- Kategorie `operativ` (Profil-Schalter „Vereinsaufgaben“) — die Push ist dort abschaltbar, die Meldung erscheint unabhängig davon im Event-Log (`user_events`).
- Die Beschreibung des Schalters „Vereinsaufgaben“ im Profil (Tab „Sonstiges“) nennt künftig auch die Umentscheidungen der Spieler, damit Trainer die Meldung dort wiederfinden.
- Keine neue Route, keine Migration.

## Capabilities

### New Capabilities
- `rsvp-aenderung-trainer-push`: Push an die Trainer eines Kaders, wenn ein Spieler innerhalb von 7 Tagen vor einem Spiel/Training seine bestehende RSVP ändert.

### Modified Capabilities
<!-- keine: die RSVP-Routen behalten Vertrag, Autorisierung und Antwortcodes -->

## Impact

- `internal/games/handler.go` (`RespondToGame`), `internal/trainings/handler.go` (`Respond`): vorherigen Status vor dem Upsert lesen, nach erfolgreichem Upsert Meldung auslösen.
- `internal/notify`: neuer Helfer zur Auflösung der Trainer-Nutzerkonten eines Kaders und ein Textbaustein für die Änderungsmeldung.
- `web/src/components/profile/ProfileMiscTab.tsx`: Beschreibungstext der Kategorie `operativ`.
- Tests in beiden Domänen-Packages und in `internal/notify`.
- Nicht betroffen: Abwesenheiten (`internal/absences`), die RSVPs automatisch setzen — siehe design.md, Non-Goals.

## Test-Anforderungen

Keine neue Route; geänderte Geschäftslogik an zwei bestehenden Routen.

| Route | Test | Erwartet |
|---|---|---|
| `POST /api/games/{id}/respond` | `TestRespondToGame_AenderungImFenster_BenachrichtigtTrainer` | 204, genau eine `operativ`-Meldung an die Spiel-Trainer |
| `POST /api/games/{id}/respond` | `TestRespondToGame_ErsteAntwort_KeineMeldung`, `…_GleicherStatus_KeineMeldung`, `…_AchtTageVorher_KeineMeldung`, `…_TrainerEigeneAntwort_KeineMeldung` | 204, keine Meldung |
| `POST /api/games/{id}/respond` | `TestRespondToGame_AbsenceLock_KeineMeldung` | 403, keine Meldung |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_AenderungImFenster_BenachrichtigtTrainer`, `TestRespond_Uebungsgruppe_BenachrichtigtTrainer` | 204, Meldung an Kader-Trainer |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_SeriesUnavailable_KeineMeldung` | 403, keine Meldung |

**Invariante:** Eine Trainer-Meldung entsteht genau dann, wenn eine bestehende Antwort eines Nicht-Trainers ihren Status innerhalb von 7 Tagen vor Terminbeginn wechselt — und nie an den Auslöser selbst. Der HTTP-Status der RSVP ist davon unabhängig.
