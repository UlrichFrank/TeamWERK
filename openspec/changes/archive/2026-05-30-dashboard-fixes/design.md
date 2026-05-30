## Context

Das Dashboard (`DashboardPage.tsx`) und der `/api/dashboard`-Endpoint sind fertig implementiert (Change `dashboard-home`). Dieser Change baut darauf auf ohne Architektur-Umbrüche. Drei voneinander unabhängige Bereiche werden angepasst:

1. **Frontend-Umbenennung** (trivial)
2. **URL-gesteuerte Kalendernavigation** (Frontend-only, kein API-Change)
3. **Dienstkonto-Formel** (Backend-Logik + neue DB-Spalte + Admin-UI)

## Goals / Non-Goals

**Goals:**
- Konsistente Begriffe: „Events" statt „Spiele" im Dashboard
- Direkter Sprung zum richtigen Kalendermonat per Deep-Link
- Faire, saisonbasierte Soll-Berechnung für Elternteile
- `games_per_season` pro Kader durch Admin/Vorstand konfigurierbar

**Non-Goals:**
- Kalender-Highlighting eines spezifischen Tages (reicht: richtiger Monat)
- Automatische Befüllung von `games_per_season` aus dem Spielplan
- Soll-Berechnung für Trainer/Admin/Spieler (bleibt `null`)

## Decisions

### D1: Kalender-URL-Param statt Route-Param

`/kalender?date=2026-06-14` statt `/kalender/2026-06` oder `/kalender/2026/6`.

**Rationale:** KalenderPage bleibt eine einzelne Route ohne verschachtelte Segmente. Query-Params sind optional — ein direkter Aufruf ohne `?date` funktioniert weiterhin mit `new Date()`. Route-Params hätten eine Änderung an App.tsx und AppShell-Links erfordert.

### D2: `games_per_season` auf `kader`, nicht auf `teams`

`kader` ist bereits saison-spezifisch (`season_id`-FK). Das Feld auf `teams` wäre saison-agnostisch und müsste bei jeder neuen Saison händisch überschrieben werden.

### D3: Formel-Berechnung im Backend (`/api/dashboard`), nicht im Frontend

Die benötigten Joins (kader_members, game_templates, family_links) sind im Backend einfacher und sicherer. Das Frontend empfängt nur den fertig berechneten `soll`-Wert. Der Datenschutz (kein Elternteil sieht den anderen) ist serverseitig automatisch gewährleistet.

### D4: Formel-Details

```
Für jeden member_id aus family_links WHERE parent_user_id = current_user:
  kader_id = kader_members.kader_id WHERE member_id = member_id AND kader.season_id = active_season
  IF kein kader_id → Kind nicht in aktivem Kader, überspringen

  heim_slots    = SUM(gti.slots_count) WHERE gt.template_type='heim' AND gt.is_active=1
  auswärts_slots = SUM(gti.slots_count) WHERE gt.template_type='auswärts' AND gt.is_active=1
  avg_per_game  = float64(heim_slots + auswärts_slots) / 2.0
  player_count  = COUNT(kader_members) WHERE kader_id = kader_id
  parent_count  = COUNT(family_links) WHERE member_id = member_id  ← 1 oder 2
  child_soll    = (kader.games_per_season * avg_per_game) / float64(player_count) / float64(parent_count)

soll = int(math.Round(sum of all child_soll))
```

Edge Cases:
- `player_count = 0` → Division durch 0 vermeiden, Kind überspringen
- `games_per_season = 0` → `soll = 0` (korrekt, kein Fehler)
- Keine aktive Template für heim oder auswärts → avg = 0, soll = 0
- Kind in keinem Kader → überspringen

### D5: API-Endpunkt für `games_per_season`

Kein neuer Endpoint nötig. Das bestehende `PUT /api/admin/kader/{id}` (falls vorhanden) wird erweitert, oder ein neuer `PATCH /api/admin/kader/{id}/games-per-season` wird angelegt — minimal, da nur ein Feld.

## Risks / Trade-offs

- **Template nicht konfiguriert** → `avg_per_game = 0`, `soll = 0`. Elternteil sieht „Ziel: 0 Dienste". Mitigation: Erklärtext im UI falls `soll = 0` auf Konfigurationslücke hinweisen.
- **KalenderPage URL-Param**: Wenn `?date` ein ungültiges Format hat, fällt die Page auf `new Date()` zurück — kein Fehler, nur falsches Monat. Mitigation: Validierung mit `isNaN(date)`.
- **`games_per_season` vergessen zu setzen**: Elternteile sehen `soll = 0`. Mitigation: AdminKaderPage kann einen leeren Wert visuell hervorheben.

## Migration Plan

1. Migration 012 deployen (`ALTER TABLE kader ADD COLUMN games_per_season INTEGER NOT NULL DEFAULT 0`)
2. Backend-Binary deployen (neue Formel greift sofort, liefert `soll = 0` bis Admin Wert einträgt)
3. Admin trägt `games_per_season` für jeden aktiven Kader ein
4. Frontend-Deploy (Umbenennung + URL-Param + Erklärtext)

Rollback: Migration 012 down löscht die Spalte; alte Binary kennt das Feld nicht.
