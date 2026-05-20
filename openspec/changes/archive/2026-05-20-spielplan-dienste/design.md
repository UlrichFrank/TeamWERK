## Context

TeamWERK verwaltet Dienste (Duty Slots) heute ohne Bezug zu konkreten Spielterminen. Admins legen Slots manuell an — pro Heimspiel 3–5 Slots (Aufbau, Bewirtung, Abbau etc.). Bei ~15 Heimspielen pro Mannschaft und mehreren Teams ist das fehleranfällig und zeitaufwendig. Es gibt keine Übersicht, welche Spieltage wie gut besetzt sind.

Diese Änderung führt ein `games`-Datenmodell ein, verknüpft `duty_slots` optional mit Spielen und generiert Slots automatisch über eine konfigurierbare Vorlage (Template). Das bestehende Dienstsystem bleibt vollständig abwärtskompatibel.

## Goals / Non-Goals

**Goals:**
- Heimspiele manuell erfassen und pro Saison/Team verwalten
- Konfigurierbare Vorlage (Template) definiert welche Slot-Typen + Zeitversatz pro Heimspiel entstehen
- Beim Anlegen eines Spiels werden Duty Slots automatisch aus dem Template generiert
- Spielplan-Kalender (Monatsansicht) mit Besetzungsampel pro Spieltag
- Spieltag-Detail mit Zeitleiste der verknüpften Dienste und Besetzungsstand
- `duty_slots.game_id` (nullable FK) verknüpft bestehende und neue Slots mit Spielen
- Schema ist vorbereitet für spätere externe Importe (H4A / Handball360) via `source`-Feld

**Non-Goals:**
- Import aus externen Quellen (Handball4All, Handball360) — separates Feature
- Auswärtsspiele (keine Dienste, werden nicht erfasst)
- Spielergebnisse / Statistiken
- Push-Benachrichtigungen bei Änderungen

## Decisions

### 1. Neues Package `internal/games/` statt Erweiterung von `duties`

**Entscheidung:** Eigenes Package `internal/games/` mit `Handler` und eigenem Routen-Block.

**Warum:** Spiele sind eine eigenständige Domäne (eigene Tabellen, eigene CRUD-Operationen). Das `duties`-Package würde bei Einbettung zu groß und schwer testbar. Das Handler-Struct-Pattern (`type Handler struct{ db *sql.DB }`) bleibt konsistent.

**Alternative:** Erweiterung von `duties` — abgelehnt, weil `games` + Templates konzeptuell getrennte Aggregate sind und das Package andernfalls zu viele Verantwortlichkeiten hätte.

### 2. Template als globale Konfiguration, nicht pro Team

**Entscheidung:** Eine aktive `game_template` gilt für alle Teams. Konfigurierbar nur durch Admin.

**Warum:** In der Praxis haben Heim-Dienste bei Team Stuttgart pro Spieltag dieselbe Struktur. Eine Template-pro-Team-Konfiguration würde Komplexität hinzufügen ohne heutigen Mehrwert.

**Alternative:** Template pro Team — als spätere Erweiterung offen gehalten (`game_templates.id` wird in `games.template_id` referenziert, mehrere Templates möglich).

### 3. Auto-Generierung beim Anlegen, nicht lazy

**Entscheidung:** Duty Slots werden sofort beim Speichern eines neuen Spiels in einer DB-Transaktion erzeugt.

**Warum:** Konsistenter State — kein Zustand "Spiel ohne Slots". Einfacheres Datenmodell: kein Job-Queue nötig.

**Risiko:** Wenn das Template nach Spiel-Anlegen geändert wird, stimmen ältere Slots nicht mehr mit dem Template überein. → Mitigation: Slots manuell nachjustierbar; Template-Änderungen wirken nur auf neue Spiele.

### 4. `duty_slots.game_id` nullable FK (additive Migration)

**Entscheidung:** Bestehende `duty_slots`-Zeilen bekommen `game_id = NULL`. Kein Backfill.

**Warum:** Rückwärtskompatibilität ohne Datenmigration. Die Dienstbörse und alle bestehenden Seiten funktionieren unverändert.

### 5. Kalender-Frontend: eigene Seite, keine Einbettung in bestehende Seiten

**Entscheidung:** Neue Route `/spielplan` mit MonatsGrid-Komponente und `/spielplan/:gameId` für Detailansicht.

**Warum:** Klare URL-Struktur, einfaches Routing in React Router v6. Kein State-Sharing mit anderen Seiten nötig.

## Risks / Trade-offs

**[Template-Drift]** Nach Anlegen mehrerer Spiele und nachträglicher Template-Änderung divergieren ältere generierte Slots vom aktuellen Template.
→ Mitigation: UI zeigt an, wann ein Spiel angelegt wurde; Admin kann Slots manuell anpassen oder Spiel löschen und neu anlegen.

**[Transaktionsgröße]** Bei einem Template mit vielen Items (z.B. 8+ Slots) wird eine größere Transaktion beim Spiel-Anlegen ausgeführt.
→ Mitigation: Unkritisch bei SQLite im WAL-Mode; realistisch sind 3–6 Items.

**[Kalender-Performance]** Monatsansicht lädt alle Spiele + deren Slot-Füllstände per JOIN.
→ Mitigation: SQLite-Query mit GROUP BY ist für <200 Spiele pro Saison unproblematisch auf dem VPS.

**[Löschen eines Spiels]** Generierte Duty Slots sind mit dem Spiel verknüpft (`game_id`). Beim Löschen eines Spiels: Slots bleiben erhalten (game_id wird NULL via ON DELETE SET NULL) oder werden kaskadiert gelöscht.
→ Entscheidung: ON DELETE SET NULL — kaskadiertes Löschen wäre riskant wenn Slots bereits belegt sind.

## Migration Plan

1. Migration `007_games.up.sql`:
   - Tabellen `game_templates`, `game_template_items`, `games` anlegen
   - `duty_slots.game_id` als nullable FK hinzufügen (ALTER TABLE via Rebuild-Pattern da SQLite kein ADD CONSTRAINT unterstützt)
2. Kein Datenmigrations-Backfill nötig — alle bestehenden Slots haben `game_id = NULL`
3. Deploy: `make deploy` führt `migrate up` aus (Migrations sind in Binary embedded)
4. Rollback: `007_games.down.sql` entfernt neue Tabellen und `game_id`-Spalte

## Open Questions

- Soll beim Löschen eines Spiels (das bereits belegte Slots hat) eine Warnung angezeigt werden, oder wird dies blockiert? → Vorschlag: Warnung + Bestätigung, keine Blockierung.
- Sollen Spiele aus vergangenen Saisons im Kalender sichtbar bleiben, oder nur die aktive Saison? → Vorschlag: Kalender zeigt standardmäßig aktive Saison, Saisonwechsel via Dropdown möglich.
