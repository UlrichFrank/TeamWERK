# Tasks: Dienst-Zielgruppe

## Backend

- [x] **DB-Migration 006**: `audience TEXT CHECK(...)` zu `duty_types`, `game_template_items`, `duty_slots` hinzufügen (`internal/db/migrations/006_duty_audience.up.sql` + `.down.sql`)

- [x] **DutyType CRUD**: `audience`-Feld in `GET /api/admin/duty-types` Response und `POST/PUT /api/admin/duty-types` Request ergänzen (`internal/duties/`)

- [x] **TemplateItem CRUD**: `audience`-Feld in `GET/PUT` der Template-Items ergänzen (`internal/games/` oder `internal/duties/`)

- [x] **Slot CRUD**: `audience`-Feld in `POST /api/duty-slots` und `PUT /api/duty-slots/:id` ergänzen

- [x] **Slot-Generierung**: Bei `POST /admin/games/:id/regenerate` `audience` aus `game_template_items` in erzeugte `duty_slots` übernehmen

- [x] **Board-Query-Filter**: In `GET /api/duty-board` Bypass-Check (admin / vorstand / vorstand_beisitzer / trainer via `member_club_functions`) und audience-Filter (`COALESCE(ds.audience, dt.audience)`) einbauen; `audience`-Feld im Response zurückgeben

## Frontend

- [x] **AdminDutyTypesPage**: Zielgruppe-Select in `DutyTypeForm` ergänzen; `EditState` um `audience` erweitern; POST/PUT übertragen

- [x] **AdminDutyTemplateDetailPage** (Template-Item-Modal): Zielgruppe-Select im Edit-Modal für Template-Items; NULL zeigt „(vom Diensttyp)"

- [x] **SpieltagDetailPage** (Slot-Edit-Modal): Zielgruppe-Select beim Bearbeiten eines Slots; NULL zeigt „(von Vorlage/Diensttyp)"

- [x] **DutySlotList**: `audience`-Feld in `BoardSlot`-Interface ergänzen; Badge in der dritten Spalte anzeigen wenn `s.audience` gesetzt; `AUDIENCE_LABELS`-Map anlegen (z.B. in `lib/constants.ts`)
