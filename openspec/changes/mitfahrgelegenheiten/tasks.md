## 1. DB Migration

- [ ] 1.1 `internal/db/migrations/013_mitfahrgelegenheiten.up.sql`: Tabelle `mitfahrgelegenheiten` anlegen (id, game_id FK, user_id FK, typ CHECK('biete','suche'), plaetze, treffpunkt, notiz, created_at, updated_at, UNIQUE(game_id, user_id))
- [ ] 1.2 `internal/db/migrations/013_mitfahrgelegenheiten.down.sql`: `DROP TABLE IF EXISTS mitfahrgelegenheiten`

## 2. Backend — Package + Handler

- [ ] 2.1 Package `internal/carpooling/` anlegen: `handler.go` mit `type Handler struct{ db *sql.DB }` + `NewHandler`
- [ ] 2.2 Response-Typen definieren: `GameEntry`, `CarpoolEntry`, `CarpoolResponse`
- [ ] 2.3 `GET /api/mitfahrgelegenheiten`: Query alle Auswärtsspiele (is_home=0, date >= heute), je zwei Listen biete/suche, `is_own` Flag, Nutzername aus `users.name`
- [ ] 2.4 `POST /api/mitfahrgelegenheiten`: INSERT OR REPLACE in `mitfahrgelegenheiten` (eigener user_id wird aus JWT-Claims gesetzt, nicht aus Request-Body)
- [ ] 2.5 `DELETE /api/mitfahrgelegenheiten/{id}`: Löschen nur wenn `user_id = current_user.id`, sonst 403
- [ ] 2.6 In `main.go` routes registrieren (authenticated Middleware-Gruppe)

## 3. Backend — Dashboard-Update

- [ ] 3.1 In `internal/dashboard/handler.go`: `vehicleInfo`-Feld um `carpoolingHint`-Struct erweitern: nächstes Auswärtsspiel (id, date, opponent) + biete_count + suche_count
- [ ] 3.2 SQL-Query: nächstes Auswärtsspiel für User (via Kader/Team, is_home=0, date >= heute, LIMIT 1) + COUNT aus mitfahrgelegenheiten
- [ ] 3.3 Dashboard-Response-Typ in Go um `CarpoolingHint *CarpoolingHint` erweitern (nil wenn kein Auswärtsspiel)

## 4. Frontend — MitfahrgelegenheitenPage

- [ ] 4.1 `web/src/pages/MitfahrgelegenheitenPage.tsx` anlegen
- [ ] 4.2 Interfaces: `CarpoolEntry`, `GameCarpoolData`
- [ ] 4.3 Laden: `GET /api/mitfahrgelegenheiten` bei Mount
- [ ] 4.4 Layout: Pro Auswärtsspiel eine Card mit Datum/Gegner als Header; zwei Spalten (≥640px) oder Tabs (<640px): „Fahrangebote" | „Mitfahrgesuche"
- [ ] 4.5 Jeder Eintrag zeigt: Nutzername, Plätze (nur biete), Treffpunkt, Notiz; eigene Einträge mit Löschen-Button (Trash2)
- [ ] 4.6 CTA-Buttons: „Ich biete Mitfahrt" / „Ich suche Mitfahrt" — öffnen Modal oder Inline-Formular
- [ ] 4.7 Formular-Felder: typ (biete/suche), plaetze (nur wenn biete, min=1), treffpunkt (optional), notiz (optional)
- [ ] 4.8 Submit: `POST /api/mitfahrgelegenheiten` → Liste neu laden
- [ ] 4.9 Löschen: `DELETE /api/mitfahrgelegenheiten/{id}` → eigenen Eintrag entfernen
- [ ] 4.10 Leerzustand: „Keine Auswärtsfahrten geplant" wenn Liste leer
- [ ] 4.11 Mobile: Card-Layout, Touch-Targets min 44px, Formular als Modal

## 5. Frontend — Navigation

- [ ] 5.1 `AppShell.tsx`: Nav-Eintrag `{ to: '/mitfahrgelegenheiten', label: 'Mitfahrgelegenheiten', roles: [] }` im Dienste-Abschnitt nach `{ to: '/dienste', ... }` einfügen
- [ ] 5.2 `App.tsx`: Route `path="mitfahrgelegenheiten"` mit `<MitfahrgelegenheitenPage />` anlegen (innerhalb AppShell/Authenticated-Wrapper)

## 6. Frontend — Dashboard

- [ ] 6.1 `DashboardPage.tsx`: `VehicleSection`-Komponente in der Fahrtgemeinschaften-Sektion durch `CarpoolingHintCard` ersetzen
- [ ] 6.2 `CarpoolingHintCard`-Komponente: zeigt nächstes Auswärtsspiel (Datum + Gegner), Angebots-/Gesuch-Zähler, Link zu `/mitfahrgelegenheiten`
- [ ] 6.3 Leerzustand: „Keine Auswärtsfahrten geplant" + Link
- [ ] 6.4 `DashboardData`-Interface: `vehicleInfo` um `carpoolingHint` erweitern (optional)

## 7. Testen

- [ ] 7.1 Manuell: Eintrag als Fahrer anlegen → erscheint in Liste
- [ ] 7.2 Manuell: Doppelter Eintrag → Update statt Fehler
- [ ] 7.3 Manuell: Fremden Eintrag löschen → 403
- [ ] 7.4 Manuell: Vergangene Auswärtsspiele erscheinen nicht
- [ ] 7.5 Manuell: Dashboard-Sektion zeigt korrekten Zähler
- [ ] 7.6 Manuell: Mobile-Layout (<640px) korrekt (Tabs/Cards)
