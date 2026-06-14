## Context

`MapsLink.tsx` ist das einzige Component, das Maps-URLs baut — aktuell immer `https://maps.google.com/?q=...`. Es gibt keine Nutzerpräferenz und keinen OS-spezifischen Link. iOS-Nutzer landen im Browser statt in Apple Maps.

AuthContext hält heute nur `{ email, role }` aus dem JWT-Payload. Profildaten (z.B. Benachrichtigungseinstellungen) werden je nach Seite ad-hoc via `GET /api/profile/me` geladen.

## Goals / Non-Goals

**Goals:**
- Nutzerpräferenz `maps_provider` (`auto` | `google` | `apple`) in DB speichern
- Präferenz einmalig nach Login laden und in AuthContext bereitstellen
- `MapsLink.tsx` wertet Präferenz aus; bei `auto` → OS-Erkennung via User-Agent
- `ProfileMiscTab.tsx` zeigt Auswahlfeld mit drei Optionen

**Non-Goals:**
- Kein OpenStreetMap
- Keine Kartenvorschau innerhalb der App
- Kein automatisches Nachladen bei Präferenzänderung ohne Seitenrefresh (bewusst simpel)

## Decisions

### 1. Speicherort: DB-Spalte in `users`, nicht localStorage

**Gewählt:** `ALTER TABLE users ADD COLUMN maps_provider TEXT NOT NULL DEFAULT 'auto' CHECK(maps_provider IN ('auto','google','apple'))`

**Alternativen:**
- `localStorage`: kein Server-Round-Trip, aber gerätespezifisch — Präferenz geht beim Gerätewechsel verloren
- Eigene Preferences-Tabelle: Overkill für ein einzelnes Feld; `users` ist der richtige Ort für User-Einstellungen

### 2. Präferenz in AuthContext, nicht in separatem Context

**Gewählt:** `AuthContext` wird um `mapsProvider: 'auto' | 'google' | 'apple'` erweitert. Nach Login / Token-Refresh wird `GET /api/profile/me` aufgerufen und der Wert gesetzt.

**Alternativen:**
- Separater `ProfileContext`: sauberere Trennung, aber mehr Boilerplate und ein zweiter Provider in `App.tsx`
- Props-Drilling: nicht praktikabel, da `MapsLink` tief im Baum sitzt (EventInfoModal etc.)
- `localStorage` als Cache: würde funktionieren, ist aber eine zweite Source of Truth neben der DB

**Warum AuthContext:** Das Muster „nach Login einmal Profil laden" passt gut — ähnlich wie bei Reminder-Preference. Kein extra Fetch, kein extra Provider.

### 3. Auto-Erkennung: User-Agent, nicht `navigator.platform`

`navigator.platform` ist deprecated. Stattdessen:
```ts
const isApplePlatform = /iPhone|iPad|iPod|Macintosh/.test(navigator.userAgent)
```
Macintosh deckt macOS ab (Safari öffnet Apple Maps nativ). Auf anderen Plattformen → Google Maps.

### 4. URL-Schema: HTTPS statt nativer URI-Schemes

`maps://` funktioniert nur in nativen iOS-Apps, nicht im Browser/PWA-Kontext.
`https://maps.apple.com/?q=...` öffnet auf iOS Safari und macOS Safari die Apple Maps App;
auf anderen Plattformen fällt es auf die Web-Version zurück — kein Fehler.

## Risks / Trade-offs

- **User-Agent-Sniffing ist unzuverlässig** → Nutzer können die Präferenz manuell überschreiben; `auto` ist nur ein sinnvoller Default, kein Muss
- **Profil-Fetch schlägt fehl** → `mapsProvider` bleibt auf Default `'auto'`; App funktioniert weiter
- **Kein Realtime-Update** → Präferenzänderung im Profil wirkt erst nach Page-Reload (AuthContext hält den Wert im Memory); für ein Nutzerpräferenz-Feld akzeptabel

## Migration Plan

1. Migration `0NN_maps-provider.up.sql`: `ALTER TABLE users ADD COLUMN maps_provider TEXT NOT NULL DEFAULT 'auto' CHECK(...)`
2. `make deploy` führt `migrate up` automatisch aus — kein manueller Schritt nötig
3. Rollback: `0NN_maps-provider.down.sql` mit `ALTER TABLE users DROP COLUMN maps_provider` (SQLite ≥ 3.35 — modernc.org/sqlite unterstützt das)

## Open Questions

- Keine offenen Fragen; Design ist vollständig spezifiziert.
