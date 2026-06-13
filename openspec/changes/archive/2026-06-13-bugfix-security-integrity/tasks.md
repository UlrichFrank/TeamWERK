## 1. Auth — Sicherheitsfixes (auth/handler.go)

- [x] 1.1 Package-Level-Dummy-Hash anlegen (`var dummyHash = []byte("$2a$10$...")`); im `ErrNoRows`-Branch von `Login` `bcrypt.CompareHashAndPassword(dummyHash, ...)` aufrufen vor HTTP 401
- [x] 1.2 `Refresh`-Handler: DELETE + INSERT in Transaktion kapseln; `accessToken, _ :=` und `plain, newHash, _ :=` durch Fehlerprüfung ersetzen; Cookie erst nach `tx.Commit` setzen
- [x] 1.3 `Register`-Handler: `hash, _ := bcrypt.GenerateFromPassword(...)` → Fehler prüfen; bei Fehler HTTP 500 zurückgeben und kein INSERT ausführen
- [x] 1.4 Test: `TestLogin_TimingAttack` — beide Login-Pfade (bekannte/unbekannte E-Mail) dauern ≥ bcrypt-Schwelle (~80ms); kein Pfad antwortet < 10ms
- [x] 1.5 Test: `TestRefreshToken_Atomic` — nach simuliertem DB-Fehler beim INSERT bleibt das alte Refresh-Token gültig (kein Permanent-Logout)
- [x] 1.6 Test: `TestRegister_BcryptError` — wenn bcrypt fehlschlägt, kein User-Record in DB, HTTP 500

## 2. SSE — Token aus URL entfernen

- [x] 2.1 `hub/handler.go`: `?token`-Query-Parameter-Auth entfernen; stattdessen HttpOnly-Cookie validieren (Cookie-Name `refresh_token` auslesen, `HashToken` → DB-Lookup für UserID/Role, analog zu `Refresh`-Handler)
- [x] 2.2 `web/src/hooks/useLiveUpdates.ts`: `const token = getAccessToken()` und `?token=...` aus der EventSource-URL entfernen; `EventSource('/api/events')` ohne Token (Cookie wird automatisch mitgeschickt)
- [x] 2.3 `web/src/hooks/useLiveUpdates.ts`: `useEffect`-Dependency-Array auf `[accessToken]` ändern; `getAccessToken` importieren oder `accessToken`-State als Parameter übergeben — bei jedem Token-Refresh baut der Effect die EventSource neu auf
- [x] 2.4 Test: `TestSSE_CookieAuth` — Verbindung mit gültigem Cookie → 200; ohne Cookie → 401; mit veraltetem `?token`-Param → 401

## 3. Frontend — Axios Refresh-Race

- [x] 3.1 `web/src/lib/api.ts`: Variable `let refreshPromise: Promise<string> | null = null` einführen; im 401-Interceptor: wenn `refreshPromise != null`, auf den laufenden Promise warten statt neuen Request starten; nach Abschluss `refreshPromise = null` setzen

## 4. Duties — Race-freier Claim/Unclaim

- [x] 4.1 `internal/duties/handler.go` `Claim`: SELECT + filled-Check entfernen; stattdessen `UPDATE duty_slots SET slots_filled = slots_filled + 1 WHERE id=? AND slots_filled < slots_total` ausführen; RowsAffected == 0 → HTTP 409; danach INSERT duty_assignment; bei UNIQUE-Fehler → `UPDATE duty_slots SET slots_filled = slots_filled - 1 WHERE id=?` und HTTP 409
- [x] 4.2 `internal/duties/handler.go` `Unclaim`: DELETE + `UPDATE duty_slots SET slots_filled = slots_filled - 1 WHERE id=?` in Transaktion kapseln
- [x] 4.3 Test: `TestClaimDutySlot_NoConcurrentOverclaim` — zwei gleichzeitige Goroutinen claimen letzten Slot; einer erhält 204, einer 409; `slots_filled == slots_total` danach

## 5. Members — Parents-Query und normalizeDate

- [x] 5.1 `internal/members/handler.go` Zeile ~1922: `u.name` → `u.first_name || ' ' || u.last_name AS name`
- [x] 5.2 `internal/members/handler.go` `normalizeDate`: Pivot von `>= 30` auf `>= 68` ändern
- [x] 5.3 Test: `TestGetParents_ReturnsCorrectName` — Member mit family_link → `/api/members/{id}/parents` gibt Name zurück (nicht leer)
- [x] 5.4 Test: `TestNormalizeDate` — Eingaben `"01.03.25"` → `"2025-03-01"`, `"15.07.72"` → `"1972-07-15"`, `"10.05.30"` → `"2030-05-10"`, `"10.05.2030"` → `"2030-05-10"`

## 6. Scheduler — Push-Idempotenz

- [x] 6.1 `internal/scheduler/scheduler.go`: `go push.SendToUsers(...)` und `INSERT OR IGNORE INTO notification_log` tauschen — erst INSERT OR IGNORE, dann RowsAffected prüfen, nur bei RowsAffected == 1 Push senden

## 7. Push — Stale-Subscription-Cleanup

- [x] 7.1 `internal/push/push.go`: `if resp.StatusCode == http.StatusGone` erweitern um `|| resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest`

## 8. Kader — UpdateKader Fehlerbehandlung

- [x] 8.1 `internal/kader/handler.go` `UpdateKader`: alle `tx.ExecContext`-Aufrufe (Zeilen ~230–276) mit Fehlerprüfung versehen; bei Fehler `tx.Rollback()` + HTTP 500
- [x] 8.2 `internal/kader/handler.go`: `tx.QueryRowContext(...).Scan(...)` für AgeClass-Lookup (Zeile ~268) auf Fehler prüfen; bei `sql.ErrNoRows` → HTTP 404; bei anderem Fehler → HTTP 500

## 9. Frontend — vorstand-Rolle in MembersPage

- [x] 9.1 `web/src/pages/MembersPage.tsx` Zeile 94: `const isAdmin = user?.role === 'admin'` → `const isAdmin = user?.role === 'admin' || user?.role === 'vorstand'`

## 10. Verifikation

- [x] 10.1 `go build ./...` fehlerfrei
- [x] 10.2 `/usr/local/go/bin/go test ./internal/auth/... ./internal/duties/... ./internal/members/... ./internal/kader/...` alle grün
- [x] 10.3 `pnpm -C web build` fehlerfrei (TypeScript-Fehler in api.ts und useLiveUpdates.ts geprüft)
- [ ] 10.4 Manueller Smoke-Test: Login → SSE-Verbindung aufgebaut (Network-Tab: kein `?token=` in der URL) → nach 16 min Token-Refresh → SSE reconnectet ohne Schleife
