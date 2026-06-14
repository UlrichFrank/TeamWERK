## Why

Nutzer verwenden TeamWERK auf iOS-Geräten und bevorzugen Apple Maps statt Google Maps — der aktuelle Hard-Link zu Google Maps erzwingt einen Umweg. Eine einstellbare Kartendienst-Präferenz verbessert die UX für alle Gerätetypen.

## What Changes

- Neues Datenbankfeld `maps_provider` in der `users`-Tabelle (`'auto'` | `'google'` | `'apple'`, Default: `'auto'`)
- `GET /api/profile/me` gibt `maps_provider` zurück
- `PUT /api/profile/me` akzeptiert und speichert `maps_provider`
- `MapsLink.tsx` baut die URL je nach Präferenz (auto = OS-Erkennung via User-Agent)
- `AuthContext` lädt `maps_provider` nach Login und stellt es app-weit bereit
- `ProfileMiscTab.tsx` zeigt einen Select mit drei Optionen

## Capabilities

### New Capabilities

- `maps-provider-preference`: Nutzerpräferenz für den Kartendienst (auto/google/apple), gespeichert in der DB, im Profil einstellbar, in MapsLink ausgewertet

### Modified Capabilities

- `user-profile`: `maps_provider` wird Teil des Profil-GET/PUT-Kontrakts

## Impact

- **DB:** Migration `users` +`maps_provider`
- **Backend:** `internal/auth` oder `internal/members` — Profil-Handler erweitern
- **Frontend:** `AuthContext.tsx`, `MapsLink.tsx`, `ProfileMiscTab.tsx`
- **Keine neuen Abhängigkeiten** — URL-Bau und User-Agent-Prüfung sind native Browser-APIs
