## Context

`sendDutyReminders` (`internal/scheduler/scheduler.go:524`) läuft als Teil von `Scheduler.Run()`,
das minütlich per Cron aufgerufen wird (`* * * * * teamwerk-scheduler.sh`,
docs/agent/02-workflow.md). Aktuell:

1. `targetDate := heute + 2 Tage`
2. Lädt alle `duty_slots` an `targetDate` mit `slots_filled < slots_total`
3. Ermittelt je Slot über `eligibleUsers(slot)` (Rolle + Team-Scope, `slotTeamScope`) alle User,
   die den Slot noch nicht selbst belegt haben
4. Verschickt pro User eine aggregierte E-Mail (`buildReminderMail`, Opt-in via
   `notification_preferences`) und einen Push (Kategorie `duty_reminders`, Opt-out, Default an)
5. Idempotenz: `duty_reminder_log(user_id, event_date)` PK für Mail, `notification_log(user_id,
   ref_type='duty_reminder', ref_id=hashDate(targetDate))` PK für Push — beide Keys tragen nur
   das Datum, keinen Offset.

Siehe proposal.md für die Motivation (mehrere Eskalationsstufen + Vorstands-Eingriffsmöglichkeit).

## Goals / Non-Goals

**Goals:**
- Vier unabhängige, wiederkehrende Erinnerungszeitpunkte (7/3/2/1 Tage) für dieselbe
  Empfängerlogik wie bisher (`eligibleUsers`/`slotTeamScope` bleiben inhaltlich unverändert).
- Eine zusätzliche, vereinsweit aggregierte Vorstands-Übersicht an 3/1 Tagen.
- Beides idempotent gegen den minütlichen Scheduler-Lauf, ohne bestehende Garantien zu brechen
  (keine Meldung bei voll belegten Slots, kein Mehrfachversand pro (User, Tag, Offset)).

**Non-Goals:**
- Keine Änderung an der Team-Scope-/Rollen-Auflösung selbst (bleibt exakt wie in den bestehenden
  Spec-Szenarien von `duty-reminder-emails`).
- Keine neue `notification_preferences.category` — die Vorstands-Übersicht hängt bewusst an
  `duty_reminders` (Auswahl aus der Klärung mit dem Nutzer).
- Kein Eingriff in `duty_slots`/`duty_assignments` selbst — reine Benachrichtigungslogik.
- Keine UI-Änderung (kein neuer Reminder-Konfigurationsscreen); die vier Offsets sind ein
  Code-Konstant, keine Admin-Einstellung.

## Decisions

### D1: Offsets als Konstanten-Slice, eine Schleife statt vier Kopien

`memberReminderOffsets = []int{7, 3, 2, 1}` (Tage vor Termin). `sendDutyReminders` iteriert diese
Liste; Slot-Query, `eligibleUsers`-Aufruf, Mail-/Push-Versand bleiben pro Iteration identisch zum
bisherigen Einzel-Lauf, nur `targetDate` und die Idempotenz-Keys (s. D2) hängen vom aktuellen
Offset ab. Alternative verworfen: vier separate Funktionen (`sendDutyReminders7d`, `…3d`, …) —
Code-Duplikation ohne Mehrwert, jede Änderung an Query/Text müsste viermal gepflegt werden.

### D2: Idempotenz — `days_before` als Teil des Schlüssels

- **E-Mail (`duty_reminder_log`):** Tabellen-Rebuild (SQLite kennt kein `ALTER TABLE … ADD
  COLUMN` mit neuem zusammengesetztem PK) analog zum Muster in Migration `018`
  (`training_sessions_new` → `DROP` → `RENAME`). Neue Spalte `days_before INTEGER NOT NULL
  DEFAULT 2`, neuer PK `(user_id, event_date, days_before)`. Backfill bestehender Zeilen mit
  `days_before = 2`, weil das der bisherige einzige Offset war — historisch korrekt, keine
  Erfindung.
- **Push (`notification_log`):** **Kein Schema-Wechsel.** `ref_type` ist bereits `TEXT`, also
  kodiert der Offset direkt im `ref_type`-String: `fmt.Sprintf("duty_reminder_%dd", offset)`
  statt des bisherigen konstanten `"duty_reminder"`. `ref_id` bleibt `hashDate(targetDate)`.
  Der bestehende PK `(user_id, ref_type, ref_id)` liefert die Eindeutigkeit pro (User, Tag,
  Offset) ohne jede Migration. Alternative verworfen: Offset in `ref_id` hineinrechnen (z. B.
  `hashDate(date)*10+offset`) — unnötig fragil (Kollisionsgefahr, wenn `hashDate` sich mal
  ändert) gegenüber der bereits vorhandenen `ref_type`-Dimension.
- **Vorstands-Übersicht:** komplett eigener Satz Keys, getrennt von der Mitglieder-Reminder-
  Idempotenz (andere Empfängergruppe, anderer Zweck — ein Vorstandsmitglied, das selbst auch
  `eligibleUsers`-Empfänger eines Slots ist, soll beide Meldungen unabhängig bekommen können).
  Push: neuer `ref_type` `fmt.Sprintf("duty_board_vorstand_%dd", offset)` in `notification_log`
  (kein Schema-Zusatz nötig, wie oben). E-Mail: neue Tabelle `duty_board_reminder_log(user_id,
  event_date, days_before, sent_at, PRIMARY KEY(user_id, event_date, days_before))` — bewusst
  keine Wiederverwendung von `duty_reminder_log`, um die beiden Zwecke (persönliche Zusage vs.
  Vereins-Übersicht) nicht in einer Tabelle zu vermischen; ein künftiger Report/eine Statistik
  über "wer hat wie oft erinnert bekommen" würde sonst falsch zählen.

### D3: Vorstands-Empfängerkreis und Aggregation

Empfänger: `member_club_functions.function = 'vorstand'` (aktueller Bestand, keine Saison-
Bindung nötig — Vereinsfunktionen sind nicht saisonal). Kein Team-Filter (vereinsweite
Übersicht, analog zur bestehenden "vereinsweiten Empfänger"-Fallback-Logik aus
`duty-reminder-emails`, aber hier bewusst *immer* vereinsweit, nicht nur beim Fehlen von
`team_id`). Aggregation: eine Query lädt alle offenen Slots an `targetDate` (dieselbe Slot-Liste
wie für die Mitglieder-Reminder desselben Offsets, kein zweiter Slot-Query nötig — die Vorstands-
Übersicht wird direkt im selben Durchlauf pro Offset aus der bereits geladenen `slots`-Liste
gebaut) und verschickt pro Vorstands-User eine Mail/Push mit allen Slots dieses Tages, analog zu
`buildReminderMail` (neue Funktion `buildBoardOverviewMail`, gleiche Grundstruktur, andere
Anrede/Framing: "eingreifen" statt "eintragen").

### D4: Text/Framing je Offset

`formatOffsetLabel(days int) string` liefert die Zeithorizont-Phrase für Betreff/Body: `1` →
"morgen", `2`–`7` → "noch N Tage". Wird sowohl in der Mitglieder- als auch der Vorstands-Mail
verwendet, damit beide Texte konsistent denselben Zeithorizont benennen.

## Risks / Trade-offs

- **[Risk] Vier Offsets bedeuten bis zu 4× mehr Push-/Mail-Volumen pro offenem Slot über die
  Woche.** → Bewusst gewollt (Eskalation ist der Zweck der Änderung); Opt-out bleibt über die
  bestehende `duty_reminders`-Kategorie möglich, keine neue Zwangs-Kategorie.
- **[Risk] `duty_reminder_log`-Rebuild-Migration auf Prod-DB (WAL, 1 GB RAM VPS) sperrt die
  Tabelle kurzzeitig.** → Tabelle ist klein (nur Reminder-Log-Zeilen, kein Hot-Path), Rebuild
  betrifft nur diese eine Tabelle, nicht `duty_slots`/`duty_assignments`. Läuft im normalen
  `make deploy`-Migrationsschritt.
- **[Risk] Ein Vorstandsmitglied, das selbst z. B. `spieler` ist, könnte zwei separate Mails
  (persönliche Erinnerung + Vorstands-Übersicht) am selben Tag bekommen.** → Akzeptiert und
  gewollt (unterschiedliche Handlungsaufforderung: "trag dich selbst ein" vs. "sieh dir die
  Gesamtlage an und greif ggf. ein"); beide sind einzeln über dieselbe Kategorie abschaltbar.
- **[Trade-off] Keine Admin-UI für die Offsets.** → Vier feste Werte als Code-Konstante sind für
  den aktuellen Bedarf ausreichend; eine konfigurierbare Liste wäre YAGNI ohne konkreten
  Folgebedarf.

## Migration Plan

1. Neue Migration `065_duty_reminder_offsets.up.sql`/`.down.sql`:
   - `duty_reminder_log`: Rebuild mit `days_before INTEGER NOT NULL DEFAULT 2`, neuer PK
     `(user_id, event_date, days_before)`, Backfill bestehender Zeilen mit `days_before=2`.
   - Neue Tabelle `duty_board_reminder_log(user_id, event_date, days_before, sent_at, PRIMARY
     KEY(user_id, event_date, days_before))`.
   - Down-Migration: `duty_board_reminder_log` droppen; `duty_reminder_log` zurück auf
     `(user_id, event_date)` rebuilden (Zeilen mit `days_before <> 2` gehen dabei verloren — sie
     existierten vor dem Up-Lauf nicht, das ist konsistent mit "additive Migrationen, kein
     Datenverlust bei Rollback auf den Vorzustand").
2. `sendDutyReminders` auf die Offset-Schleife umstellen (D1), Idempotenz-Keys wie in D2.
3. Neue Funktion für die Vorstands-Übersicht (separater, aber im selben Offset-Loop
   aufgerufener Zweig, nur bei `offset IN (3, 1)`).
4. Kein Rollback-Sonderfall nötig — folgt dem bestehenden `make deploy-rollback`/additive-
   Migrationen-Muster aus docs/agent/10-deployment.md.
