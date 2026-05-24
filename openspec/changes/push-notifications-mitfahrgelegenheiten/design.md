## Context

TeamWERK ist eine PWA (vite-plugin-pwa + Workbox Service Worker) auf einem IONOS VPS mit 1 GB RAM. Die Mitfahrgelegenheiten-Funktion hat Biete/Suche-Einträge pro Spiel. Bisher keine Echtzeit-Benachrichtigungen. Die Web Push API (VAPID-Standard) erlaubt Browser-Push ohne eigene Push-Server-Infrastruktur — der Browser leitet über Apple Push Notification Service (APNS) bzw. Google FCM weiter.

## Goals / Non-Goals

**Goals:**
- Authentifizierte User erhalten Push Notifications bei relevanten Mitfahrgelegenheits-Ereignissen
- iOS 16.4+ (als installierte PWA) und Android Chrome werden unterstützt
- Kein sichtbares Onboarding — Subscribe passiert transparent beim App-Start
- RAM-Footprint bleibt gering (keine zusätzlichen Daemons, kein Redis)

**Non-Goals:**
- Push bei gelöschten "suche"-Einträgen (nur "biete"-Löschungen triggern Push)
- Direkte User-zu-User-Nachrichten / Chat
- Push für andere Domänen (Dienste, Spielplan etc.) — vorerst nur Mitfahrgelegenheiten
- Notification-Einstellungen im UI

## Decisions

### 1. VAPID statt proprietärer Push-Dienst

**Entscheidung:** Web Push mit VAPID (Voluntary Application Server Identification).  
**Warum:** Kein externer Account nötig (kein Firebase, kein OneSignal). Der Server signiert Pushes selbst mit einem ECDH-Schlüsselpaar. Apple und Google akzeptieren VAPID direkt — es gibt keinen Middleware-Dienst der ausfallen kann.  
**Alternative:** OneSignal/Firebase würde Onboarding vereinfachen, aber eine externe Abhängigkeit und Datenweitergabe bedeuten.

### 2. `github.com/SherClockHolmes/webpush-go` als Go-Library

**Entscheidung:** Diese Library übernimmt VAPID-Signing, Payload-Verschlüsselung (RFC 8291) und HTTP-Anfragen an Push-Endpoints.  
**Warum:** Einzige ausgereifte Go-Library für Web Push, 1.2k Stars, aktiv gepflegt, minimale Dependencies.  
**RAM:** Vernachlässigbar, keine Goroutinen im Hintergrund.

### 3. Push-Versand synchron im HTTP-Handler (fire-and-forget)

**Entscheidung:** Push-Nachrichten werden direkt nach dem DB-Write in einer Goroutine gesendet (`go sendPushNotifications(...)`), ohne Queue.  
**Warum:** Kein Redis/Message-Queue verfügbar (1 GB RAM, kein externer Dienst). Bei fehlgeschlagenem Push (toter Endpoint) wird der DB-Eintrag still gelöscht (Standard-Verhalten: HTTP 410 Gone → Subscription entfernen).  
**Risiko:** Bei VPS-Neustart laufende Goroutinen verlieren sich — akzeptabel, da Push-Volumen gering (Vereins-App).

### 4. Subscription-Speicherung in SQLite

**Entscheidung:** Tabelle `push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at)`, ein User kann mehrere Subscriptions haben (Geräte).  
**Warum:** Passt zum bestehenden Datenbankmodell, kein extra Dienst nötig.

### 5. Service Worker via vite-plugin-pwa Custom SW

**Entscheidung:** Push-Event-Handler in `web/src/sw.ts` (custom Service Worker), importiert via `injectManifest`-Modus in vite-plugin-pwa.  
**Warum:** Der bisherige `generateSW`-Modus erlaubt keinen eigenen Push-Handler. Wechsel auf `injectManifest` gibt volle Kontrolle, Workbox-Caching bleibt erhalten.

### 6. Plattformspezifischer Silent-Skip auf iOS (nicht als PWA installiert)

**Entscheidung:** Der `display-mode: standalone`-Check wird **nur auf iOS** angewendet. Android Chrome subscribed ohne Installationspflicht. Kein Banner, kein Hinweis.  
**Warum:** iOS Push funktioniert ausschließlich in installierten PWAs (ab iOS 16.4). Android Chrome hingegen unterstützt Web Push nativ im Browser ohne PWA-Install. Ein universeller `standalone`-Check würde Android-User still ausschließen.  
**Implementierung:**
```ts
const isIOS = /iphone|ipad|ipod/i.test(navigator.userAgent)
if (isIOS && !window.matchMedia('(display-mode: standalone)').matches) return
```
**Alternative:** Ein universeller `standalone`-Check wäre einfacher, würde aber Android-Chrome-User ohne PWA-Installation ausschließen — nicht gewünscht.

## Risks / Trade-offs

- **Tote Endpoints:** Push-Endpoints können ungültig werden (Reinstall, Browser-Reset). → Bei HTTP 410 vom Push-Service wird die Subscription aus der DB gelöscht.
- **iOS 16.4 Pflicht:** Ältere iOS-Versionen bekommen gar keine Pushes. → Silent fail, kein Impact auf Funktionalität.
- **Kein Retry:** Fehlgeschlagene Sends (Network Error) werden nicht wiederholt. → Akzeptabel bei Nicht-kritischen Benachrichtigungen.
- **Vite-Plugin-Modus-Wechsel:** `generateSW` → `injectManifest` erfordert eine neue `sw.ts`-Datei. → Workbox-Imports müssen manuell eingebunden werden, bestehende Caching-Regeln bleiben erhalten.

## Migration Plan

1. VAPID-Schlüsselpaar einmalig generieren: `go run ./cmd/teamwerk gen-vapid`
2. Keys in `.env` und auf VPS in `/etc/teamwerk/env` eintragen
3. Migration 013 ausführen (automatisch via `make deploy`)
4. Neues Binary deployen
5. Kein Rollback-Risiko: neue Tabelle, neue Routen — bestehende Features unberührt

## Open Questions

_(keine)_
