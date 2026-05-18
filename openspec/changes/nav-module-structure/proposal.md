## Why

Die Sidebar-Navigation wächst mit jedem Feature und ist aktuell eine flache Liste ohne erkennbare Struktur. Neue Nutzer finden sich schwerer zurecht, und künftige Erweiterungen (z. B. Spielplan, Statistiken) lassen sich nicht sauber einordnen. Eine Modul-Hierarchie schafft Orientierung und skaliert für neue Funktionen.

## What Changes

- Die flache `navItems`-Liste in `AppShell.tsx` wird durch eine hierarchische Modul-Struktur ersetzt
- 3 Module mit ein-/ausklappbaren Untereinträgen:
  - **Mitglieder** – Mitglieder, Mein Profil
  - **Dienste** – Dienstbörse, Dienstkonten, Dienst-Planung
  - **Administration** – Beitrittsanfragen, Verein, Teams, Nutzer, Diensttypen
- Module bleiben beim Seitenstart aufgeklappt; der Zustand wird im `localStorage` gespeichert
- Module ohne sichtbare Untereinträge (wegen Rollenfilter) werden ausgeblendet

## Capabilities

### New Capabilities

- `nav-module-sidebar`: Kollabierbare Modul-Navigation in der Sidebar mit Rollen-basierter Sichtbarkeit

### Modified Capabilities

## Impact

- **Frontend:** `AppShell.tsx` — einzige betroffene Datei, keine API-Änderungen
- **Keine Backend-Änderungen**
- **Keine neuen Routen**
