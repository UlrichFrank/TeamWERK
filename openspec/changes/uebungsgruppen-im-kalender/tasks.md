## 1. Migration

- [x] 1.1 `internal/db/migrations/067_calendar_token_practice_groups.up.sql`: `ALTER TABLE calendar_tokens ADD COLUMN include_practice_groups INTEGER NOT NULL DEFAULT 1`
- [x] 1.2 `.down.sql` mit passendem Rückbau (SQLite: `ALTER TABLE … DROP COLUMN`)

## 2. Feed-Backend (`internal/calendar/handler.go`)

- [x] 2.1 `tokenSettings` um `IncludePracticeGroups` (`include_practice_groups`) erweitern; alle vier Token-Routen (`GetToken`, `UpsertToken`, `GetChildToken`, `UpsertChildToken`) lesen/schreiben die Spalte
- [x] 2.2 `fetchTrainings` auf `JOIN kader k ON k.id = ts.kader_id` + `LEFT JOIN teams t ON t.id = ts.team_id` umstellen; Label `COALESCE(t.name, k.name, '')`
- [x] 2.3 `fetchTrainings` nimmt die beiden Toggles entgegen und filtert: nur `include_training` → `ts.team_id IS NOT NULL`, nur `include_practice_groups` → `ts.team_id IS NULL`, beide → kein Zusatzfilter
- [x] 2.4 `Feed` ruft `fetchTrainings` nur auf, wenn mindestens einer der beiden Toggles gesetzt ist

## 3. Feed-Tests (`internal/calendar/`)

- [x] 3.1 `uebungsgruppen_test.go`: `TestIcalFeed_OhneUebungsgruppe` → `TestIcalFeed_MitUebungsgruppe` (Termin **ist** im Feed, Mannschaftstermin als Gegenbeleg bleibt)
- [x] 3.2 Test: `include_practice_groups=false` filtert den Übungsgruppen-Termin heraus, Mannschaftstermin bleibt
- [x] 3.3 Test: `include_training=false` + `include_practice_groups=true` → nur der Übungsgruppen-Termin
- [x] 3.4 Test: `SUMMARY` trägt `Training: <kader.name>` (nicht den `training_sessions.title`)
- [x] 3.5 `allTogglesOn()` in `handler_test.go` um das sechste Feld erweitern
- [x] 3.6 Test: Token-Roundtrip (`POST` → `GET`) führt `include_practice_groups`

## 4. Frontend Kalender-Abo

- [x] 4.1 `web/src/components/profile/ProfileKalenderTab.tsx`: `Toggles`-Typ, `ALL_ON`, `labels` (`include_practice_groups: 'Übungsgruppen'`) und das Mapping im `useEffect` erweitern
- [x] 4.2 Prüfen, dass `ChildProfilePage.tsx` über denselben `apiPath`-Weg mitzieht (keine eigene Toggle-Liste)

## 5. `/termine`-Ladefenster

- [x] 5.1 `web/src/pages/TerminePage.tsx`: obere Fenstergrenze = späteres von `season.end_date` und `isoDaysFromNow(365)`; Kommentar mit Begründung (Termine jenseits des Saisonendes)
- [x] 5.2 Helper `web/src/lib/terminWindow.ts` (`terminLoadWindow`) — beide Grenzen an einer Stelle, testbar über injizierbares `now`
- [x] 5.3 Vitest: Saisonende vor/nach dem rollierenden Jahr → jeweils die richtige Grenze

## 6. Zeilenlimit

- [x] 6.1 `internal/trainings/handler.go` `ListSessions`: `httpx.Paging(r, 100, 1000)`
- [x] 6.2 Go-Test: mehr als 200 sichtbare Termine, `limit=500` → alle in `items`, `total` stimmt (der Deckel selbst ist in `internal/httpx` unit-getestet)

## 7. Abschluss

- [x] 7.1 `make test` + `pnpm -C web test` + `make lint` grün
- [ ] 7.2 `/verify-change` durchlaufen (Route→Tests, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`)
- [x] 7.3 CHANGELOG-Einträge (`[feat] calendar: …`, `[fix] termine: …`)

## 8. Dokumentation

- [x] 8.1 Gotcha „Übungsgruppen" in `docs/agent/06-gotchas.md`: iCal-Feed aus der Liste der strukturell ausgeschlossenen Flächen nehmen, Weg heraus (Anker `kader_id` + eigener Schalter) festhalten
