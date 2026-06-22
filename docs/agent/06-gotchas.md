# Bekannte Gotchas

**SQLite DATE-Felder:** API gibt Datumsfelder als ISO-Timestamp zurück (`"2026-05-30T00:00:00Z"`). Im Frontend immer `.slice(0, 10)` für Vergleiche und `date + 'T12:00:00'`-Konstruktionen.

**Aktive Saison:** Spielplan, Dienst-Erstellung und Dienst-Konten setzen eine aktive Saison voraus (Verwaltung `/admin/saisons`). Ohne aktive Saison schlagen game- und slot-Inserts mit FK-Fehler fehl.

**SSE Live-Updates:** Jede Mutations-Route (`POST`/`PUT`/`DELETE`) muss `h.hub.Broadcast("domain-event")` aufrufen; das Frontend abonniert mit `useLiveUpdates((event) => { if (event === 'domain-event') reload() })`. `Handler`-Structs mit Mutationen brauchen ein `hub *hub.EventHub`-Feld (in `main.go` via `NewHandler(db, hub)` übergeben). Fehlt `Broadcast` (Backend) **oder** `useLiveUpdates` (Frontend), bleibt die Seite nach fremden Änderungen stumm.

**Push Notifications:** Infrastruktur in `internal/notifications/`. VAPID-Keys via `go run ./cmd/teamwerk gen-vapid` in `.env` (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL`). Senden immer als Goroutine (darf den HTTP-Response nicht blockieren):

```go
go notifications.SendToUsers(h.db, h.cfg, []int{userID1, userID2}, "Titel", "Text", "/ziel-url")
```

Frontend-Hook `usePushSubscription` (in `AppShell.tsx`) registriert automatisch beim App-Start. iOS nur als Homescreen-PWA (`display-mode: standalone`). Subscriptions in `push_subscriptions`; ungültige Endpoints (HTTP 410) werden bereinigt. Scheduled Notifications → Job im `internal/scheduler/`, idempotent via `notification_log`.

**App-Icon-Badge:** Chat-Pushes (`chat.Handler.SendMessage`, `SendBroadcast`) berechnen pro Empfänger via `chat.ComputeUnreadForUser` den aktuellen Chat-Unread und senden ihn als absolutes Feld `badge: number` in der Payload — versendet über `push.SendToUserWithBadge` statt `push.SendToUsers`. Der Service Worker (`web/src/sw.ts`) setzt damit `navigator.setAppBadge`/`clearAppBadge`. Im offenen Frontend setzt `AppShell` den Badge zusätzlich live über `useEffect([chatUnread])`. Andere Push-Caller (Games/Trainings/Duties) bleiben bei `push.SendToUsers` ohne `badge`-Feld — der Service Worker rührt den Badge dann nicht an.

**Auto-Duty-Regen:** Jede Spieländerung (`POST/PUT/DELETE /api/games/{id}`) triggert Regeneration der Dienst-Slots für Event-Datum ± 1 Tag (beachtet `same_day_behavior`/`adjacent_day_behavior` der Duty-Types). Slots mit `is_custom=1` (manuell angelegt/editiert) werden geschont. Response enthält `regen_summary`. Vor Deploy manuell-editierte Bestandsslots mit `UPDATE duty_slots SET is_custom=1 WHERE id IN (...)` schützen.

**SEPA-Beitragslauf:** `/admin/beitragslauf` (`vorstand`, `kassierer`, `admin`). Bewusst einfach: **kein Pro-rata** (voller Jahresbeitrag), Fälligkeit **immer 01.07.** der Saison, alle Lastschriften **RCUR** (keine FRST), Spieler gelten als Kinder. Vor dem ersten Lauf müssen die SEPA-Stammdaten (`glaeubiger_id`, `iban`, `bic`, `kontoinhaber`) unter Einstellungen → Verein gepflegt sein, sonst liefert `POST /api/fee-run/export` HTTP 400. Beitragsmatrix unter Einstellungen → Beiträge (3 Kategorien, Cent, Historie via `valid_from`). „Lauf bestätigen" schreibt das append-only Saison-Protokoll. Kassierer darf Mitglieder lesen + Bankdaten via `PUT /api/members/{id}/bank-details` korrigieren.
