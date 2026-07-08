## Why

Bei der Untersuchung eines Falls, in dem eine Nutzerin (Rolle `standard`) keine Chat-Push-Nachrichten mehr erhält, sind zwei strukturelle Defekte in der Push-Pipeline aufgefallen. Beide sind **rollenunabhängig** — es gibt an keiner Stelle des Push-Pfads rollenbasierte Logik; die System-Rolle beeinflusst die Zustellung nicht. Die Defekte betreffen jede/n Nutzer:in und untergraben die Verlässlichkeit der Benachrichtigungen.

## What Changes

- **Defekt 1 — `chat`-Kategorie ist im DB-CHECK verboten:** Der CHECK-Constraint von `notification_preferences.category` (Migration `001`) listet `chat` nicht auf, obwohl das Profil-UI (`ProfileMiscTab.tsx`) und `push.GetAllPreferences` `chat` als vollwertige Kategorie behandeln. Folgen: (a) `PUT /api/profile/notification-preferences` mit dem Chat-Toggle verletzt den CHECK → **HTTP 500**, und da der Handler pro Kategorie in einer nicht-transaktionalen Go-Map-Schleife schreibt, persistiert ein **nichtdeterministischer Teil** der übrigen Kategorien; (b) da eine `chat`-Zeile nie gespeichert werden kann, findet `FilterByPushPref(…, "chat")` nie `push_enabled=0` → der Chat-Toggle ist **kosmetisch** und kann Chat-Pushes nicht abschalten.
  - Migration `026` baut `notification_preferences` neu auf (SQLite: neue Tabelle / copy / drop / rename in einer Transaktion) und nimmt `chat` in den CHECK auf.
  - `UpdateNotificationPreferences` läuft künftig **in einer einzigen Transaktion** (alles-oder-nichts) und weist **unbekannte Kategorien mit HTTP 400** statt 500 ab.
- **Defekt 2 — zu aggressives Löschen von Push-Abos:** `push.SendToUsers` / `SendToUserWithBadge` löschen ein `push_subscriptions`-Abo bei HTTP 410, 404 **und** 401, 400. Die Web-Push-Spezifikation verlangt das Entfernen nur bei **404/410** (Endpoint endgültig weg). 401/400 sind transiente VAPID-Signatur-/Payload-Fehler; ein Löschen darauf vernichtet ein **noch gültiges** Abo dauerhaft. Da das erneute Abonnieren (`usePushSubscription`) jeden Fehler still in `catch {}` schluckt, bleibt der Verlust unbemerkt und unbehoben.
  - Löschung künftig **nur bei 404/410**; 401/400 werden **geloggt** (`slog.Warn`), nicht gelöscht.
  - Das Re-Subscribe im Frontend wird **beobachtbar** (Logging statt stilles Schlucken), damit künftige Abo-Verluste diagnostizierbar sind.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `notification-preferences`: Die `chat`-Kategorie wird persistierbar (bislang durch den DB-CHECK unmöglich); das Speichern von Präferenzen ist transaktional (alles-oder-nichts) und lehnt unbekannte Kategorien mit HTTP 400 statt 500 ab.
- `web-push-subscriptions`: Abos werden nur noch bei permanenten Fehlern (404/410) gelöscht; transiente Fehler (400/401) führen zu Logging statt Löschen.

## Impact

- **Migration:** neu `internal/db/migrations/026_notification_preferences_chat_category.up.sql` + `.down.sql` (Tabellen-Rebuild inkl. `chat` im CHECK).
- **Backend:** `internal/notifications/handler.go` (`UpdateNotificationPreferences` → transaktional + Kategorie-Whitelist/400); `internal/push/push.go` (Delete-Switch in `SendToUsers` und `SendToUserWithBadge` auf 404/410 verengen, 400/401 loggen).
- **Frontend:** `web/src/hooks/usePushSubscription.ts` (Fehler beobachtbar loggen statt `catch {}` still zu verwerfen).
- **Tests:** neue/ergänzte Tests in `internal/notifications/`, `internal/push/` gemäß `docs/agent/07-testing.md` (siehe Test-Anforderungen in `tasks.md`).
- **Kein** neuer externer Dienst, keine neue Abhängigkeit, kein nennenswerter RAM-Footprint.
- **Kein Datenverlust:** bestehende `notification_preferences`-Zeilen werden beim Rebuild 1:1 kopiert.
