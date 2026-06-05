# Design: Dienst-Zielgruppe

## Datenbankschema

### Migration 006 (nächste freie Nummer nach 005)

```sql
-- 006_duty_audience.up.sql
ALTER TABLE duty_types ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
ALTER TABLE game_template_items ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
ALTER TABLE duty_slots ADD COLUMN audience TEXT CHECK(audience IN ('spieler','trainer','vorstand','vorstand_beisitzer','eltern'));
```

```sql
-- 006_duty_audience.down.sql
-- SQLite unterstützt kein DROP COLUMN vor 3.35, aber modernc.org/sqlite unterstützt es
ALTER TABLE duty_types DROP COLUMN audience;
ALTER TABLE game_template_items DROP COLUMN audience;
ALTER TABLE duty_slots DROP COLUMN audience;
```

### Semantik

- NULL = keine Einschränkung (Standard, Abwärtskompatibilität gewährleistet)
- Wert = nur Nutzer mit dieser Vereinsfunktion (oder Eltern-Status) sehen den Slot

## Kaskadenlogik

```
duty_types.audience
        │
        ▼  (bei Spielplan-Generierung: NULL = übernehmen)
game_template_items.audience
        │
        ▼  (bei Slot-Erzeugung: NULL = übernehmen)
duty_slots.audience
        │
        ▼  (Board-Query: COALESCE(ds.audience, dt.audience))
```

Bei der Slot-Generierung aus einem Template (`POST /admin/games/:id/regenerate`) wird `audience` aus `game_template_items.audience` direkt in `duty_slots.audience` übernommen.

Im Board-Query wird `COALESCE(ds.audience, dt.audience)` ausgewertet, falls ein Slot kein eigenes audience gesetzt hat. Das ermöglicht nachträgliche Änderungen am Diensttyp.

## Board-Query-Filter

```sql
-- Effektive Audience pro Slot
COALESCE(ds.audience, dt.audience) AS effective_audience

-- Filter-Bedingung (wird nur angewendet wenn kein Bypass)
AND (
  effective_audience IS NULL
  OR (effective_audience = 'eltern' AND EXISTS (
    SELECT 1 FROM family_links fl WHERE fl.parent_user_id = :user_id
  ))
  OR EXISTS (
    SELECT 1 FROM member_club_functions mcf
    JOIN members m ON m.id = mcf.member_id
    WHERE m.user_id = :user_id AND mcf.function = effective_audience
  )
)
```

## Bypass-Regeln

Folgende Nutzer sehen **immer alle** Dienste ihres Teams, unabhängig von der Zielgruppe:

| Bedingung | Basis |
|-----------|-------|
| System-Rolle `admin` | `users.role = 'admin'` |
| Vereinsfunktion `vorstand` | `member_club_functions.function = 'vorstand'` |
| Vereinsfunktion `vorstand_beisitzer` | `member_club_functions.function = 'vorstand_beisitzer'` |
| Vereinsfunktion `trainer` | `member_club_functions.function = 'trainer'` |

Bypass wird im Go-Handler ermittelt: wenn der Nutzer admin ist ODER eine der drei Vereinsfunktionen hat, wird die audience-Filterbedingung weggelassen.

## UI-Änderungen

### Zielgruppen-Auswahl (Shared)

Dropdown mit Optionen:
- `""` → „Keine Einschränkung" (Standard)
- `spieler` → „Spieler"
- `trainer` → „Trainer"
- `vorstand` → „Vorstand"
- `vorstand_beisitzer` → „Vorstands-Beisitzer"
- `eltern` → „Eltern"

### AdminDutyTypesPage (`/admin/diensttypen`)

In `DutyTypeForm`: neues Feld „Zielgruppe" als Select mit obigen Optionen. Wird in `EditState` als `audience: string` geführt (leer = NULL). Übertragen via `audience: state.audience || null` im POST/PUT.

### AdminDutyTemplateDetailPage (`/admin/dienstplan-vorlagen/:id`)

Beim Bearbeiten eines Template-Items: Zielgruppe-Feld im Edit-Modal. Zeigt aktuellen Wert oder „(vom Diensttyp)" als Placeholder wenn NULL.

### SpieltagDetailPage (`/kalender/:id`) — Dienst bearbeiten

Im Edit-Modal eines Slots: Zielgruppe-Feld. Zeigt aktuellen Slot-Wert oder „(von Vorlage/Diensttyp)" als Placeholder-Text wenn NULL.

### DutySlotList — Badge

In der dritten Spalte (neben `{s.vacancies} frei` und assignees): wenn `s.audience` gesetzt, kleines Badge z.B.:

```tsx
{s.audience && (
  <span className="text-xs bg-brand-info/10 text-brand-text px-1.5 py-0.5 rounded">
    {AUDIENCE_LABELS[s.audience]}
  </span>
)}
```

## API-Änderungen

### GET /api/duty-board

Response-Felder der Slots um `audience` ergänzen (kann NULL sein). Filter-Logik im Handler.

### POST/PUT /api/duty-slots, /api/admin/duty-types, /api/admin/game-template-items

`audience` als optionaler Parameter akzeptieren und speichern.

### GET /api/admin/duty-types

`audience` im Response zurückgeben.
