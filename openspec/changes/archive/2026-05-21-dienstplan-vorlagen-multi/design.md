## Context

Aktuell gibt es genau eine globale Dienstplan-Vorlage (`game_templates`, `is_active=1 LIMIT 1`). Die Slot-Generierung (`CreateGame`, `RegenerateSlots`, `PreviewSlots`) liest immer diese eine Vorlage. Heim- und Auswärtsspiele sowie Turniere erfordern aber unterschiedliche Dienstpläne (z. B. kein Auf-/Abbau bei Auswärtsspielen). Der Admin/Trainer muss heute manuell Slots anpassen, weil es kein Template für Auswärtsspiele gibt.

Das REST-Interface (`/api/admin/game-template`) ist nicht öffentlich und wird nur intern (Web-Frontend) genutzt.

## Goals / Non-Goals

**Goals:**
- Mehrere Dienstplan-Vorlagen parallel verwaltbar machen (`template_type`: `heim`, `auswärts`, `generisch`)
- Slot-Generierung wählt Vorlage automatisch nach Spieltyp (`is_home`)
- UI: Listenansicht + Detailseite (Pattern: Mitglieder-Seite)
- Löschen einer Vorlage aus der Listenansicht
- REST-Umbenennung: `/api/admin/game-template` → `/api/admin/duty-templates`
- UI-Umbenennung: „Spiel-Vorlage" → „Dienstplan-Vorlage"

**Non-Goals:**
- Vorlagen je Team (alle Teams teilen die Vorlagen)
- Historisierung: welche Vorlage wurde für ein Spiel verwendet
- Import/Export von Vorlagen
- Vorlagen je Saison

## Decisions

### 1. `template_type` als TEXT CHECK statt ENUM

SQLite unterstützt kein echtes ENUM. `CHECK(template_type IN ('heim','auswärts','generisch'))` ist der SQLite-idiomatische Weg.

**Alternativen erwogen:** INTEGER-Mapping (0/1/2) — abgelehnt, da schlechte Lesbarkeit im DB-Dump.

### 2. `is_active` bleibt in der DB, wird aber nicht mehr verwendet

Die Spalte bleibt für die Migration erhalten, damit bestehende Daten (die eine Vorlage mit `is_active=1`) nicht verloren gehen. Die bestehende Vorlage erhält automatisch `template_type='generisch'` in der Migration.

**Alternativen erwogen:** Spalte sofort droppen — abgelehnt, SQLite hat kein `DROP COLUMN` vor Version 3.35, und modernc.org/sqlite ist >= 3.35, aber das Risiko auf dem VPS ist gering. Dennoch: Spalte zu behalten ist simpler und ohne Risiko.

### 3. Fallback-Logik bei Slot-Generierung

Vorlage-Suche: `template_type = X` → falls nicht gefunden, `template_type = 'generisch'` → falls immer noch nicht gefunden, Fehler (kein stiller Fallback).

**Alternativen erwogen:** Immer die erste Vorlage nehmen — abgelehnt, da unvorhersagbar wenn mehrere Vorlagen vom selben Typ existieren. Besser expliziter Fehler mit Hinweis.

### 4. Mehrere Vorlagen gleichen Typs erlaubt

Die DB erlaubt mehrere `heim`-Vorlagen. Die Slot-Generierung nimmt die erste (`ORDER BY id ASC`). Die UI zeigt alle an; der Admin ist verantwortlich für Konsistenz.

**Alternativen erwogen:** UNIQUE-Constraint auf `template_type` — abgelehnt, weil es die Verwaltung (Vorlage kopieren, anpassen, dann alte löschen) unnötig einschränkt.

### 5. Breaking REST Change ohne Deprecation-Periode

`/api/admin/game-template` wird direkt auf `/api/admin/duty-templates` umgestellt. Der Endpunkt ist nicht öffentlich; Frontend und Backend werden im selben Deployment gewechselt.

### 6. Frontend-Aufteilung

`AdminGameTemplatePage.tsx` → aufgeteilt in:
- `AdminDutyTemplatesPage.tsx` — Tabelle mit allen Vorlagen, Löschen-Aktion
- `AdminDutyTemplateDetailPage.tsx` — Detailseite (Items bearbeiten, Typ setzen, Speichern)

Dieses Pattern ist identisch mit `MembersPage` / `MemberDetailPage` und konsistent mit dem bestehenden Coding-Stil.

## Risks / Trade-offs

- **Mehrere Vorlagen gleichen Typs** → Slot-Generierung nimmt immer die erste (ORDER BY id). Wenn der Admin versehentlich zwei `heim`-Vorlagen hat, ist nicht offensichtlich welche verwendet wird. → Mitigation: UI-Warnung in der Liste wenn `template_type` doppelt vorkommt.
- **Bestehende `game_template`-Aufrufe** → nach dem Deploy gibt es einen kurzen Moment wo altes Frontend auf neuen Backend trifft oder umgekehrt. → Mitigation: Atomarer Deploy (build + restart in einem Schritt via `make deploy`).
- **`is_active`-Spalte bleibt** → leichter technischer Debt. → Mitigation: TODO-Kommentar in Migration, kann in einer späteren Migration entfernt werden.

## Migration Plan

1. **Migration `02X_duty_templates_multi.up.sql`**:
   - `ALTER TABLE game_templates ADD COLUMN template_type TEXT NOT NULL DEFAULT 'generisch'`
   - `UPDATE game_templates SET template_type = 'generisch' WHERE is_active = 1`
   - `ALTER TABLE game_templates ADD CONSTRAINT ... CHECK (template_type IN ('heim','auswärts','generisch'))` — SQLite: via neue Tabelle + INSERT SELECT + DROP + RENAME
   - Nummer wählen: nächste freie nach 020

2. **Backend**: `internal/games/handler.go` — alle Endpunkte auf `/api/admin/duty-templates` umstellen, Slot-Suche anpassen

3. **Frontend**: Neue Pages anlegen, Route in `App.tsx` aktualisieren, Nav-Eintrag in `AppShell.tsx` anpassen

4. **Deploy**: `make deploy` (build + migrate up + restart)

**Rollback**: `migrate down` entfernt die neue Spalte; alte Frontend-Version neu deployen.

## Open Questions

- Soll die UI eine Warnung zeigen wenn zwei Vorlagen den gleichen `template_type` haben?
- Sollen Vorlagen kopierbar sein (z. B. „Duplizieren"-Button)?
