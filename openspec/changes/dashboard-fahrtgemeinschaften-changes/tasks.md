## 1. Datenbank

- [ ] 1.1 Migration `018_carpooling_events.up.sql` erstellen: Tabelle `carpooling_events (id INTEGER PK AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, type TEXT NOT NULL CHECK(type IN ('biete_deleted','suche_deleted')), actor_name TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`
- [ ] 1.2 Migration `018_carpooling_events.down.sql` erstellen: `DROP TABLE IF EXISTS carpooling_events`

## 2. Backend: Delete-Handler erweitern

- [ ] 2.1 `Delete`-Handler in `internal/carpooling/handler.go` auf explizite Transaktion (`db.BeginTx`) umstellen
- [ ] 2.2 Vor dem DELETE: bei `typ='biete'` alle User-IDs mit `pending`/`confirmed` Paarung gegen diesen Eintrag abfragen und für jeden einen `carpooling_events`-Eintrag (`type='biete_deleted'`, `actor_name` aus JWT) in die Transaktion schreiben
- [ ] 2.3 Vor dem DELETE: bei `typ='suche'` prüfen ob eine `pending`/`confirmed` Paarung existiert; falls ja, Event für den Biete-User anlegen (`type='suche_deleted'`)
- [ ] 2.4 Transaktion committen; bei Fehler rollback und HTTP 500 zurückgeben

## 3. Backend: Dashboard-Handler erweitern

- [ ] 3.1 `CarpoolingHint`-Struct in `internal/dashboard/handler.go` um `MyEntry` (optional: `*MyEntryInfo` mit `ID int`, `Typ string`), `Paarungen []PaarungInfo` und `RecentEvents []CarpoolingEvent` erweitern
- [ ] 3.2 In `queryCarpoolingHint`: eigenen Eintrag des Users für das nächste Spiel abfragen (`SELECT id, typ FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ?`)
- [ ] 3.3 Paarungen laden: alle Paarungen wo `biete_user_id = userID` oder `suche_user_id = userID` für das Spiel; Filter: `status IN ('pending','confirmed')` ODER (`status='rejected'` AND `updated_at >= datetime('now','-48 hours')`); Gegenseiten-Namen mitladen
- [ ] 3.4 Events laden: `SELECT type, actor_name, created_at FROM carpooling_events WHERE user_id = ? AND game_id = ? AND created_at >= datetime('now','-48 hours')` ORDER BY `created_at DESC`
- [ ] 3.5 JSON-Tags für neue Felder setzen (`myEntry`, `paarungen`, `recentEvents`); leere Slices als `[]` (nicht `null`) serialisieren

## 4. Frontend: Interfaces und CarpoolingHintCard

- [ ] 4.1 In `DashboardPage.tsx` Interface `CarpoolingHint` um `myEntry: { id: number; typ: string } | null`, `paarungen: PaarungInfo[]` und `recentEvents: CarpoolingEvent[]` erweitern; neue Interfaces `PaarungInfo` und `CarpoolingEvent` anlegen
- [ ] 4.2 `CarpoolingHintCard` erweitern: eigenen Status anzeigen (Mein Eintrag: Suche / Biete / kein Eintrag)
- [ ] 4.3 Paarungsliste rendern: je Paarung eine Zeile mit Icon (✓ confirmed, ✗ rejected, ? pending), Name der Gegenseite und relativem Zeitstempel (`heute`, `gestern`, Wochentag)
- [ ] 4.4 Events aus `recentEvents` rendern: Icon ⚠ mit Text „[Name] hat Angebot/Gesuch zurückgezogen" und relativem Zeitstempel
- [ ] 4.5 Gesamtzähler (Angebote / Gesuche) und Link „Zur Übersicht →" beibehalten
