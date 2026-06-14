## 1. Datenbank-Migration

- [x] 1.1 Migration `0NN_maps-provider.up.sql` anlegen: `ALTER TABLE users ADD COLUMN maps_provider TEXT NOT NULL DEFAULT 'auto' CHECK(maps_provider IN ('auto','google','apple'))`
- [x] 1.2 Migration `0NN_maps-provider.down.sql` anlegen: `ALTER TABLE users DROP COLUMN maps_provider`
- [x] 1.3 `make migrate-up` lokal ausführen und prüfen, dass die Spalte angelegt ist

## 2. Backend — Profil-API erweitern

- [x] 2.1 Profil-Response-Struct um `MapsProvider string` ergänzen (in `GET /api/profile/me` Handler)
- [x] 2.2 DB-Scan in `getProfile` um `maps_provider` erweitern
- [x] 2.3 `PUT /api/profile/me` Handler: `maps_provider` aus Request-Body lesen, validieren (`auto`/`google`/`apple`) und in DB schreiben
- [x] 2.4 Manuell testen: `GET /api/profile/me` gibt `maps_provider` zurück; `PUT` mit `apple` speichert korrekt; `PUT` mit ungültigem Wert liefert 400

## 3. Frontend — AuthContext erweitern

- [x] 3.1 `AuthContext.tsx`: Typ `User` um `mapsProvider: 'auto' | 'google' | 'apple'` erweitern (Default `'auto'`)
- [x] 3.2 Nach Login und Token-Refresh `GET /api/profile/me` aufrufen und `mapsProvider` im Context-State setzen
- [x] 3.3 Fehlerfall abfangen: bei fehlgeschlagenem Profil-Fetch bleibt `mapsProvider` auf `'auto'`

## 4. Frontend — MapsLink.tsx anpassen

- [x] 4.1 `MapsLink.tsx`: `mapsProvider` aus AuthContext lesen (`useAuth()`)
- [x] 4.2 URL-Logik implementieren: `'google'` → `maps.google.com`, `'apple'` → `maps.apple.com`, `'auto'` → User-Agent-Check (`/iPhone|iPad|iPod|Macintosh/.test(navigator.userAgent)`)

## 5. Frontend — ProfileMiscTab.tsx

- [x] 5.1 Select-Element mit drei Optionen anlegen: „Automatisch (Gerät erkennen)", „Google Maps", „Apple Maps"
- [x] 5.2 Aktuellen Wert aus Profil-API laden und Select vorbelegen
- [x] 5.3 Bei Änderung `PUT /api/profile/me` aufrufen und Feedback (Erfolg/Fehler) anzeigen
- [x] 5.4 UI im Browser testen: Select zeigt korrekten Initialwert; Speichern persistiert; nach Reload ist Wert korrekt
