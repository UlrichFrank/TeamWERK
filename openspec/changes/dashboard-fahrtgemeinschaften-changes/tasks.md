## 1. Datenbank

- [ ] 1.1 Migration `018_carpooling_events.up.sql` erstellen: Tabelle `carpooling_events (id INTEGER PK AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, type TEXT NOT NULL CHECK(type IN ('biete_created','suche_created','pairing_requested','pairing_confirmed','pairing_rejected','pairing_cancelled','biete_deleted','suche_deleted')), actor_name TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`
- [ ] 1.2 Migration `018_carpooling_events.down.sql` erstellen: `DROP TABLE IF EXISTS carpooling_events`

## 2. Backend: Upsert-Handler erweitern

- [ ] 2.1 Nach erfolgreichem Insert eines `biete`-Eintrags: alle User-IDs mit `suche`-Eintrag für dasselbe Spiel abfragen (exkl. aktueller User) und für jeden einen `carpooling_events`-Eintrag (`type='biete_created'`) schreiben
- [ ] 2.2 Nach erfolgreichem Insert eines `suche`-Eintrags: alle User-IDs mit `biete`-Eintrag für dasselbe Spiel abfragen und für jeden einen `carpooling_events`-Eintrag (`type='suche_created'`) schreiben

## 3. Backend: Delete-Handler erweitern

- [ ] 3.1 Delete-Handler auf explizite Transaktion (`db.BeginTx`) umstellen
- [ ] 3.2 Vor dem DELETE bei `typ='biete'`: User-IDs mit `pending`/`confirmed` Paarung ermitteln und für jeden `carpooling_events`-Eintrag (`type='biete_deleted'`) in Transaktion schreiben
- [ ] 3.3 Vor dem DELETE bei `typ='suche'`: Biete-User mit `pending`/`confirmed` Paarung ermitteln und ggf. Event schreiben (`type='suche_deleted'`)
- [ ] 3.4 Transaktion committen; bei Fehler rollback und HTTP 500

## 4. Backend: Paarungshandler erweitern

- [ ] 4.1 `RequestPairing`: nach erfolgreichem INSERT `carpooling_events`-Eintrag (`type='pairing_requested'`) für `oppositeUserID` schreiben (game_id aus biete-Eintrag laden)
- [ ] 4.2 `ConfirmPairing`: nach erfolgreichem UPDATE Event (`type='pairing_confirmed'`) für `initiatorUserID` schreiben
- [ ] 4.3 `RejectPairing`: nach erfolgreichem UPDATE Event schreiben — `type='pairing_rejected'` wenn vorheriger Status `pending` war, `type='pairing_cancelled'` wenn `confirmed` war; Empfänger jeweils `oppositeUserID`

## 5. Backend: Dashboard-Handler erweitern

- [ ] 5.1 `CarpoolingHint`-Struct erweitern: `MyEntry *MyEntryInfo` (`ID int`, `Typ string`), `Paarungen []PaarungInfo` (Name der Gegenseite, PaarungID), `RecentEvents []CarpoolingEvent` (`Type string`, `ActorName string`, `CreatedAt string`)
- [ ] 5.2 In `queryCarpoolingHint`: eigenen Eintrag des Users abfragen (`SELECT id, typ FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ?`)
- [ ] 5.3 Bestätigte Paarungen laden: `SELECT p.id, u.name FROM mitfahrt_paarungen p JOIN mitfahrgelegenheiten m ON ... JOIN users u ON ... WHERE status='confirmed' AND (biete_user_id=? OR suche_user_id=?)` für das nächste Spiel
- [ ] 5.4 Recent Events laden: `SELECT type, actor_name, created_at FROM carpooling_events WHERE user_id = ? AND game_id = ? AND created_at >= datetime('now','-48 hours') ORDER BY created_at DESC`
- [ ] 5.5 Leere Slices als `[]` (nicht `null`) serialisieren (JSON-Tags + Initialisierung mit `make([]..., 0)`)

## 6. Frontend: Interfaces und CarpoolingHintCard

- [ ] 6.1 In `DashboardPage.tsx` Interface `CarpoolingHint` erweitern: `myEntry: { id: number; typ: string } | null`, `paarungen: { paarungId: number; partnerName: string }[]`, `recentEvents: { type: string; actorName: string; createdAt: string }[]`
- [ ] 6.2 `CarpoolingHintCard` erweitern: eigenen Status anzeigen (Mein Eintrag: Suche / Biete / kein Eintrag)
- [ ] 6.3 Bestätigte Paarungen rendern: je Paarung eine Zeile mit `<Check>`-Icon und Partnername
- [ ] 6.4 `recentEvents` rendern: Icon und Text je Event-Typ (`biete_created` → „… bietet Mitfahrt an", `pairing_confirmed` → „… hat Mitfahrt bestätigt", `biete_deleted` → „… hat Angebot zurückgezogen", etc.) mit relativem Zeitstempel (`heute`, `gestern`, Wochentag)
- [ ] 6.5 Gesamtzähler (Angebote / Gesuche) und Link „Zur Übersicht →" beibehalten

## 7. Frontend: SSE Live-Update

- [ ] 7.1 In `DashboardPage.tsx` `useLiveUpdates` aus `../hooks/useLiveUpdates` importieren und nach dem initialen Load im `useEffect`-Block einbinden: bei Event `"mitfahrgelegenheiten"` → `loadDashboard(true)` (silent reload, kein Spinner)
