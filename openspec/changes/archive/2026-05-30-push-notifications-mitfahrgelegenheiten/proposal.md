## Why

Mitglieder, die Mitfahrgelegenheiten anbieten oder suchen, erfahren erst beim nächsten manuellen App-Aufruf von passenden Einträgen anderer. Push Notifications ermöglichen Echtzeit-Benachrichtigungen direkt auf iOS/Android ohne eigene Infrastruktur — die PWA-Basis ist bereits vorhanden.

## What Changes

- Neue DB-Tabelle `push_subscriptions` speichert Web Push Endpoints pro User und Gerät
- VAPID-Schlüsselpaar wird einmalig generiert und in `.env` hinterlegt
- Neues Go-Package `internal/notifications` kapselt VAPID-Versand und Subscription-Verwaltung
- 3 neue API-Routen: VAPID Public Key abrufen, Subscription registrieren/löschen
- Service Worker erhält Push-Event-Handler und Notification-Click-Handler
- Frontend abonniert Push-Notifications beim App-Start silent (kein Onboarding-Banner)
- Carpooling-Handler löst nach Upsert und Delete Notifications an betroffene User aus

**Trigger-Logik:**
- POST "biete" → alle User mit "suche" für dasselbe Spiel werden benachrichtigt
- POST "suche" → alle User mit "biete" für dasselbe Spiel werden benachrichtigt
- DELETE "biete" → alle User mit "suche" für dasselbe Spiel werden benachrichtigt

## Capabilities

### New Capabilities

- `web-push-subscriptions`: Verwaltung von Web Push Subscriptions (VAPID Keys, Endpoint-Speicherung, Subscribe/Unsubscribe API)
- `carpooling-notifications`: Auslösen von Push Notifications bei relevanten Mitfahrgelegenheiten-Ereignissen

### Modified Capabilities

_(keine bestehenden Specs betroffen)_

## Impact

- **Backend:** Neues Package `internal/notifications`, neue Migration `013_push_subscriptions`, Dependency `github.com/SherClockHolmes/webpush-go`, Änderungen in `internal/carpooling/handler.go`
- **Frontend:** Erweiterung des Service Workers (Push-Handler), neuer `usePushSubscription`-Hook in `web/src/`
- **Deployment:** 3 neue `.env`-Variablen: `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL`
- **Rollen:** Alle authentifizierten User (spieler, elternteil, trainer, admin)
