## Context

Aktuell regeneriert `POST /api/admin/games/{id}/regenerate` Dienste für genau ein Spiel. Die Optimierungslogik (`applyBehavior`, `classifySlotPosition`) benötigt den vollständigen Tageskontext (alle Spielzeiten des Tages via `loadSameDayContext`), den das Backend bereits lädt — aber jedes Spiel wird isoliert gespeichert. Bei mehreren Heimspielen an einem Tag entsteht Inkonsistenz: Dienste sind erst korrekt optimiert, wenn alle Spiele einzeln regeneriert wurden.

Die Kern-Infrastruktur existiert bereits vollständig:
- `loadSameDayContext(date, seasonID)` → liefert alle Spielzeiten des Tages + Vor-/Folgetag-Flags
- `applyBehavior(...)` → wendet same_day/adjacent_day-Regeln an
- `classifySlotPosition(...)` → klassifiziert Dienste relativ zur Spielposition

Fehlend ist nur die Orchestrierung: ein Endpoint, der alle Spiele eines Tages in einer Transaktion verarbeitet.

## Goals / Non-Goals

**Goals:**
- Neuer Endpoint `POST /api/admin/games/regenerate-day?date=YYYY-MM-DD&season_id=N` verarbeitet alle Spiele eines Tages atomisch
- Für jedes Spiel wird das zugewiesene oder das passende Standard-Template verwendet
- Leere Slots aller Tagesspiele werden gelöscht, neue optimiert erzeugt — in einer DB-Transaktion
- Frontend: Button in der Kalenderansicht auf Tagesebene, mit Template-Übersicht pro Spiel und Bestätigungsdialog
- Warnung bei Konflikten (gleicher Diensttyp, gleiche Zeit, verschiedene Spiele) wird bereits von der Preview-Logik geliefert

**Non-Goals:**
- Einzel-Regenerierung wird nicht ersetzt oder entfernt
- Keine automatische Trigger-Logik (kein Hook beim Anlegen eines Spiels)
- Keine Änderung an der Optimierungslogik selbst

## Decisions

### 1. Einzelne Transaktion für den ganzen Tag

Alle Deletes und Inserts laufen in einer `BeginTx`-Transaktion. Bei Fehler rollback des gesamten Tages.

*Alternative: Pro Spiel eine Transaktion* — abgelehnt, weil ein Teilfehler den Tag inkonsistent hinterlässt.

### 2. Template-Auflösung pro Spiel

Für jedes Spiel gilt: gespeichertes `template_id` hat Vorrang, sonst wird über `findTemplateForGame(isHome)` das passende Standard-Template gesucht. Spiele ohne Template werden übersprungen (kein Fehler).

*Alternative: Ein Template für den ganzen Tag* — abgelehnt, da ein Tag Heim- und Auswärtsspiele mischen kann.

### 3. Nur leere Slots löschen

Wie bei Einzel-Regenerierung: `DELETE FROM duty_slots WHERE game_id=? AND slots_filled=0`. Belegte Slots bleiben erhalten; der Response meldet `kept_slots` pro Spiel.

### 4. Frontend-Einstieg: Tages-Klick im Kalender

Der bestehende Tages-Klick im Spielplan-Kalender öffnet bereits einen Create-Dialog. Neben dem bestehenden „Spiel anlegen"-Flow wird ein zweiter Bereich „Dienste für diesen Tag generieren" eingeblendet — aber nur wenn bereits Spiele an dem Tag existieren.

*Alternative: Button auf der Spieldetailseite* — abgelehnt, da der Nutzer sonst jedes Spiel einzeln aufrufen muss.

## Risks / Trade-offs

- **Mehrere Spiele mit gleichem Template-Typ** → `findTemplateForGame` nimmt immer das Template mit der niedrigsten ID. Bestehende Warnung im AdminDutyTemplatesPage deckt das ab.
- **Langer Spieltag (>4 Spiele)** → Transaktion hält SQLite-Write-Lock länger. Bei einem VPS mit 1 GB RAM und SQLite WAL unproblematisch (kein Concurrent Write-Load erwartet).
- **Keine Preview im Batch-Dialog** → Nutzer sieht nicht vorab, welche Dienste entstehen. Mitigation: Dialog listet Spiele + zugewiesene Templates auf; Konflikte werden nach der Generierung gemeldet.
