## Context

`AppShell.tsx` rendert aktuell eine flache Liste von `NavLink`-Elementen. Die Rollen-Filterung erfolgt per `item.roles.includes(user.role)`. Die Sidebar ist `w-56 bg-brand-blue text-white`.

## Goals / Non-Goals

**Goals:**
- Modul-Struktur mit 3 Modulen, die ein- und ausklappbar sind
- Klapp-Zustand pro Modul in `localStorage` persistieren (Key: `nav-open-<modulname>`)
- Module ohne sichtbare Untereinträge komplett ausblenden
- Aktiver Moduleintrag hebt das Modul visuell hervor (z. B. Modulname fett, wenn Kind aktiv)
- Stil: Modulname als Gruppenüberschrift (leicht größer/fett, mit Pfeil-Icon), Untereinträge eingerückt

**Non-Goals:**
- Mehrfach verschachtelte Navigation (nur 2 Ebenen)
- Animierte Übergänge (kein Framer Motion o. Ä.)
- Persistenz des Zustands im Backend

## Decisions

**1. Datenstruktur**

```ts
interface NavModule {
  label: string
  items: { to: string; label: string; roles: string[] }[]
}
```

Module werden als `NavModule[]`-Array definiert; Rollen-Sichtbarkeit wird pro Item und pro Modul gefiltert.

**2. Klapp-Zustand**

`useState` pro Modul, initialisiert aus `localStorage`. Fallback: alle aufgeklappt. Beim Toggle wird `localStorage` synchron aktualisiert — kein useEffect nötig.

**3. Modul-Header-Stil**

Klickbarer `<button>` mit `px-4 py-2 text-xs font-semibold uppercase tracking-wider text-white/60` für den Modulnamen + Pfeil-Chevron (`▾`/`▸`). Kein eigener Farbwechsel beim Hover des Modulnamens (nur Icon-Rotation reicht).

**4. Aktiver Zustand**

Wenn ein Kind-`NavLink` aktiv ist (`useMatch` oder isActive in NavLink-Callback), gilt das Modul als aktiv — Modulname wird `text-white` statt `text-white/60`. Die Untereinträge nutzen weiterhin `isActive` aus NavLink.

**5. Keine neue Abhängigkeit**

Nur React `useState` + `localStorage`. Keine externe Bibliothek.

## Risks / Trade-offs

- **localStorage im SSR:** Kein Problem, da die App ein reines SPA ist (kein Server-Rendering)
- **Klapp-Zustand versteckt aktive Seite:** Mitigiert durch localStorage-Persistenz — der Zustand bleibt beim Reload erhalten; bei erstem Besuch sind alle aufgeklappt

## Migration Plan

1. `AppShell.tsx` refactoren: `navItems` → `navModules`, Render-Logik erweitern
2. Kein Deploy-Schritt nötig, nur Frontend-Build
