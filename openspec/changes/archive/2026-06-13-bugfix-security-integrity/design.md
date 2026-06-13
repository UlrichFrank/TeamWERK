## Context

Ein strukturierter Code-Review des gesamten TeamWERK-Backends und -Frontends hat 13 bestätigte Bugs gefunden. Alle Befunde sind durch direktes Lesen des Codes verifiziert — keine spekulativen Probleme.

**Systemkontext:** Go + SQLite auf einem 1-GB-VPS (IONOS), React/TypeScript-Frontend, JWT-Auth mit Access Token (15 min, Memory) + Refresh Token (7 Tage, HttpOnly-Cookie), SSE für Live-Updates.

Die Bugs gliedern sich in drei Kategorien:

| Kategorie | Anzahl | Dateien |
|---|---|---|
| Auth/Security | 5 | `auth/handler.go`, `useLiveUpdates.ts`, `api.ts` |
| Business Logic Race Conditions | 4 | `duties/handler.go`, `members/handler.go`, `scheduler/scheduler.go` |
| Korrektheit/Datenqualität | 4 | `members/handler.go`, `push/push.go`, `kader/handler.go` |

## Goals / Non-Goals

**Goals:**
- Alle 13 bestätigten Bugs beheben
- Keine neuen Features einführen
- Jeder Fix ist eigenständig testbar und deploybar
- Bestehende API-Kontrakte bleiben unverändert (außer SSE-Auth-Migration)

**Non-Goals:**
- Rate Limiting auf Login/Register/ForgotPassword (wäre ein eigener Change mit Middleware-Entscheidung)
- JWT-Algorithm-Pinning über den aktuellen Stand hinaus (golang-jwt/v5 blockiert alg:none bereits)
- CSRF-Token-Einführung (Bearer-Auth im Header ist per Definition CSRF-sicher)
- Refactoring von `UpdateKader` über den Fehler-Check hinaus

## Decisions

### A1: Login-Timing — Dummy-bcrypt statt Constant-Time-Hash

**Entscheidung:** Im `ErrNoRows`-Branch einen Dummy-bcrypt-Vergleich gegen einen statischen Hash durchführen:
```go
var dummyHash = []byte("$2a$10$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.")
// Im ErrNoRows-Branch:
bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
http.Error(w, "invalid credentials", http.StatusUnauthorized)
return
```
Der `dummyHash` ist ein gültiger bcrypt-Hash mit Cost 10. `CompareHashAndPassword` läuft damit ~100ms — gleich lang wie ein echtes Lookup. Der Hash kann als Package-Level-Variable vorab berechnet werden.

Alternative verworfen: `time.Sleep(fixedDuration)` — fragil, da sich bcrypt-Zeiten mit CPU-Last ändern und ein Sleep zu kurz oder zu lang sein kann.

### A2: Refresh-Token-Rotation — Transaktion

**Entscheidung:** DELETE + INSERT + Cookie-Set in einer DB-Transaktion kapseln. Fehler explizit prüfen und mit HTTP 500 quittieren.

```go
tx, err := h.db.BeginTx(r.Context(), nil)
// DELETE, INSERT, tx.Commit()
// Cookie erst nach erfolgreichem Commit setzen
```

Hinweis: SQLite mit WAL-Mode und `PRAGMA foreign_keys=ON` unterstützt Transaktionen korrekt. Kein RETURNING nötig (nicht kompatibel mit alten SQLite-Versionen) — `LastInsertId` oder direktes Commit reicht.

### A3: SSE-Auth — Cookie statt Query-Parameter

**Entscheidung:** Der Backend-SSE-Handler (`hub/handler.go`) soll das Refresh-Token-Cookie für die Authentifizierung nutzen, da dieses bereits als HttpOnly-Cookie vorliegt und nicht in Server-Logs erscheint.

Konkrete Umsetzung: Die bestehende `auth.Middleware` prüft `Authorization: Bearer`. Da EventSource keine Header sendet, muss der SSE-Handler direkt das Cookie lesen und validieren — entweder einen Kurz-JWT aus dem Cookie ableiten oder den Refresh-Token direkt validieren und eine einmalige SSE-Session anlegen.

**Empfohlener Weg:** Einen neuen Cookie-basierten Auth-Pfad ausschließlich für den SSE-Endpunkt implementieren: Cookie validieren → UserID/Claims extrahieren → SSE-Verbindung aufbauen. Der `?token`-Query-Parameter-Pfad wird aus der Middleware entfernt.

**Frontend:** `useLiveUpdates.ts` übergibt keinen Token mehr. `withCredentials: true` (bereits gesetzt via Axios) gilt nicht für EventSource — EventSource sendet Cookies automatisch, wenn sie same-origin ist. Kein weiteres Frontend-Setup nötig.

SSE-Reconnect: `useEffect`-Dependency-Array auf `[accessToken]` setzen. Wenn der Axios-Interceptor einen neuen Access Token setzt, triggert der Effect-Cleanup die alte EventSource und öffnet eine neue Verbindung mit dem aktualisierten Cookie.

### B1: Duty-Slot-Claim — Konditionelles UPDATE als Gate

**Entscheidung:** Das atomare Pattern für den Claim:

```sql
UPDATE duty_slots
SET slots_filled = slots_filled + 1
WHERE id = ? AND slots_filled < slots_total
```
Danach `RowsAffected` prüfen: 0 → HTTP 409 (voll). 1 → INSERT duty_assignment. Falls INSERT fehlschlägt (UNIQUE-Verletzung: Nutzer bereits eingetragen) → UPDATE rückgängig machen via:
```sql
UPDATE duty_slots SET slots_filled = slots_filled - 1 WHERE id = ?
```

Alternative (Transaktion mit SELECT FOR UPDATE): SQLite unterstützt kein SELECT FOR UPDATE in WAL-Mode zuverlässig über mehrere Verbindungen. Das konditionelle UPDATE ist sicherer und braucht keine Transaktion.

### B2: Unclaim — Transaktion

**Entscheidung:** DELETE + Decrement in einer Transaktion. Einfachster sicherer Ansatz, da Unclaim selten ist und keine Performance-Anforderungen hat.

### C1: normalizeDate-Pivot

**Entscheidung:** Pivot von `>= 30` auf `>= 68` (ISO-8601-Empfehlung). Rationale: Jahrgang 68 = 1968 (aktuell 57 Jahre alt), Jahrgang 67 = 2067 (in 41 Jahren noch nicht geboren). Dieser Pivot hat ~40 Jahre Puffer. Ein gleitender Pivot (`currentYear - 2000 + 1`) wäre präziser, aber unnötige Komplexität für einen CSV-Import.

### C2: Push-Cleanup bei 401/400

**Entscheidung:** Analoge Behandlung wie 410 — Subscription löschen bei 401 und 400. 5xx-Fehler werden weiterhin nur geloggt (transient). Dies entspricht der Web Push Specification §7.

## Risks / Trade-offs

- [SSE-Auth-Migration: bestehende Clients verlieren die Verbindung nach Deploy] → Kein Client-Problem: EventSource reconnectet automatisch. Der einzige Moment ist der Deploy selbst (kurze Unterbrechung aller SSE-Verbindungen). Nach Reconnect wird das Cookie-Auth-Verfahren genutzt. Frontend-Deploy und Backend-Deploy müssen koordiniert (gleichzeitig) erfolgen.

- [Dummy-bcrypt-Hash im Login-Code] → Der Hash ist ein statischer konstanter Wert. Er wird nie für echte Auth genutzt. Keine Sicherheitsrisiken.

- [konditionelles UPDATE + separater INSERT für Duty-Claim] → Bei hoher Konkurrenz (unrealistisch für einen Handball-Verein) könnten kurzzeitige Inkonsistenzen zwischen `slots_filled` und tatsächlichen Assignments entstehen, falls der INSERT nach dem UPDATE-Gate fehlschlägt und das Rollback-UPDATE (decrement) auch fehlschlägt. Dieses Szenario ist für die Zielnutzerzahl (< 200 User) vernachlässigbar; eine vollständige SQLite-Transaktion wäre die robustere Alternative.

- [UpdateKader: Fehlerprüfung aller 12 tx.ExecContext-Aufrufe] → Diese werden direkt nach dem Aufruf geprüft. Da alle in einer Transaktion sind, würde ein Fehler ohnehin beim Commit auffallen. Die explizite Prüfung verbessert die Debuggability und gibt sofortiges HTTP 500 zurück, statt auf tx.Commit zu warten.

## Migration Plan

Kein Schema-Change, keine neue Migration erforderlich.

**Deploy-Reihenfolge:**
1. Backend bauen und deployen (`make deploy`)
2. Frontend wird dabei ebenfalls deployt (eingebettet in Binary)
3. Bestehende SSE-Verbindungen brechen kurz ab und bauen sich mit Cookie-Auth neu auf
4. Kein manueller Eingriff erforderlich

**Rollback:** Der vorherige Binary-Stand kann jederzeit via `systemctl stop teamwerk && cp bin/teamwerk.prev /usr/local/bin/teamwerk && systemctl start teamwerk` wiederhergestellt werden. Kein DB-Rollback nötig (keine Schema-Änderungen).
