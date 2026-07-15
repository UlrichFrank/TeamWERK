## Why

Der Chat („Nachrichten") ist heute nur über die Sidebar (`Verein → Nachrichten`, mit Unread-Badge) erreichbar. Das Dashboard — die Startseite unter `/` — bündelt die vier für den Nutzer wichtigsten Bereiche (Termine, Dienste, Fahrgemeinschaften, Team), zeigt aber nichts vom Chat. Ungelesene Nachrichten und Mitteilungen bleiben dadurch auf dem primären Einstieg unsichtbar.

## What Changes

- Neue Dashboard-Section **„Nachrichten"** — Optik und Struktur exakt wie die vier bestehenden Sections (kollabierbares `Accordion`, `bg-brand-surface-card`-Card mit `border-t-4 border-brand-yellow`, `DashboardRow`-Einträge).
- Die Section listet die ungelesenen Konversationen und Mitteilungen (neueste zuerst, begrenzt auf wenige Einträge) und verlinkt sie direkt in den passenden Chat-Tab.
- Fußzeile mit Link „Zum Chat →" (analog zu den `Link`-Footern der anderen Sections).
- Kein neuer Backend-Endpunkt — die Section nutzt die vorhandenen `GET /api/chat/conversations` und `GET /api/chat/broadcasts`.

## Capabilities

### New Capabilities

- `dashboard-nachrichten`: Das Dashboard zeigt eine Section mit ungelesenen Chat-Konversationen und Mitteilungen samt Direkt-Link in den Chat.

## Impact

- **Frontend:** `web/src/pages/DashboardPage.tsx` — neue Section + Datenlade-Logik; `AppShell.tsx` bleibt unverändert (Badge dort ist eigenständig).
- **Backend:** keine Änderung (nur Lesen bestehender Chat-Endpunkte).
- **Live-Update:** Section abonniert `useChatEvents` (bzw. lädt beim Dashboard-Mount), damit neue Nachrichten/Mitteilungen die Liste aktualisieren.
