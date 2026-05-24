## 1. Datenbank & Konfiguration

- [x] 1.1 Migration `013_push_subscriptions.up.sql` erstellen: Tabelle `push_subscriptions (id INTEGER PK, user_id INTEGER FK, endpoint TEXT UNIQUE, p256dh TEXT, auth TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`
- [x] 1.2 Migration `013_push_subscriptions.down.sql` erstellen: `DROP TABLE IF EXISTS push_subscriptions`
- [x] 1.3 `go get github.com/SherClockHolmes/webpush-go` hinzufügen
- [x] 1.4 `config`-Package um `VAPIDPublicKey`, `VAPIDPrivateKey`, `VAPIDEmail` erweitern (aus .env lesen)
- [x] 1.5 Subcommand `gen-vapid` in `cmd/teamwerk/main.go` ergänzen: generiert VAPID-Keypair und gibt ihn auf stdout aus
- [x] 1.6 `.env.example` um `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL` erweitern

## 2. Backend: Notifications-Package

- [x] 2.1 Package `internal/notifications` anlegen mit `Handler struct { db *sql.DB; cfg *config.Config }`
- [x] 2.2 `GET /api/push/vapid-public-key` implementieren: gibt `{ "publicKey": "..." }` zurück
- [x] 2.3 `POST /api/push/subscribe` implementieren: `INSERT OR REPLACE INTO push_subscriptions` mit User-ID aus JWT
- [x] 2.4 `DELETE /api/push/subscribe` implementieren: löscht Subscription anhand von `endpoint` und User-ID
- [x] 2.5 Funktion `SendToUsers(db, cfg, userIDs []int, title, body, url string)` implementieren: lädt Subscriptions, sendet via webpush-go, löscht bei HTTP 410

## 3. Backend: Carpooling-Handler erweitern

- [x] 3.1 `Upsert`-Handler: nach erfolgreichem DB-Write ermitteln welche User mit dem anderen `typ` für dasselbe Spiel eingetragen sind (exkl. aktueller User), Notification async senden (`go notifications.SendToUsers(...)`)
- [x] 3.2 `Delete`-Handler: vor dem DELETE den `typ` des zu löschenden Eintrags abfragen; wenn `typ = "biete"`, alle User mit "suche" für dasselbe Spiel ermitteln und async Notification senden
- [x] 3.3 Notification-Text zusammenstellen: Spielgegner und -datum aus DB laden für aussagekräftige Push-Texte
- [x] 3.4 Routen in `cmd/teamwerk/main.go` registrieren: `GET/POST/DELETE /api/push/...` unter Authenticated-Gruppe

## 4. Frontend: Service Worker

- [x] 4.1 `vite.config.ts` von `registerType: 'autoUpdate'` (generateSW) auf `strategy: 'injectManifest'` mit `srcDir: 'src', filename: 'sw.ts'` umstellen
- [x] 4.2 `web/src/sw.ts` anlegen: Workbox-Precaching-Import + `push`-EventListener (Notification anzeigen) + `notificationclick`-EventListener (`clients.openWindow(event.notification.data.url)`)
- [x] 4.3 Workbox-Caching-Regeln aus bisheriger vite.config.ts in `sw.ts` übertragen

## 5. Frontend: Subscribe-Hook

- [x] 5.1 `web/src/hooks/usePushSubscription.ts` erstellen: prüft `'PushManager' in window`; iOS-Detection via `/iphone|ipad|ipod/i.test(navigator.userAgent)` — auf iOS nur subscriben wenn `display-mode: standalone`, auf Android ohne Install-Check; Permission !== 'denied'; führt `pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: vapidPublicKey })` durch
- [x] 5.2 VAPID Public Key via `GET /api/push/vapid-public-key` laden und als `applicationServerKey` (Uint8Array) übergeben
- [x] 5.3 Subscription-Objekt an `POST /api/push/subscribe` senden; alle Fehler still schlucken
- [x] 5.4 `usePushSubscription`-Hook in `AppShell.tsx` einbinden (einmalig beim App-Start via `useEffect`)
