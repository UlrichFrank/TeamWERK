## Context

AppShell.tsx definiert die Navigation über ein `navModules`-Array mit NavItems. Jedes Item hat `roles: string[]` — eine Whitelist. Leeres Array bedeutet „alle sehen es". Es gibt keinen Ausschluss-Mechanismus.

Aktuell: „Mein Profil" hat `roles: ['elternteil', 'spieler']` — Trainer fehlt. „Mitglieder" hat `roles: ['admin', 'vorstand', 'trainer']` — Trainer soll aber Kader verwenden. „Kader" hat `roles: ['admin', 'vorstand']` — Trainer gesperrt.

Backend: Kader-API-Routen liegen in der `RequireRole("admin", "vorstand")`-Gruppe in `main.go`.

## Goals / Non-Goals

**Goals:**
- „Mein Profil" für alle Rollen außer `admin` sichtbar machen
- `excludeRoles`-Eigenschaft als ergänzenden Mechanismus einführen (kein Ersatz für `roles`)
- „Mitglieder" auf `admin`/`vorstand` einschränken
- „Kader" für `trainer` in Navigation und Backend freischalten

**Non-Goals:**
- Kein Refactoring der gesamten Rollenlogik
- Keine Filterung der Kader-Anzeige nach Trainer-Zugehörigkeit (Trainer sieht alle Kader)

## Decisions

**`excludeRoles` ergänzt `roles`, ersetzt es nicht.**
Filter-Logik: Item sichtbar wenn `(roles.length === 0 || roles.includes(role)) && !excludeRoles?.includes(role)`.
Rationale: Bestehende Items bleiben unverändert. `excludeRoles` ist optional — undefined bedeutet kein Ausschluss.

**Trainer erhält vollen Kader-Zugriff (wie Vorstand).**
Kader-Routes werden in die `RequireRole("admin", "vorstand", "trainer")`-Gruppe verschoben. Keine feingranulare Einschränkung einzelner Operationen — der Trainer verwaltet seinen Kader eigenverantwortlich.

## Risks / Trade-offs

[Neue Rollen in Zukunft] → `excludeRoles: ['admin']` schließt nur `admin` aus; neue Rollen sind automatisch eingeschlossen. Das ist gewünscht, muss aber beim Hinzufügen neuer Rollen bedacht werden.

[Trainer sieht alle Kader, nicht nur eigene] → Akzeptiertes Trade-off per Anforderung. Kein Backend-Filtering notwendig.
