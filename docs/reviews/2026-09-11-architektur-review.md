# TeamWERK Architektur-Review

Stand 11.09.2026 · Branch `feat/chat-message-search` @ `0c922564` · gestaltete Fassung: `2026-09-11-architektur-review.html`

Vier Prüfstränge (Backend, Frontend, Security, Datenbank/Betrieb/CI) plus `make metrics`, jeder Befund mit Fundstelle. Die als kritisch und hoch eingestuften Security-Befunde wurden zusätzlich von Hand im Code gegengelesen.

## Kennzahlen

| Kennzahl | Wert |
|---|---:|
| Commits seit 17.05.2026 | 1.369 |
| Go-Zeilen Produktivcode / Tests | 40,4k / 57,3k (Ratio 1,42) |
| TypeScript-Zeilen | 50,4k |
| Go-Coverage / Frontend-Coverage | 64,7 % / 50,3 % |
| Migrationen | 59, lückenlos |
| Lint-Issues (golangci, ESLint-Errors) | 0 |
| Komplexitäts-Verstöße (gocyclo / gocognit / funlen / dupl) | 20 / 58 / 8 / 9 |
| Frontend-Duplikation (jscpd, 5-Zeilen-Schwelle) | 4,29 % |

## Gesamturteil: 3,3 / 5

**Solide Fundamente, Schulden eine Ebene tiefer.** Schichtenmodell, Test-Harness und die mechanischen Gates sind für ein Vereinsprojekt ungewöhnlich reif und verhindern Erosion, statt sie nur zu dokumentieren. Zero-Knowledge für Bankdaten ist ernsthaft umgesetzt, SQL durchgehend parametrisiert, Token-Hygiene lehrbuchhaft.

Die Schwächen liegen fast ausschließlich in der Handler-Implementierung und dort, wo die Gates per Konstruktion nicht hinsehen: Objekt-Level-Autorisierung, der Upload-Pfad, verschluckte Fehler und Logging. Ein Befund ist kritisch, weil er die Zero-Knowledge-Zusage vollständig aushebelt; er ist mit kleinem Aufwand zu schließen.

## Scorecard (1 bis 5, 5 = vorbildlich)

| Dimension | Wert | Kurzbegründung |
|---|:-:|---|
| Test-Harness & CI | 5 | 14 Architektur-Gates, Permission-Matrix, E2E im CI |
| Architektur & Schichtung | 4 | Foundation/Domain/Composition mechanisch erzwungen |
| API-Schicht Frontend | 4 | Single-Flight-Refresh, Cache-Leerung bei Identitätswechsel, kein `any` |
| Datenschutz & Krypto | 4 | Zero-Knowledge zu Ende geführt, Retention implementiert |
| Authentifizierung | 4 | bcrypt, gehashte Tokens, Lockout, Rate-Limiting; Impersonation ohne Audit |
| Schema & Migrationen | 4 | Down-Skripte vollständig; Indizes fehlen an heißen Stellen |
| Dokumentation | 4 | `docs/agent` ohne Drift; kein README für Menschen |
| Styling & Design-System | 4 | Button-Gate vorbildlich; 154 rohe Farbklassen ohne Gate |
| Datenzugriff | 3 | DSN-Pragmas sauber; DATE-Gotcha zweimal ungefixt, N+1 in Listen |
| Nebenläufigkeit | 3 | Hub sauber; kein `recover()` in Goroutinen, Context-Bug im Reset-Pfad |
| Frontend-Struktur & State | 3 | `ChatPage` 3.276 Zeilen, 55 `useState` |
| Performance Frontend | 3 | 323 KB gzip Erstladung, 1 von 45 Routen lazy |
| Deploy & DB-Betrieb | 3 | kein Smoke-Test, kein Rollback, kein Checkpoint/VACUUM |
| Eingabe-Härtung | 3 | SQL parametrisiert; Upload vertraut dem Client-MIME |
| Handler-Qualität Go | 2 | 180 ungeprüfte Scan/Exec, 5 `writeJSON`-Kopien, divergente Paginierung |
| Fehlerbehandlung & Logging | 2 | 469× „internal error“, 500er ohne slog, 102× `err.Error()` an den Client |
| Autorisierung (Objekt-Ebene) | 2 | `media.Serve` ohne Objekt-Check, 6 weitere Routen ohne Gate |
| Zugänglichkeit | 2 | 2 von 23 Modals mit `role=dialog`, `EditModal` ohne Fokus-Management |

## Was gut gelöst ist

- **Architektur wird geprüft, nicht behauptet.** `internal/arch/` mit 14 Gates (Layering, Broadcast, Push-Fan-out, Audience, tote Capabilities, Authz- und Permission-Matrix über 354 Routen × 11 Personas). Jede Domäne wird genau einmal importiert, vom Composition Root.
- **Sicherheits-Grundlagen.** Keine SQL-Injection. Alle Tokens SHA256-gehasht, 32 Byte `crypto/rand`, Einmalverwendung, Lockout, Rate-Limiting, Refresh-Cookie `HttpOnly`+`Secure`+`SameSite=Strict`, CSP ohne `unsafe-inline` im `script-src`. H4A-Passwort nachweislich nicht persistiert.
- **Zero-Knowledge zu Ende gedacht.** Server hält keinen Entschlüsselungsschlüssel, PBKDF2 600k, RSA-2048, AES-GCM-256, Modellgrenzen ehrlich dokumentiert.
- **Frontend-Infrastruktur.** `lib/api.ts`, `lib/errors.ts`, Button-Gate, 121 verhaltensbasierte Testdateien ohne Snapshot.
- **Betriebsdisziplin.** DSN-Pragmas pro Verbindung, `SQLITE_BUSY`-Metrik, Panic-Recovery, Wartungsmodus, Umzugs-Runbook, Dependabot, E2E im CI.

## Befunde

### Security und Datenschutz

| Schwere | Befund | Fundstelle | Aufwand |
|---|---|---|---|
| **kritisch** | Upload-XSS mit Diebstahl des Vereins-Privatschlüssels: `POST /api/upload/user-photo` überspringt das Sniffing bei erlaubtem Client-`Content-Type`, Endung aus Dateiname, `ServeContent` liefert `.html` als `text/html` ohne `Content-Disposition`; `.js` unter `script-src 'self'` erlaubt; Tresor hält PKCS8 exportierbar in `sessionStorage`. Bei entsperrtem Tresor ist die Bankdaten-Verschlüsselung aller Mitglieder dauerhaft gebrochen. Gegenmuster existiert in `internal/media`. | `internal/upload/handler.go:112-140, 512`, `web/src/contexts/VaultContext.tsx:43, 93` | S, Folge M |
| hoch | `GET /api/media/{id}` ohne Objekt-Check; alle Chat- und Mitteilungsbilder für jeden Eingeloggten über durchzählbare IDs. `matrix_test.go:316` dokumentiert das als Sollverhalten. | `internal/media/handler.go:134-152` | S |
| hoch | Sechs Routen ohne Membership-/Team-Gate: `GET /api/training-sessions/{id}`, `POST /api/games/{id}/lineup` (vereinsweit statt teamgebunden), `POST /chat/conversations/{id}/read`, `DELETE …/members/me`, Duty `fulfill`/`cash-substitute` | `trainings/handler.go:1445`, `games/handler.go:3151`, `chat/handler.go:1128, 1158`, `duties/handler.go:1377-1400` | M |
| hoch | HSTS nirgends aktiv; `.env.example` liefert ein funktionsfähiges `JWT_SECRET`, Server prüft nur „nicht leer“ | `nginx-teamwerk.conf:50`, `config.go:125`, `.env.example:4` | S |
| hoch | Impersonation ohne Audit-Spur, Token vom echten Login nicht unterscheidbar | `auth/handler.go:986-1025` | S |
| mittel | 102× `err.Error()` an den Client; Backups unverschlüsselt; kein Löschkonzept für `users`, kein Art.-15-Export; kein `govulncheck`/`pnpm audit` im CI; 18 Dependabot-Alerts auf `main` (7 high) | u. a. `internal/kader`, `deploy/backup-teamwerk.sh`, `members/handler.go:2431` | S bis M |

### Backend (Go)

| Schwere | Befund | Fundstelle | Aufwand |
|---|---|---|---|
| hoch | 500er ohne Spur: `games` 86 Fehlerantworten bei 0 slog-Aufrufen, `trainings` 61:0, `kader` 37:0; 62× `fmt.Fprintf(os.Stderr)`; 1.518 Plain-Text-Fehler gegen 31 maschinenlesbare Codes | `internal/games`, `internal/trainings` | M |
| hoch | 180 ungeprüfte `Scan`/`Exec` per errcheck-Ausnahme; `duties.Fulfill`/`CashSubstitute` antworten 204 unabhängig vom UPDATE-Ergebnis; ungeprüfter Scan + `[:10]` ist ein Slice-Panic | `.golangci.yml`, `duties/handler.go:1375, 1387`, `absences/handler.go:499` | L (Ratchet) |
| hoch | Kein `recover()` in Hintergrund-Goroutinen; Reset-Token-INSERT mit `r.Context()` nach der Antwort (Mail raus, Link tot) | `health/recover.go:26`, `auth/handler.go:1070` | S |
| mittel | DATE-Gotcha ungefixt in `RegenerateSlots` (Repair-Lauf tut still nichts) und `PreviewSlots` (meldet nie Konflikt) | `games/handler.go:2369, 2300` | S |
| mittel | Paginierung 4× kopiert, 2× ohne Deckel (`GET /api/games?limit=100000`); 5 `writeJSON`-Kopien; N+1 in `kader.ListKader`, `ListGames`, `ListConversations`; `duties.Claim` ohne Transaktion | `kader/handler.go:294`, `games/handler.go:613`, `duties/handler.go:1310` | M |
| mittel | Gott-Dateien: `games/handler.go` 3.806 Zeilen, `members` 3.040, `chat` 2.259; `chat.ListMessages` kognitive Komplexität 107; Ratchet zweimal nach oben re-baselined, Coverage-Floor 38 % bei Ist 68 % | `metrics/thresholds.yml` | L |
| niedrig | `SetMaxOpenConns` ungesetzt; `DB_PATH` stiller Default; VAPID in `serve()` unvalidiert; `internal/mailer` ohne Test | `db/db.go:28`, `config.go` | S |

### Frontend (React)

| Schwere | Befund | Fundstelle | Aufwand |
|---|---|---|---|
| hoch | `ChatPage.tsx`: 3.276 Zeilen, 8 Komponenten, 55 `useState`, 23 `useRef`; `KalenderPage` 2.003/64, `AdminSettingsPage` 1.426/61 | `pages/ChatPage.tsx:255-2258` | M |
| hoch | Modals für Tastatur/Screenreader unbenutzbar: 2 von 23 mit `role="dialog"`, `EditModal` ohne Rolle, `aria-modal`, Fokus-Trap | `components/EditModal.tsx:24-30` | S |
| mittel | Typ-Duplikation mit Drift: `Member` 9×, `Team`/`Season` 7×, `Game` 6×; `is_active` vs. `isActive` | `AdminTrainingsPage.tsx:58`, `DashboardPage.tsx:20` | M |
| mittel | Kein Route-Splitting: 50 statische Imports, 1 lazy, 323 KB gzip Erstladung | `App.tsx:21` | S |
| mittel | 154 rohe Tailwind-Farbklassen (63 in `MemberStammdatenTab`); 27 hand-gerollte Ladezustände, 11× `set-state-in-effect` | `components/admin/MemberStammdatenTab.tsx` | S |
| niedrig | Hooks an zwei Orten; `noUncheckedIndexedAccess` aus; `exhaustive-deps` nur Warnung; 0× `React.memo` | `tsconfig.json`, `eslint.config.js:32-56` | S |

### Datenbank, Deploy, CI

| Schwere | Befund | Fundstelle | Aufwand |
|---|---|---|---|
| hoch | `duty_slots` ohne Index bei 32 `WHERE game_id`; `family_links` nur PK (Rückrichtung ungestützt); `duty_assignments` ohne `user_id`-Index; Chat-Suche per `LIKE '%…%'` ohne FTS5 | `001_initial.up.sql:449`, `chat/search.go` | S / M |
| mittel | Deploy ohne Health-Check, Rollback, Staging; 27 von 59 Migrationen transformieren Daten direkt auf Prod | `Makefile:97-126` | S bis M |
| mittel | Kein Checkpoint/VACUUM; `message_reads`, `notification_log` ohne Retention; Backup nur pull-basiert | `deploy/setup-vps.sh`, `internal/scheduler` | S / M |
| niedrig | `metrics-gate` nicht im CI; Coverage-Floors weit unter Ist; kein README | `.github/workflows/ci.yml`, `metrics/thresholds.yml` | S |

## Handlungsanweisung

Aufwand: S unter einem Tag, M ein bis drei Tage, L eine Woche und mehr. Jeder Punkt ein eigener OpenSpec-Change mit Test, der ohne den Fix rot ist.

### Welle 1 — diese Woche: Sicherheit schließen

1. **Upload-Pfad härten** (S): immer sniffen, Endung aus erkanntem Typ (`media.extByMime`), `Content-Disposition: attachment` außer für Bilder. Test: HTML mit `image/png`-Header wird abgelehnt.
2. **Tresor-Schlüssel non-extractable** (M): `CryptoKey` mit `extractable=false` in IndexedDB statt PKCS8 in `sessionStorage`.
3. **`media.Serve` gaten** (S) über Konversations-Mitgliedschaft bzw. `broadcast_reads`; Matrix-Eintrag korrigieren.
4. **Sechs Objekt-Gates nachziehen** (M).
5. **HSTS aktivieren, `JWT_SECRET` ≥ 32 Byte erzwingen, Beispielwert aus `.env.example` entfernen** (S).
6. **Impersonation auditieren** (S): `user_events`-Eintrag plus `impersonated_by`-Claim.
7. **Dependabot-Alerts abarbeiten, `govulncheck` + `pnpm audit` als CI-Job** (S).

### Welle 2 — nächste 30 Tage: Betriebsfähigkeit und Fehlersichtbarkeit

1. **`internal/httpx`** (M): `WriteJSON`, `WriteError` (JSON-Code außen, `slog.Error` innen, nie `err.Error()` an den Client), `PathID`, `Paging` mit Deckel. Zuerst `games`, `trainings`, `kader`.
2. **`notify.SendAsync` mit `recover()` + Panic-Metrik; `auth/handler.go:1070` auf `context.Background()`** (S).
3. **Migration 060:** `idx_duty_slots_game`, `idx_duty_slots_date_season`, `idx_family_links_member`, `idx_duty_assignments_user` (S).
4. **DATE-Gotcha in `RegenerateSlots`/`PreviewSlots` fixen** mit Regressionstest (S).
5. **Deploy absichern** (M): Smoke-Test gegen `/api/healthz`, `deploy-rollback`-Target, Backup-Cron auf dem VPS, wöchentlicher `wal_checkpoint(TRUNCATE)`.
6. **Fixture-Matrix für Objekt-Autorisierung** (M): je `{id}`-Route fremdes Objekt → 403/404. Ohne dieses Gate kehrt die Fehlerklasse zurück.
7. **`EditModal` zugänglich machen** (S), eigenständige Modals darauf ziehen (M).

### Welle 3 — Quartal: Struktur-Schulden abtragen, Ratchets scharf stellen

1. **Errcheck-Ausnahme für `Rows.Scan`/`Exec` aufheben**, ~180 Stellen als Ratchet (L).
2. **`ChatPage.tsx` in `components/chat/` auftrennen, `MessageBubble` memoisieren** (M); dann `games/handler.go` entlang der Routengruppen (L).
3. **Zentrales Typmodul `lib/types/`** plus Gate gegen Doppeldeklaration (M).
4. **Route-Splitting** für `/admin/*`, `ChatPage`, `KalenderPage`; `useResource`-Hook für die 27 hand-gerollten Ladezustände (S + M).
5. **Gate für rohe Farbklassen** nach `buttonStyles.gate.test.ts`, Startwert 154 als Ratchet (S).
6. **Ratchets nachziehen:** Coverage-Floors 60 % / 45 %, `metrics-gate` im CI, `exhaustive-deps` auf `error`, `noUncheckedIndexedAccess` messen (S).
7. **Datenschutz-Rest** (M): Löschkonzept `users`, Art.-15-Export, verschlüsselte Backups, FTS5-Entscheidung, PII-History-Rewrite vor dem Open-Sourcing.

## Methodik und Grenzen

Statische Analyse auf `0c922564`, vier unabhängige Prüfstränge mit Fundstellenpflicht, `make metrics`. Kritische und hohe Security-Befunde von Hand gegengelesen. Nicht durchgeführt: dynamische Tests, Penetrationstest, Lasttest, CVE-Scan (Dependabot-Stand von GitHub). Bewertungen sind Expertenurteile auf einer Fünferskala.
