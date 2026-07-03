## Context

Heute steuert ein einziger Bool `rsvp_opt_out` (auf `training_sessions`, `training_series`, `games`) den RSVP-Default und **greift nur für Stammkader-Spieler** — der Code-Zweig `rsvpOptOut == 1 && !item.IsExtended` in `internal/trainings/handler.go:1409` (analog `internal/games/handler.go`) ist die Wahrheit. Der Erweiterte Kader ist per Design ausgeschlossen und hat effektiv immer „muss aktiv zusagen"-Semantik.

Die Trainer-Sonderregel (immer `confirmed`, hart-codiert) wurde im laufenden Change `termine-trainer-rsvp` (PR #127) etabliert und bleibt in diesem Proposal unangetastet.

Die UI-Kopplung liegt in drei Modals (`TrainingEditModal.tsx`, `GameEditModal.tsx`) und einem Series-Bulk-Formular (`AdminTrainingsPage.tsx`). Jedes rendert heute genau eine Checkbox.

## Goals / Non-Goals

**Goals:**
- Zwei orthogonale Voreinstellungen (Stammkader-Spieler, Erweiterter Kader) mit je drei Modi (`confirmed | declined | none`).
- UI-Texte in einfacher Sprache, kein „Opt-Out"/„Opt-In"-Jargon.
- Bestandsdaten migrieren ohne sichtbare Verhaltensänderung für existierende Termine.
- Widerspruch „Default-Absage + Grund erforderlich" mechanisch verhindern.
- Header-Zähler bleiben mit der Detail-Tabelle konsistent (auch bei Default-Werten).

**Non-Goals:**
- Trainer-Voreinstellung — bleibt hart `confirmed`, keine Enum-Spalte.
- Pro-Team-Voreinstellung bei Multi-Team-Spielen — ein Game hat weiter genau **eine** Voreinstellung pro Rolle.
- Neue Push-/Mail-Benachrichtigungen an Mitglieder mit Default-Absage.
- Historische Umrechnung archivierter Termine (das Migrations-Backfill ist reine Feld-Übersetzung).

## Decisions

### Zwei separate Enum-Spalten statt einem strukturierten JSON-Feld

Die neuen Voreinstellungen werden als **zwei skalare Enum-Spalten** pro Tabelle gespeichert (`rsvp_default_players TEXT CHECK (… IN ('confirmed','declined','none'))`, analog `rsvp_default_extended`).

- **Alternative**: eine JSON-Spalte `rsvp_defaults` mit `{"players": "confirmed", "extended": "none"}`.
- **Warum abgelehnt**: SQLite/`database/sql` haben keine native JSON-Struktur, jede Query bräuchte `json_extract`. Zwei Skalare sind indizierbar, `CHECK`-constraint-fähig und minimalinvasiv für die bestehenden `SELECT`-Queries.

### `rsvp_opt_out` entfernen statt parallel führen

Die alte Spalte wird per Migration **gelöscht**, nicht deprecated.

- **Alternative**: `rsvp_opt_out` behalten und aus den neuen Spalten computen (Trigger oder App-Code).
- **Warum abgelehnt**: doppelte Wahrheit driftet. Der Bool kann den neuen `declined`-Modus nicht abbilden, sobald jemand ihn nutzt — eine Rückwärtsprojektion wäre lossy. Sauberer Cut mit Backfill.

### Header-Zähler mit `COALESCE(response.status, default)` berechnen

Der Header-Counter zählt `confirmed`/`declined`/`maybe` heute per `LEFT JOIN` auf `training_responses`/`game_responses`. Für Default-Beteiligung wird die Aggregation um `COALESCE(r.status, default_for_role(m))` erweitert, wobei `default_for_role` je nach Zugehörigkeit (`kader_members` vs. `kader_extended_members`) den passenden Session-Default nachzieht.

- **Alternative**: nur aktive Antworten zählen, Zeilen-Anzeige weicht bewusst ab.
- **Warum abgelehnt**: Nutzer erwartet, dass die Zahl im Header dem entspricht, was die Tabelle darunter zeigt. Divergenz wäre ein Bug-Melder.

Trainer bleiben — wie bereits in `termine-trainer-rsvp` festgelegt — per `NOT IN (SELECT member_id FROM kader_trainers …)` aus der Zähl-Aggregation ausgeschlossen.

### Konfliktsperre `declined` + `rsvp_require_reason` in UI **und** Backend

Beide Seiten prüfen dieselbe Regel: mindestens eine der beiden Voreinstellungen darf `declined` sein **oder** `rsvp_require_reason=1` — nicht beides gleichzeitig.

- **UI**: Radio-Auswahl von `declined` deaktiviert die Reason-Checkbox mit Tooltip „nicht mit ‚standardmäßig abgesagt' kombinierbar"; umgekehrt sperrt gesetzte Reason-Checkbox die `declined`-Radios.
- **Backend**: Payload-Validierung wirft HTTP 400 mit `{"error":"invalid_rsvp_settings"}`, falls doch beide gesetzt kommen (defensiver Gürtel + Hosenträger für API-Konsumenten außerhalb der eigenen UI).

- **Alternative A**: Kombination stillschweigend erlauben; Default-Absagen brauchen dann keinen Grund.
- **Alternative B**: Kombination ignorieren, im Zweifel gewinnt eine Seite.
- **Warum abgelehnt**: beides führt zu subtiler, überraschender Semantik. Explizite Sperre ist ehrlich.

### Migrations-Backfill: konservativ

Backfill-Regeln (Migration `018`):
- `rsvp_default_players = 'confirmed'` wenn `rsvp_opt_out = 1`, sonst `'none'`.
- `rsvp_default_extended = 'none'` überall (das ist heute bereits das effektive Verhalten für den Erweiterten Kader).
- `DROP COLUMN rsvp_opt_out`.

Sichtbares Verhalten für existierende Termine ändert sich damit **nicht**.

### Virtuelle Default-Anzeige in der Detail-Tabelle

Zeilen ohne Response-Row zeigen den effektiven Default-Status. Damit ein Nutzer sieht, dass das keine explizite Antwort ist, wird der Status dezenter dargestellt: `text-brand-text-subtle italic`. Aktive Antworten bleiben in `text-brand-text` normal. Trainer-Zeilen behalten die bestehende (virtuelle) `confirmed`-Darstellung.

- **Alternative**: kein visueller Unterschied.
- **Warum abgelehnt**: sonst wirkt die Tabelle so, als hätten alle explizit zugesagt/abgesagt. Der Unterschied „das ist die Voreinstellung, keine echte Antwort" ist für Trainer bei der Personalplanung relevant.

## Risks / Trade-offs

**[Widerspruch stiller Server-Konsumenten]** — Externe API-Aufrufer, die weiterhin `rsvp_opt_out` senden, kriegen nach der Migration `unknown field` bzw. der Wert wird ignoriert. → **Mitigation**: `rsvp_opt_out` im Handler explizit ablehnen (HTTP 400 mit sprechender Message), damit der Aufrufer merkt, dass sich die Semantik geändert hat. Kein „silent no-op". Es gibt keine bekannten externen Aufrufer außer dem eigenen Frontend.

**[Header-Zähler-Query wird komplexer]** — Aggregation muss jetzt Response-Status ∪ Session-Default ∪ Rollen-Zuordnung kombinieren. → **Mitigation**: SQL bleibt lesbar durch getrennte `UNION`-Zweige pro Rolle (Stammkader / Erweitert / Trainer=NULL), jeweils mit `COALESCE`. Performance ist unkritisch (max. wenige Dutzend Mitglieder pro Termin). Deckung durch neue Tests.

**[UI-Sperre kann nervig sein]** — Trainer, der zuerst „standardmäßig abgesagt" wählt und dann Grund erforderlich setzen will, muss erst zurückschalten. → **Mitigation**: Tooltip erklärt die Sperre. Alternative wäre eine Warnmodale, die beim Bestätigen erscheint — komplexer, kein Mehrwert.

**[Bestehende Series-Copy-Semantik]** — `insertSessions` kopiert Serien-Felder beim Anlegen neuer Sessions. Wenn die Copy-Liste vergessen wird, erben neue Sessions nicht die Voreinstellung der Serie. → **Mitigation**: Test-Fall, der eine Serie mit `players='declined'` anlegt und prüft, dass eine generierte Session dieselben Werte trägt.

**[Alte Migrations müssten ebenfalls angepasst werden — nein]** — Migration `011_event_notes.up.sql` rekonstruiert Sessions/Games mit `rsvp_opt_out`. Die neue Migration `018` transformiert **das Ergebnis** dieser Rekonstruktion; die Historie in `011` bleibt unverändert (up-Migrationen sind append-only).

## Migration Plan

1. Migration `018_rsvp_defaults_per_role.up.sql` (+ `.down.sql`) implementieren und testen (`make migrate-up` / `make migrate-down` lokal roundtrip).
2. Backend-Handler auf neue Felder umstellen (Trainings zuerst, Games symmetrisch). Handler-Tests erweitern.
3. Frontend-Typen und Modals auf Radio-Gruppen umstellen. Bestehende `rsvp_opt_out`-Felder aus Frontend entfernen.
4. `TermineDetailPage` — virtuelle Default-Anzeige (dezent) für Zeilen ohne Response.
5. `pnpm -C web build/test/lint`, `go test ./...`, `openspec validate rsvp-defaults-per-rolle`.
6. Deploy als Standard-Release, kein Feature-Flag. Rollback = Code-Revert + Migration `018` down (Backfill der zwei Enums zurück in `rsvp_opt_out=1 ⇔ players='confirmed'`).

## Open Questions

Keine offen — Klärungen aus dem Explore-Modus (Konfliktsperre, Trainer bleibt, Zähler bezieht Default ein, konservatives Backfill) sind in den Decisions verankert.
