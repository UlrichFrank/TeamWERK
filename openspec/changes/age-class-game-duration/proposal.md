## Why

Die Spieldauer unterscheidet sich je Altersklasse (A–D Jugend) — aktuell ist sie nur global pro Spielplan-Vorlage konfigurierbar, was manuelle Anpassung aller Vorlagen erfordert. Da `teams` bereits ein `age_class`-Feld hat, können wir Spieldauer und Pausenzeit automatisch aus der Altersklasse ableiten.

## What Changes

- Neue Tabelle `age_class_game_rules` speichert Halbzeit-Dauer und Pausenzeit je Altersklasse
- Neue Admin-Seite „Jugend" (unter Kaderplanung oder Admin) zum Bearbeiten dieser Werte
- Standardwerte bei Erstanlage: A=30/15, B=25/10, C=25/10, D=20/10 (Minuten Halbzeit / Pause)
- Spielplan-Vorlagen-Erstellung (`POST /api/admin/games`) kann die Regel der Team-Altersklasse zur Slot-Zeitberechnung heranziehen

## Capabilities

### New Capabilities

- `age-class-game-rules`: Verwaltung (CRUD) von Spieldauer und Pausen-Konfiguration pro Altersklasse (A–D Jugend) im Admin-Bereich

### Modified Capabilities

- `games`: Slot-Generierung nutzt künftig `age_class_game_rules` der zugehörigen Mannschaft, wenn kein `game_duration_minutes` in der Vorlage gesetzt ist

## Impact

- **DB:** neue Migration `age_class_game_rules`-Tabelle mit Seed-Daten
- **Backend:** neues Package oder Erweiterung `internal/config` — Handler für GET/PUT der Altersklassen-Regeln
- **Frontend:** neue Seite `web/src/pages/AdminAgeClassRulesPage.tsx`, Nav-Eintrag in AppShell unter Admin
- **Spielplan-Logik:** `internal/games` — Slot-Zeitberechnung liest ggf. `age_class_game_rules`
