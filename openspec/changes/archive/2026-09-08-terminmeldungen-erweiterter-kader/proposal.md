## Why

Wer im **erweiterten Kader** steht, sieht alle Termine seiner Mannschaft in App und
Kalender — bekommt aber je nach Domäne eine andere Benachrichtigung, weil die
Empfängermenge dreimal getrennt implementiert ist und dreimal etwas anderes antwortet:

| Meldung | Stammkader | dessen Eltern | erw. Kader | dessen Eltern | Trainer |
|---|---|---|---|---|---|
| Spiel angelegt / geändert / abgesagt (`internal/games`) | ✅ | ✅ | ❌ | ❌ | ❌ |
| Training abgesagt / verschoben / gelöscht (`internal/trainings`) | ✅ | ✅ | ✅ | ❌ | ❌ |
| Erinnerung 24 h / 3 h, Termin-Hinweis (`internal/scheduler`) | ✅ | ✅ | ❌ | ❌ | ❌ |

Praktisch heißt das: ein Spieler des erweiterten Kaders erfährt von einem Spiel **gar
nicht** — der Kalendereintrag ist seine einzige Einladung, und der wird gepollt, nicht
zugestellt. Beim Training erfährt er es selbst, seine Eltern aber nicht — bei
Minderjährigen ist das die falsche Hälfte, denn gefahren wird von zu Hause. Und die
Unterschiede sind niemandem erklärbar: sie sind kein Produktentscheid, sondern das
Ergebnis dreier Kopien derselben Funktion (`teamMembersAndParents`), die über die Jahre
auseinandergelaufen sind.

Die letzte Spalte trägt denselben Fehler in reiner Form: **Trainer stehen in keiner der
drei Auflösungen** (`kader_trainers` kommt in keiner vor). Ein Co-Trainer erfährt die
Verlegung des eigenen Spiels nur, wenn er zufällig auch Spieler ist.

## What Changes

- **Eine** Empfängerregel für alle Meldungen zu Mannschaftsterminen: Stammkader **und**
  erweiterter Kader jeweils **samt Eltern** (via `family_links`), dazu die **Trainer** des
  Kaders — aufgelöst über die aktive Saison. Sie gilt für Spiel-Meldungen,
  Trainings-Meldungen, die 24-h-/3-h-Erinnerungen und die debounced Termin-Hinweise.
- Der **Auslöser bleibt in der Menge**: wer einen Termin anlegt, ändert oder absagt,
  bekommt die eigene Meldung mit — als Bestätigung, dass sie raus ist, und als Beleg des
  Wortlauts, den die Mannschaft gelesen hat.
- Die drei Kopien von `teamMembersAndParents` (`internal/games`, `internal/trainings`,
  `internal/scheduler`) werden durch **einen** Helfer in der Foundation-Schicht ersetzt.
  Ein Architektur-Test hält die Regel danach mechanisch an einer Stelle fest — im selben
  Stil wie das bestehende Broadcast- und Push-Fan-out-Gate.
- **Mehr Empfänger, keine neue Meldung:** Anlass, Text, Kategorie, `url` und das
  `silent`-Flag bleiben unverändert. Wer Push für eine Kategorie abgeschaltet hat, bekommt
  weiterhin keine — sieht die Meldung aber wie alle anderen im Event-Log.
- **Nicht betroffen: Dienste.** Die Dienstbörse und ihre Erinnerungen sprechen weiter den
  Stammkader an. Die Dienstpflicht folgt der Kader-Zugehörigkeit (`duty_accounts` rechnet
  über `player_memberships`); wer im erweiterten Kader aushilft, schuldet dem Verein keine
  Dienststunden und soll auch nicht dazu aufgefordert werden.
- **Nicht betroffen: Chat.** Team-Gruppen lösen den erweiterten Kader längst mit auf.

## Capabilities

### New Capabilities

- `terminmeldung-empfaenger`: die eine Regel, wer eine Benachrichtigung zu einem
  Mannschaftstermin bekommt — inklusive der Abgrenzung zu Diensten und zur Sichtbarkeit.

### Modified Capabilities

- `push-games`: Empfängermenge umfasst ausdrücklich den erweiterten Kader, dessen Eltern
  und die Trainer des Kaders.
- `push-trainings`: Empfängermenge umfasst ausdrücklich die Eltern der erweiterten
  Kader-Mitglieder (die Mitglieder selbst waren bereits enthalten) und die Trainer.
- `push-reminders`: die 24-h-/3-h-Erinnerungen zu Spiel und Training gehen an dieselbe Menge.
- `event-notes`: der debounced Hinweis-Push geht an dieselbe Menge.

## Impact

- `internal/notify/audience.go` (neu) — `TeamAudience(db, teamIDs…)` als einzige Auflösung
- `internal/games/handler.go` — `teamMembersAndParents` entfällt (3 Aufrufstellen)
- `internal/trainings/handler.go` — `teamMembersAndParents` entfällt (4 Aufrufstellen)
- `internal/scheduler/scheduler.go`, `internal/scheduler/event_notes_push.go` —
  `teamMembersAndParents` / `teamMembersAndParentsMulti` entfallen
- `internal/arch/` — neues Gate gegen erneute Kopien der Empfängerauflösung
- Keine Migration, kein Schema, keine neue Route, keine neue Abhängigkeit, keine
  Frontend-Änderung. Wirksam ab Deploy; für Bestandstermine gibt es keine Nachmeldung.

## Test-Anforderungen

| Route / Job | Test | Erwartung |
|---|---|---|
| `POST /api/games` | `TestCreateGame_ErwKaderUndElternWerdenBenachrichtigt` | Mitglied nur in `kader_extended_members` **und** sein Elternteil aus `family_links` stehen in der Empfängermenge; Stammkader unverändert enthalten |
| `PUT /api/games/{id}` | `TestUpdateGame_ErwKaderWirdBenachrichtigt` | Änderungsmeldung erreicht den erweiterten Kader; Text, Kategorie `games` und `url` unverändert |
| `DELETE /api/games/{id}` | `TestDeleteGame_ErwKaderWirdBenachrichtigt` | Absagemeldung erreicht den erweiterten Kader; `url` bleibt der leere String |
| `DELETE /api/games/{id}` | `TestDeleteGame_SilentUnterdruecktAuchErwKader` | `{"silent":true}` mit Capability `suppress_event_notification` → **kein** Empfänger, auch nicht im erweiterten Kader; Broadcast läuft trotzdem |
| `PUT /api/training-sessions/{id}` | `TestUpdateSession_ElternDesErwKadersWerdenBenachrichtigt` | Elternteil eines nur erweiterten Mitglieds ist Empfänger (heute fehlt genau dieser Zweig) |
| `DELETE /api/training-series/{id}` | `TestDeleteSeries_ErwKaderUndElternWerdenBenachrichtigt` | Serienabsage erreicht erweiterten Kader und dessen Eltern |
| Scheduler `game_reminder_24h` / `_3h` | `TestGameReminder_ErwKaderErhaeltReminder` | erweitertes Mitglied + Elternteil bekommen den Reminder; `notification_log` verhindert das Duplikat pro Nutzer und Slot |
| Scheduler `training_reminder_24h` / `_3h` | `TestTrainingReminder_ErwKaderErhaeltReminder` | dito für Trainingseinheiten mit Status `active` |
| Scheduler Termin-Hinweis | `TestEventNotePush_ErwKaderErhaeltHinweis` | fällige `pending_event_notes_push`-Row erreicht den erweiterten Kader; Row wird gelöscht |
| — (Einheit) | `TestTeamAudience_FuenfGruppen` | Stammkader, dessen Eltern, erw. Kader, dessen Eltern und Trainer sind Empfänger; ein Vereinsmitglied ohne Kader-Zugehörigkeit nicht |
| — (Einheit) | `TestTeamAudience_TrainerOhneElternZweig` | ein `family_link` auf einen Trainer erzeugt **keinen** Empfänger |
| `POST /api/games` | `TestCreateGame_AusloeserBekommtDieEigeneMeldung` | der anlegende Trainer steht in der Empfängermenge seiner eigenen Meldung |
| — (Einheit) | `TestTeamAudience_KeineDoppeltenEmpfaenger` | Mitglied in beiden Kader-Listen und in zwei betroffenen Mannschaften erscheint **genau einmal**; Mitglied ohne `user_id` erscheint nicht; Kader einer inaktiven Saison zählt nicht |
| — (Einheit) | `TestTeamAudience_LeereTeamListe` | leere Team-Liste liefert leere Menge ohne Query und ohne Fehler |
| — (Architektur) | `TestKeineEigeneEmpfaengerAufloesung` | außerhalb von `internal/notify` verbindet keine Funktion `player_memberships` mit `family_links`, außer den begründeten Allowlist-Einträgen der Dienst-Pfade; ein verwaister Allowlist-Eintrag lässt den Test ebenfalls fehlschlagen |

**Garantierte Invariante:** Für ein und denselben Termin liefert jede Meldung — Anlage,
Änderung, Absage, 24-h-/3-h-Erinnerung, Termin-Hinweis — **dieselbe** Empfängermenge. Kein
Test darf eine Terminmeldung grün lassen, die den erweiterten Kader auslässt.
