## Context

Zwei rollenunabhängige Defekte in der Push-Pipeline (Details in `proposal.md`). Der Push-Pfad kennt keine Rollenlogik; „Rolle standard" im Ausgangsfall ist Korrelation, nicht Ursache. Betroffen sind zwei Domänen-Packages und ein Frontend-Hook:

- `internal/db/migrations/001_initial.up.sql:464` — CHECK ohne `chat`.
- `internal/notifications/handler.go:33` — `UpdateNotificationPreferences`, nicht-transaktionale Go-Map-Schleife.
- `internal/push/prefs.go` — `FilterByPushPref` / `GetAllPreferences` (kennt `chat` bereits).
- `internal/push/push.go:78,148` — Delete-Switch mit 410/404/401/400.
- `web/src/hooks/usePushSubscription.ts` — `catch {}` schluckt jeden Fehler.

Constraints: SQLite ohne `ALTER … DROP/ADD CONSTRAINT` → CHECK-Änderung erfordert Tabellen-Rebuild. VPS 1 GB RAM, kein neuer Dienst, kein ORM (`database/sql`).

## Goals / Non-Goals

**Goals:**
- `chat` als persistierbare Präferenz-Kategorie (DB + transaktionaler Handler + korrektes Filterverhalten).
- Abo-Löschung auf permanente Fehler (404/410) begrenzen; transiente Fehler (400/401/5xx) protokollieren statt löschen.
- Re-Subscribe-Fehler im Frontend beobachtbar machen.
- Regressions-Tests für beide Defekte gemäß `docs/agent/07-testing.md`.

**Non-Goals:**
- Diagnose/Behebung des konkreten Andrea-Falls (braucht Prod-DB-Abfrage, separat).
- Umfassende „alle Notification-Fälle"-Test-Suite (Folge-Exploration).
- Änderung des Krypto-/VAPID-Setups oder Einführung von Push-Retry-Queues.

## Decisions

**D1 — CHECK-Änderung via Tabellen-Rebuild (Migration 026).**
SQLite kann einen CHECK nicht in-place ändern. Standard-Rezept in einer Transaktion: `PRAGMA foreign_keys=OFF` (im Migrationskontext), neue Tabelle `notification_preferences_new` mit erweitertem CHECK anlegen, `INSERT INTO … SELECT * FROM notification_preferences`, alte Tabelle droppen, neue umbenennen. Die erlaubte Menge wird an Code/UI angeglichen: `games, trainings, duties, duty_reminders, carpooling, membership, chat`.
*Down-Migration:* Rebuild zurück auf die CHECK-Menge **ohne** `chat`; vorhandene `chat`-Zeilen werden dabei verworfen (im `.down.sql` dokumentieren).
*Alternative verworfen:* CHECK ganz entfernen und Validierung nur im Handler — verlöre die DB-seitige Integritätsgarantie, gegen Projekt-Konvention (Status/Enum als CHECK).

**D2 — `UpdateNotificationPreferences` transaktional + Whitelist.**
`sql.Tx` öffnen, alle Kategorien per Upsert schreiben, bei erster unbekannter Kategorie (nicht in der Whitelist) `tx.Rollback()` + HTTP 400; sonst `tx.Commit()` + 204. Die Whitelist wird als eine Quelle der Wahrheit im `push`-Package definiert (z.B. `push.ValidCategories`), von Handler und `GetAllPreferences` genutzt, damit Code und DB-CHECK nicht erneut auseinanderlaufen.
*Alternative verworfen:* nur DB-CHECK ausschlaggebend, 500 beibehalten — schlechtes API-Verhalten und weiterhin partielle Writes.

**D3 — Delete-Switch verengen.**
In `SendToUsers` und `SendToUserWithBadge` das `switch resp.StatusCode` auf `http.StatusGone, http.StatusNotFound` reduzieren (Delete). Für 400/401 ein `slog.Warn("push transient failure", "status", …, "subscription", s.id)` ohne Delete. Beide Funktionen teilen dieselbe Logik — eine gemeinsame Hilfsfunktion `handlePushResponse(db, subID, statusCode)` vermeidet Duplikat-Drift (die aktuell duplizierte Delete-Logik ist die Quelle des Bugs an zwei Stellen).
*Alternative verworfen:* 401 weiterhin löschen — 401 tritt bei transienten VAPID-Signaturfehlern serverweit auf und würde alle Abos wiederholt vernichten.

**D4 — Frontend beobachtbar.**
`catch (err) { console.warn('[push] subscribe failed', err) }` statt leerem `catch {}`. Kein UI-Element, kein Telemetrie-Ausbau (Non-Goal) — nur Konsolen-Sichtbarkeit als minimaler, DSGVO-neutraler Schritt.

**D5 — Test-Fundament: Send-Seam + Fixtures (Voraussetzung für D3-Tests).**
Der Delete-Switch (D3) lässt sich heute nicht testen: `TestConfig()` hat `VAPIDPrivateKey==""`, wodurch `SendToUsers`/`SendToUserWithBadge` sofort per Guard zurückkehren; zudem spricht `webpush.SendNotification` ohne Naht direktes HTTP. Daher wird als Package-Var-Seam `var sendNotification = webpush.SendNotification` eingeführt (idiomatisch wie `chat.pushFn`), im Test überschreibbar, um pro Statuscode das Delete-Verhalten zu prüfen. Ergänzend neue `testutil`-Fixtures `CreatePushSubscription` und `CreateNotificationPreference` (fehlen bislang komplett). Dieses Fundament wird bewusst **hier** angesiedelt (nicht im Folge-Change `notification-test-coverage`), weil D3 sonst gar nicht testbar wäre — der Coverage-Change baut nur noch die Breite darauf auf.
*Alternative verworfen:* echtes VAPID-Keypair + httptest-Endpoint + ECDH-Fixture (höhere Treue) — für die reine Statuscode-Logik von D3 Overkill; als optionaler E2E-Test dem Coverage-Change überlassen.

## Risks / Trade-offs

- **Rebuild-Migration verliert Daten bei Fehler** → Rebuild strikt in Transaktion; `INSERT … SELECT` kopiert 1:1; vor Prod-Lauf DB-Backup (ohnehin Deploy-Routine). Tabelle ist klein (eine Zeile pro Nutzer×Kategorie).
- **Foreign-Key-Referenzen auf `notification_preferences`** → keine; nur `user_id`-FK *raus* auf `users`. Beim Rebuild FK-Definition in der neuen Tabelle unverändert übernehmen.
- **Weniger Löschungen (D3) ⇒ tote Abos bei echtem 400/401 bleiben liegen** → akzeptabel: 404/410 decken den „endgültig weg"-Fall ab; echte Dauerfehler auf 400/401 sind selten und erzeugen nur Log-Rauschen, kein Fehlverhalten. Kein unbeschränktes Wachstum, da 404/410 weiterhin bereinigen.
- **Bestehende Tests kodieren das alte 401/400-Delete-Verhalten** → diese Tests müssen mit-angepasst werden (Teil der Tasks), sind aber die Regressions-Absicherung für D3.

## Migration Plan

1. Migration `026_notification_preferences_chat_category.{up,down}.sql` schreiben (D1), `make migrate-up` lokal.
2. Backend-Änderungen D2/D3, Frontend D4.
3. Tests (Handler-Tx/400, FilterByPushPref-chat, push-Delete-Matrix) grün; volles Gate (`/verify-change`).
4. Deploy führt `migrate up` automatisch aus (`make deploy`). Rollback: `make migrate-down` (Down-Rebuild ohne `chat`) + vorheriges Binary.

## Open Questions

- Keine blockierenden. (Der konkrete Andrea-Fall wird außerhalb dieses Change über eine Prod-DB-Abfrage geklärt.)
