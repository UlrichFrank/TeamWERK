## Why

Ein umfassender Code-Review hat 13 bestätigte Bugs gefunden — darunter sicherheitskritische Lücken (Login-Timing-Attack, Token-Exposition in Server-Logs, nicht-atomare Token-Rotation) und Datenkorrektheitsfehler (Überbuchung von Dienst-Slots, kaputte Eltern-Abfrage, Doppel-Push durch Race). Diese Bugs sind im Produktivbetrieb aktiv und werden mit diesem Change vollständig behoben.

## What Changes

**Gruppe A — Sicherheit (auth + SSE)**

- `auth/handler.go` Login: Dummy-bcrypt-Aufruf im ErrNoRows-Branch, um Timing-Attack auf E-Mail-Enumeration zu schließen
- `auth/handler.go` Refresh: DELETE + INSERT in Transaktion kapseln; Fehler nicht mehr mit `_` verwerfen
- `auth/handler.go` Register: `bcrypt.GenerateFromPassword`-Fehler prüfen → HTTP 500 statt leerer Hash in DB
- `web/src/hooks/useLiveUpdates.ts`: Token aus URL-Parameter entfernen; SSE-Endpunkt über HttpOnly-Cookie authentifizieren
- `web/src/hooks/useLiveUpdates.ts`: EventSource nach Token-Refresh mit frischem Token neu aufbauen (Dependency auf `accessToken`)
- `web/src/lib/api.ts`: Shared `refreshPromise` im 401-Interceptor, um konkurrente Refresh-Aufrufe zu verhindern

**Gruppe B — Korrektheit (Business Logic)**

- `internal/duties/handler.go` Claim: Atomare Sequenz via konditionellem UPDATE + RowsAffected statt SELECT/INSERT-Race
- `internal/duties/handler.go` Unclaim: DELETE + UPDATE in Transaktion
- `internal/members/handler.go` Parents-Query: `u.name` → `u.first_name || ' ' || u.last_name`
- `internal/members/handler.go` normalizeDate: Pivot von `>= 30` auf `>= 68` (ISO-8601-Empfehlung)
- `internal/scheduler/scheduler.go` Push-Reminder: `INSERT OR IGNORE` vor `go push.SendToUsers` ausführen; RowsAffected prüfen
- `internal/push/push.go`: HTTP 401 und 400 löschen ebenfalls die Subscription (wie bereits 410)
- `internal/kader/handler.go` UpdateKader: Fehler aller `tx.ExecContext`-Aufrufe prüfen; bei Fehler rollback + HTTP 500

## Capabilities

### New Capabilities

_(keine neuen Capabilities — ausschließlich Bugfixes)_

### Modified Capabilities

- `auth`: Refresh-Token-Rotation wird atomar; Login-Timing-Angriff wird durch Dummy-bcrypt-Aufruf abgemildert; bcrypt-Fehler bei Register werden korrekt behandelt
- `sse-live-updates`: Token-Transport wechselt von URL-Query-Parameter auf Cookie; EventSource wird nach Token-Refresh neu aufgebaut
- `duties`: Claim-Sequenz wird race-frei (konditionelles UPDATE + RowsAffected); Unclaim in Transaktion
- `members`: Parents-Query gibt korrekte Ergebnisse zurück; normalizeDate-Pivot korrigiert für Spieler ab 2030
- `push-reminders`: Idempotenz-Logik dreht die Reihenfolge um (erst loggen, dann senden)
- `web-push-subscriptions`: Cleanup bei HTTP 401 und 400 ergänzt

## Impact

- **Backend:** `internal/auth/handler.go`, `internal/duties/handler.go`, `internal/members/handler.go`, `internal/scheduler/scheduler.go`, `internal/push/push.go`, `internal/kader/handler.go`
- **Frontend:** `web/src/lib/api.ts`, `web/src/hooks/useLiveUpdates.ts`
- **Kein Schema-Change**, keine neue Migration
- **SSE-Authentifizierung:** Backend-Middleware für den `/api/events`-Endpunkt muss auf Cookie-basierte Auth umgestellt werden (statt Query-Parameter-Token-Check)
