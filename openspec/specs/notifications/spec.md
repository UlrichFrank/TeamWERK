# notifications Specification

## Purpose

Zentrale Notification-Fassade, die Push und Email für alle Domänen-Events koordiniert. Aufrufer übergeben Empfänger-IDs, Kategorie und Nachrichteninhalt — die Fassade entscheidet anhand der gespeicherten `notification_preferences`, welcher Kanal (Push, Email, beide) pro Nutzer verwendet wird.
## Requirements
### Requirement: Kategoriebasierte Notification-Fassade

Das System SHALL eine zentrale Funktion `notify.Send(db, cfg, uids, category, title, body, url, opts ...Option)` bereitstellen, die für die übergebenen Nutzer-IDs anhand der gespeicherten `notification_preferences` der angegebenen Kategorie automatisch Push und/oder Email auslöst. Aufrufer DÜRFEN NICHT mehr direkt zwischen Push- und Email-Versand entscheiden.

Die Fassade SHALL:

- **jedem** übergebenen Nutzer eine Event-Log-Zeile schreiben — **vor** und **unabhängig von** jeder Präferenz-Filterung (siehe Capability `event-log`)
- nur Push an Nutzer schicken, deren `notification_preferences.push_enabled = 1` für die Kategorie ist (Default 1, wenn keine Zeile existiert)
- nur Email an Nutzer schicken, deren `notification_preferences.email_enabled = 1` für die Kategorie ist (Default 0, wenn keine Zeile existiert)
- den Email-Versand asynchron pro Empfänger als Goroutine ausführen, damit der HTTP-Response nicht blockiert
- für leere Empfängerlisten still und ohne Fehler zurückkehren — ohne Log-Zeile

`notification_preferences` SHALL ausschließlich den **Zustellkanal** steuern, niemals die **Sichtbarkeit** im Event-Log.

Die Fassade SHALL zwei funktionale Optionen anbieten:

- `notify.NoEmail()` — unterdrückt den Email-Zweig vollständig. Für Meldungen, die bewusst push-only sind oder deren Aufrufer eine eigene, fachlich abweichende Mail baut.
- `notify.SkipPushPref()` — sendet Push unabhängig von `push_enabled`. Ausschließlich für Meldungen, deren Nichtzustellung Datenverlust bedeutet. Der Email-Zweig bleibt präferenzgesteuert.

Beide Optionen SHALL den Event-Log-Fan-out unberührt lassen.

Die Signatur SHALL variadisch sein, damit bestehende Aufrufstellen unverändert gültig bleiben.

#### Scenario: Nutzer hat nur Push aktiv (Default)

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [u1], "duties", ...)` mit einem Nutzer ohne Preference-Zeile aufruft
- **THEN** erhält der Nutzer eine Push Notification
- **THEN** erhält der Nutzer keine Email
- **THEN** entsteht genau eine `user_events`-Zeile für u1

#### Scenario: Nutzer hat Email aktiv für die Kategorie

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [u1], "duties", ...)` mit einem Nutzer aufruft, der `email_enabled=1` für `duties` hat
- **THEN** wird eine Email an die in `users.email` hinterlegte Adresse verschickt
- **THEN** wird, sofern `push_enabled=1` ist, zusätzlich eine Push verschickt
- **THEN** enthält die Email den `body`-Text plus eine Zeile „Direktlink: https://intern.team-stuttgart.org{url}"
- **THEN** entsteht genau eine `user_events`-Zeile für u1

#### Scenario: Nutzer hat Push deaktiviert und Email aktiv

- **WHEN** ein Nutzer `push_enabled=0` und `email_enabled=1` für `games` hat und das System ein Spiel-Ereignis verschickt
- **THEN** erhält der Nutzer eine Email
- **THEN** erhält der Nutzer keine Push
- **THEN** entsteht **trotzdem** genau eine `user_events`-Zeile für den Nutzer

#### Scenario: Nutzer hat Push und Email deaktiviert

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [u1], "duties", ...)` mit einem Nutzer aufruft, der `push_enabled=0` und `email_enabled=0` für `duties` hat
- **THEN** erhält der Nutzer weder Push noch Email
- **THEN** entsteht **trotzdem** genau eine `user_events`-Zeile für u1 — der Log ist der einzige Kanal, der ihn erreicht

#### Scenario: NoEmail unterdrückt nur den Email-Zweig

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [u1], "carpooling", ..., notify.NoEmail())` mit einem Nutzer aufruft, der `email_enabled=1` für `carpooling` hat
- **THEN** wird keine Email verschickt
- **THEN** wird Push nach Präferenz verschickt
- **THEN** entsteht genau eine `user_events`-Zeile für u1

#### Scenario: SkipPushPref ignoriert die Push-Präferenz

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [u1], "sonstiges", ..., notify.SkipPushPref())` mit einem Nutzer aufruft, der `push_enabled=0` für `sonstiges` hat
- **THEN** wird die Push trotzdem verschickt
- **THEN** richtet sich der Email-Versand weiterhin nach `email_enabled`
- **THEN** entsteht genau eine `user_events`-Zeile für u1

#### Scenario: Leere Empfängerliste

- **WHEN** ein Aufrufer `notify.Send(db, cfg, [], "duties", ...)` aufruft
- **THEN** kehrt die Fassade ohne Fehler zurück
- **THEN** entsteht keine `user_events`-Zeile

### Requirement: Push-Fan-out ausschließlich über die Fassade

Alle Pfade, die Push an Nutzergruppen versenden, SHALL `notify.Send` verwenden. Direkte Aufrufe von `push.SendToUsers` außerhalb von `internal/notify` und `internal/push` SIND NICHT ZULÄSSIG.

Diese Regel SHALL mechanisch durch einen Architektur-Test (`internal/arch/pushfanout_test.go`) erzwungen werden, der alle `internal/`-Packages parst. Ausnahmen SHALL ausschließlich über eine Allowlist mit Begründung möglich sein; ein Allowlist-Eintrag, der auf keinen real existierenden Aufruf zeigt, SHALL den Test fehlschlagen lassen.

`push.SendToUserWithBadge` in `internal/chat` SHALL die einzige zulässige Ausnahme sein — eigener Zustellkanal, eigener App-Icon-Badge, bewusst nicht im Event-Log (Chat hat eigene Ungelesen-Zähler).

Der Grund für die Regel ist der Event-Log: ein Sender, der die Fassade umgeht, erzeugt eine Lücke, die nichts kaputt macht und deshalb unbemerkt bleibt.

#### Scenario: Push-only-Meldung ohne Email

- **WHEN** eine Mitfahr-Paarung bestätigt wird
- **THEN** ruft `carpooling.ConfirmPairing` `notify.Send(..., "carpooling", ..., notify.NoEmail())` auf — nicht mehr `push.FilterByPushPref` + `push.SendToUsers` direkt
- **THEN** entsteht eine `user_events`-Zeile für den Empfänger

#### Scenario: Datenverlust-Warnung ignoriert die Push-Präferenz

- **WHEN** der Scheduler eine bevorstehende Video-Löschung ankündigt
- **THEN** ruft er `notify.Send(..., "sonstiges", ..., notify.SkipPushPref())` auf
- **THEN** erhält auch ein Trainer mit `push_enabled=0` die Push
- **THEN** entsteht eine `user_events`-Zeile für jeden betroffenen Trainer

#### Scenario: Aufrufer mit eigener Email behält sie

- **WHEN** der Scheduler den Dienst-Reminder verschickt
- **THEN** ruft er `notify.Send(..., "duty_reminders", ..., notify.NoEmail())` für Push und Log auf
- **THEN** verschickt er seine eigene, strukturierte Reminder-Mail unverändert daneben
- **THEN** bleibt der `notification_log`-Idempotenzschutz **vor** dem `notify.Send`-Aufruf, damit ein zweiter Cron-Lauf keine doppelten Log-Zeilen schreibt

#### Scenario: Architektur-Test schlägt bei neuem Direktaufruf an

- **WHEN** ein Package außerhalb von `internal/notify`/`internal/push` `push.SendToUsers` aufruft und nicht in der Allowlist steht
- **THEN** schlägt `TestArchitecture_KeinDirekterPushFanout` fehl

#### Scenario: Verwaister Allowlist-Eintrag

- **WHEN** ein Allowlist-Eintrag auf ein Package zeigt, das `push.SendToUsers` nicht mehr aufruft
- **THEN** schlägt `TestArchitecture_PushAllowlistOhneWaisen` fehl

