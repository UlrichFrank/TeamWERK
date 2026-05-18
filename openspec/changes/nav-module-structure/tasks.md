## 1. Datenstruktur

- [x] 1.1 `navItems`-Array in `AppShell.tsx` durch `navModules: NavModule[]` ersetzen — Interface mit `label: string` und `items: { to, label, roles }[]`; 3 Module definieren: Mitglieder, Dienste, Administration mit den entsprechenden Einträgen

## 2. Klapp-Logik

- [x] 2.1 `openModules`-State als `Record<string, boolean>` anlegen, initialisiert aus `localStorage` (Key `nav-open-<label>`), Fallback `true` für alle Module
- [x] 2.2 `toggleModule(label: string)`-Funktion implementieren: State toggeln + `localStorage` synchron aktualisieren

## 3. Render-Logik

- [x] 3.1 Render-Schleife umbauen: pro Modul sichtbare Items filtern (`item.roles.includes(user.role)`); Modul nur rendern wenn mindestens ein Item sichtbar ist
- [x] 3.2 Modul-Header als `<button>` rendern: `px-4 py-2 w-full text-left flex items-center justify-between text-xs font-semibold uppercase tracking-wider`; Text `text-white` wenn ein Kind aktiv ist (via `useMatch` oder Pfad-Check), sonst `text-white/50`; Chevron-Icon rotiert bei eingeklapptem Zustand (`▾` / `▸`)
- [x] 3.3 Untereinträge nur rendern wenn Modul aufgeklappt (`openModules[label]`); bestehende `NavLink`-Klassen beibehalten, nur `pl-7` als zusätzliche Einrückung hinzufügen
- [x] 3.4 TypeScript-Check: `npx tsc --noEmit` muss fehlerfrei durchlaufen
