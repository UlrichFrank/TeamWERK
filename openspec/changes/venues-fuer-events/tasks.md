## 1. Datenbank-Migration

- [x] 1.1 Migration `024_venues.up.sql` anlegen: venues-Tabelle + venue_id FK auf games, training_series, training_sessions; location TEXT aus training_series/training_sessions entfernen
- [x] 1.2 Migration `024_venues.down.sql` anlegen: venues-Tabelle droppen, venue_id FKs entfernen, location TEXT wiederherstellen

## 2. Backend — Venues Package

- [x] 2.1 `internal/venues/handler.go` anlegen: Handler-Struct mit db, NewHandler, Routen-Methoden
- [x] 2.2 `GET /api/admin/venues` implementieren: alle Venues zurückgeben (id, name, street, city, postal_code, country, note, is_home_venue)
- [x] 2.3 `POST /api/admin/venues` implementieren: Validierung (name, street, city, postal_code Pflicht), Heimhallen-Uniqueness-Logik (UPDATE SET is_home_venue=0 WHERE is_home_venue=1 vor Insert wenn is_home_venue=true)
- [x] 2.4 `PUT /api/admin/venues/{id}` implementieren: Update inkl. Heimhallen-Uniqueness-Logik
- [x] 2.5 `DELETE /api/admin/venues/{id}` implementieren: Venue löschen (ON DELETE SET NULL übernimmt FK-Cleanup)

## 3. Backend — Games Integration

- [x] 3.1 `internal/games/handler.go`: venue_id in INSERT/UPDATE für Spiel-Anlage/-Bearbeitung ergänzen
- [x] 3.2 Games-SELECT-Queries erweitern: JOIN auf venues, venue-Objekt in Response einbetten (null wenn kein Venue)
- [x] 3.3 Heimhallen-Autofill-Endpunkt oder Venue-in-Response liefern: GET /api/admin/venues gibt is_home_venue mit, Frontend nutzt das für Autofill

## 4. Backend — Trainings Integration

- [x] 4.1 `internal/trainings/handler.go`: location TEXT durch venue_id ersetzen in training_series INSERT/UPDATE
- [x] 4.2 `internal/trainings/handler.go`: location TEXT durch venue_id ersetzen in training_sessions INSERT/UPDATE
- [x] 4.3 Training-SELECT-Queries erweitern: JOIN auf venues, venue-Objekt in Series- und Session-Responses einbetten

## 5. Backend — Router

- [x] 5.1 `cmd/teamwerk/main.go`: venues.NewHandler instanziieren, Routen registrieren (GET admin+trainer, POST/PUT/DELETE admin only)

## 6. Frontend — Shared Komponenten

- [x] 6.1 `web/src/components/VenuePicker.tsx` anlegen: Dropdown mit clientseitiger Textsuche, lädt Venues via GET /api/admin/venues, zeigt Namen + Stadt
- [x] 6.2 VenuePicker: „+ Neuen Ort anlegen"-Option ergänzen, die inline Modal öffnet
- [x] 6.3 VenuePicker: Modal-Formular (name, street, city, postal_code, note), POST /api/admin/venues, nach Erfolg neuen Venue direkt auswählen
- [x] 6.4 VenuePicker: Auswahl-löschen-Button (× oder leere Option)
- [x] 6.5 `web/src/components/MapsLink.tsx` anlegen: Icon-Link der `https://maps.google.com/?q=…` in neuem Tab öffnet; rendert nichts wenn venue null

## 7. Frontend — Admin Venues Seite

- [x] 7.1 `web/src/pages/AdminVenuesPage.tsx` anlegen: Tabelle aller Venues mit Name, Adresse, Heimhallen-Badge
- [x] 7.2 AdminVenuesPage: „+ Neuer Ort"-Button, Formular-Modal (anlegen)
- [x] 7.3 AdminVenuesPage: Edit-Modal pro Venue (bearbeiten)
- [x] 7.4 AdminVenuesPage: Löschen-Button mit Bestätigung
- [x] 7.5 AdminVenuesPage: Heimhallen-Toggle (Radio-Semantik, nur einer aktiv)
- [x] 7.6 Mobile Card-Layout für AdminVenuesPage (< 640px)

## 8. Frontend — Games Integration

- [x] 8.1 `web/src/pages/AdminGamesPage.tsx`: VenuePicker in Spiel-Anlage/Bearbeitungs-Formular einbauen
- [x] 8.2 AdminGamesPage: Heimhallen-Autofill bei Spieltyp „Heim" (is_home_venue-Venue aus der geladenen Venues-Liste setzen)
- [x] 8.3 AdminGamesPage: MapsLink in Spielplan-Tabelle/Detail anzeigen

## 9. Frontend — Trainings Integration

- [x] 9.1 `web/src/pages/TrainingsPage.tsx`: VenuePicker in Training-Series-Formular einbauen (ersetzt location-Textfeld)
- [x] 9.2 TrainingsPage: VenuePicker in Training-Session-Formular einbauen, Series-Venue als Default vorausfüllen
- [x] 9.3 TrainingsPage: MapsLink in Training-Session-Ansicht anzeigen

## 10. Frontend — Routing & Navigation

- [x] 10.1 `web/src/App.tsx`: Route `/admin/veranstaltungsorte` → AdminVenuesPage ergänzen
- [x] 10.2 `web/src/components/AppShell.tsx`: Nav-Eintrag „Veranstaltungsorte" unter Admin-Bereich (admin only)
