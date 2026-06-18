# Release Notes: Auto-Duty-Optimization (v1.0)

## Summary

Spielplan-Mutationen triggern jetzt automatisch die Regeneration von Dienst-Slots für das betroffene Event-Datum sowie die benachbarten Tage. Die manuelle „Dienste generieren"-Aktion entfällt aus der UI — der Dienstplan wird konsistent mit den `same_day_behavior`- und `adjacent_day_behavior`-Regeln der Duty-Types gehalten.

## What's New

- **Automatische Dienst-Regeneration:** Beim Anlegen, Verschieben oder Löschen eines Heim- oder Auswärtsspiels regeneriert das System sofort die Dienst-Slots nach den aktuellen Template-Regeln.
- **Einfachere UI:** Der Knopf „Dienste generieren" ist nicht mehr nötig und wurde von `/kalender` und `/kalender/{id}` entfernt.
- **Schutz für manuelle Edits:** Manuell erstellte oder editierte Slots werden mit `is_custom=1` gekennzeichnet und vom Auto-Regen nicht angetastet.
- **Helfer-Benachrichtigungen:** Wenn ein Dienst durch Auto-Regen entfällt oder sich die Variante ändert, erhält der betroffene Helfer automatisch eine Push-Benachrichtigung.
- **Konsistenter Dienstplan:** Die Nachbarschaftsregeln (skip, reduce) werden immer befolgt — kein manueller Nachbau nötig.

## Breaking Changes

- **Endpoint-Responses ändern:** `POST /api/admin/kalender` und `PUT /api/admin/kalender/{id}` antworten jetzt mit Status 200/201 und enthalten zusätzlich ein `regen_summary`-Objekt. `DELETE /api/kalender/{id}` antwortet jetzt mit 200 (statt 204) und enthält auch `regen_summary`.
- **Slot-Array ignoriert (Heim/Auswärts):** Das `slots[]`-Array im Request wird für Heim- und Auswärtsspiele ignoriert — die Slots werden vom Backend erzeugt. Für generische Events bleibt `slots[]` erhalten.

## For Admins/Vorstand

### Vor dem Deploy

1. **Bestandsdaten schützen:** Falls Ihr bereits manuell-editierte Dienst-Slots im Spielplan habt (z.B. „wir brauchen diese Woche 4 statt 2 Kasse"), müssen diese mit `is_custom=1` gekennzeichnet werden, damit der Auto-Regen sie nicht überschreibt:
   ```sql
   UPDATE duty_slots SET is_custom=1 WHERE id IN (1, 5, 23, …);
   ```
   Eine Liste der potenziellen Kandidaten findet Ihr mit:
   ```sql
   SELECT ds.id, ds.event_name, ds.event_date, ds.duty_type_id, ds.slots_total
   FROM duty_slots ds
   JOIN game_templates gt ON ds.template_id = gt.id
   WHERE ds.slots_total > (SELECT slots_count FROM game_template_items gti WHERE gti.template_id = gt.id LIMIT 1)
   LIMIT 50;
   ```

2. **Test-Spielplan:** Testet im Staging/Test-System: legt Heimspiele an aufeinanderfolgenden Tagen an, beobachtet die Auto-Regen-Effekte.

### Nach dem Deploy

- Der Spielplan wird sofort bei jeder Änderung synchronisiert. Es muss nicht mehr explizit „Dienste generieren" geklickt werden.
- Helfer erhalten automatisch Push-Benachrichtigungen, wenn ihre Dienste sich durch Spielplan-Änderungen anpassen.
- Falls etwas schiefgeht oder Slots unerwartet regeneriert werden: DB-Admin kann per `UPDATE duty_slots SET is_custom=1 WHERE …` einzelne Slots schützen und `POST /api/kalender/regenerate-day?date=2026-06-15` aufrufen.

## Technical Details

- Neue DB-Migration 037: `duty_slots.is_custom INTEGER NOT NULL DEFAULT 0`
- Neue Funktionen `runAutoRegen`, `regenSingleDay` in `internal/games/regen.go`
- Handler `CreateGame`, `UpdateGame`, `DeleteGame` integrieren Auto-Regen vor Commit
- Frontend: `<RegenSummaryCard>` zeigt Änderungen nach dem Save an
- Tests: `TestCreateGame_AutoRegenSkipsAdjacentDay`, `TestUpdateGame_TimeChangeRegenSlots`, etc. decken Auto-Regen-Pfade ab

## Rollback

Falls größere Probleme auftreten:
1. Rollback Binary + DB-Migration auf Version vor 037
2. `make migrate-down` auf VPS
3. Alte `is_custom`-Spalte wird gelöscht, alte Slot-Struktur wiederhergestellt

## Feedback & Support

- Tretet ein Regression auf oder geht ein Slot verloren: bitte sofort Bescheid sagen mit Datum und Spieldetails
- Auto-Regen lädt sehr viele Slots auf einmal? Performance-Messungen durchführen (sollte < 500ms pro Mutation sein)
