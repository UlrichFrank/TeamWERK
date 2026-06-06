## Context

TeamWERK verwaltet Spiele (Heim, Auswärts, Generisch) und Trainings (Series + Sessions). Bisher haben Spiele kein Ortsfeld; Trainings speichern einen unstrukturierten Freitext (`location TEXT`). Nutzer können nicht direkt zur Veranstaltungsstätte navigieren.

Constraints: SQLite auf VPS (1 GB RAM), kein externes Geocoding-Service, Pure-Go-Build ohne CGo.

## Goals / Non-Goals

**Goals:**
- Strukturierte Postadresse für alle Events (Spiele + Trainings)
- Wiederverwendbare Venues — gleiche Halle einmal anlegen, für viele Events nutzen
- Maps-Navigation per Deep-Link direkt aus der App
- Heimhalle als Default für neue Heimspiele

**Non-Goals:**
- Kein Geocoding / kein Speichern von lat/lng — Maps-App löst Adresse selbst auf
- Keine eingebettete Kartenansicht
- Keine automatische Venue-Suche via externe API (z.B. Google Places)
- Keine Offline-Kartenfunktion

## Decisions

### 1. Venues als eigene Tabelle (nicht inline)

Dieselbe Halle taucht bei einem Verein dutzende Male pro Saison auf. Eine shared `venues`-Tabelle vermeidet redundante Dateneingabe und erlaubt nachträgliche Korrekturen (z.B. Adressänderung der Heimhalle) an einer Stelle.

Alternative: Inline-Felder je Event — abgelehnt wegen Redundanz und Inkonsistenz.

### 2. Kein Geocoding, nur strukturierte Adresse

Maps-Deep-Link aus Adresse: `https://maps.google.com/?q=Straße+PLZ+Stadt` funktioniert auf allen Plattformen ohne externen API-Call. Koordinaten bringen keinen Mehrwert für den Use-Case "navigieren".

Alternative: Nominatim-Geocoding beim Speichern — abgelehnt wegen externer Abhängigkeit, Quota-Risiko und unnötiger Komplexität.

### 3. is_home_venue Flag (max. 1 pro Verein)

Ein einzelnes Boolean-Flag auf dem Venue-Datensatz. Das Backend erzwingt Eindeutigkeit: beim Setzen von `is_home_venue=true` wird das Flag aller anderen Venues auf `false` gesetzt (UPDATE before INSERT/UPDATE). Kein separater Club-FK nötig, da TeamWERK nur einen Verein verwaltet.

### 4. location TEXT in Trainings wird ersetzt

`training_series.location` und `training_sessions.location` werden durch `venue_id FK` ersetzt. Kein Dual-Mechanismus (FK + Freitext), da das Freitext-Feld kaum befüllt ist und parallele Logik fehleranfällig wäre. Bestehende Freitext-Werte gehen bei der Migration verloren (akzeptiert).

### 5. Hybrid-Picker: Dropdown + Inline-Modal

Der VenuePicker lädt alle Venues einmalig beim Mount (`GET /api/admin/venues`). Clientseitige Filterung reicht (Venues-Liste bleibt klein, < 100 Einträge). „+ Neuen Ort anlegen" öffnet ein Modal mit minimalem Formular — kein Seitenwechsel.

### 6. Zugriffsrechte

- `GET /api/admin/venues` — admin + trainer (Lesezugriff für Picker in Trainings-Formularen)
- `POST/PUT/DELETE /api/admin/venues` — admin only
- Venue-Daten in Game/Training-Responses — alle authentifizierten Nutzer (nur Lesezugriff)

## Risks / Trade-offs

- **Datenverlust bei Migration** → `location TEXT` wird nicht migriert. Risiko gering, da Feld selten befüllt. Rollback via `.down.sql` stellt altes Schema wieder her.
- **Venue-Löschung mit Referenzen** → `ON DELETE SET NULL` auf allen FKs; Venue kann gelöscht werden, Event verliert dann nur seinen Ort. Alternative `RESTRICT` wäre sicherer aber UX-feindlicher.
- **Heimhalle-Autofill nur Frontend-seitig** → Backend liefert `is_home_venue` im Venue-Objekt; Autofill-Logik liegt im Frontend-Formular. Kein serverseitiger Zwang — akzeptiert, da es ein UX-Feature ist.

## Migration Plan

1. Migration `024_venues` anlegen:
   - `venues`-Tabelle erstellen
   - `ALTER TABLE games ADD COLUMN venue_id ...`
   - `ALTER TABLE training_series DROP COLUMN location` + `ADD COLUMN venue_id ...`
   - `ALTER TABLE training_sessions DROP COLUMN location` + `ADD COLUMN venue_id ...`
2. Backend deployen (Binary hat Migration eingebettet, `make deploy` führt `migrate up` automatisch aus)
3. Heimhalle manuell unter `/admin/veranstaltungsorte` anlegen und als Heimhalle markieren

**Rollback:** `make migrate-down` auf VPS, altes Binary deployen.

## Open Questions

- Sollen Trainer (nicht nur Admins) neue Venues anlegen dürfen? → Vorläufig: nur Admins (konsistent mit bestehenden Admin-CRUD-Patterns). Kann später per Rolle erweitert werden.
