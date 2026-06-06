## Why

Alle Event-Typen (Heim-/Auswärtsspiele, generische Events, Trainings) haben derzeit keinen strukturierten Ort — Trainings speichern einen Freitext, Spiele gar nichts. Nutzer können weder zur Veranstaltungshalle navigieren noch die Adresse nachschlagen, ohne die App zu verlassen und separat zu suchen.

## What Changes

- Neue `venues`-Tabelle mit strukturierter Postadresse (Name, Straße, Stadt, PLZ, Land, Notiz) und `is_home_venue`-Flag
- `games` erhält `venue_id FK` (nullable); Heimspiele werden automatisch mit der Heimhalle vorausgefüllt
- `training_series.location TEXT` und `training_sessions.location TEXT` werden durch `venue_id FK` ersetzt
- Neue Admin-API `/api/admin/venues` (CRUD)
- Venue-Daten werden in Game- und Training-Responses eingebettet
- Maps-Deep-Link (`https://maps.google.com/?q=…`) überall wo ein Venue angezeigt wird
- Hybrid-Picker: Dropdown mit Suche + inline „Neu anlegen"-Modal (kein Kontextwechsel)
- Neue Admin-Seite `/admin/veranstaltungsorte` für Venue-Verwaltung

## Capabilities

### New Capabilities

- `venue-management`: CRUD für Veranstaltungsorte inkl. is_home_venue-Flag und Admin-Verwaltungsseite
- `venue-picker`: Wiederverwendbare Picker-Komponente mit Inline-Anlage für alle Event-Formulare
- `maps-navigation`: Maps-Deep-Link-Anzeige überall wo ein Venue dargestellt wird

### Modified Capabilities

- `games`: Spiele erhalten ein optionales Venue-Feld; Heimspiele werden automatisch vorausgefüllt
- `trainings`: Training-Series und -Sessions erhalten venue_id statt location-Freitext

## Impact

**Backend:**
- Neue Migration `024_venues` (venues-Tabelle + FKs auf games, training_series, training_sessions)
- Neues Package `internal/venues/` mit Handler
- `internal/games/handler.go` — venue_id in Queries, Autofill-Logik für Heimspiele
- `internal/trainings/handler.go` — venue_id statt location TEXT
- `cmd/teamwerk/main.go` — Router-Registrierung

**Frontend:**
- `web/src/components/VenuePicker.tsx` (neu)
- `web/src/components/MapsLink.tsx` (neu)
- `web/src/pages/AdminVenuesPage.tsx` (neu)
- `web/src/pages/AdminGamesPage.tsx` — VenuePicker einbauen
- `web/src/pages/TrainingsPage.tsx` — VenuePicker einbauen
- `web/src/App.tsx`, `web/src/components/AppShell.tsx` — Route + Nav

**Datenbank:** Migration entfernt `location TEXT` aus training_series/training_sessions (Breaking für bestehende Einträge — Daten werden verworfen, da Freitext-Feld kaum befüllt)
