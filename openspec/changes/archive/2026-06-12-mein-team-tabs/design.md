## Context

Die `RosterSection`-Komponente in `MeinTeamPage.tsx` rendert aktuell Trainer, Spieler und Eltern als gestapelte Abschnitte innerhalb einer Karte. Bei Nutzern mit mehreren Teams wird die Seite dadurch unübersichtlich lang. Die Änderung ist rein frontend-seitig und erfordert keine API- oder Backend-Anpassungen.

## Goals / Non-Goals

**Goals:**
- Tab-Navigation (Team / Trainer / Eltern) innerhalb jeder `RosterSection`-Karte
- Lokaler Tab-Zustand pro Karte via `useState`
- Leere Tabs sichtbar mit Leertext

**Non-Goals:**
- Keine Änderungen an API oder Backend
- Kein persistenter Tab-Zustand (kein URL-Parameter, kein localStorage)
- Keine neue wiederverwendbare Tab-Komponente — der Tab-Bar bleibt inline in `RosterSection`

## Decisions

**Inline Tab-Bar statt generischer Komponente**  
Da es im Projekt keine bestehende Tab-Komponente gibt und die Anforderung auf eine einzige Seite beschränkt ist, wird der Tab-Bar direkt in `RosterSection` implementiert. Eine generische Abstraktion wäre Overengineering für diesen Scope.

**`useState` für aktiven Tab**  
Jede `RosterSection`-Instanz hält ihren eigenen Tab-State (`'team' | 'trainer' | 'eltern'`). Initial-State: `'team'`. Keine Synchronisation zwischen Karten nötig.

**Leere Tabs sichtbar**  
Tabs werden immer gerendert — auch wenn der Inhalt leer ist. Leertext: `— keine Einträge —`. So bleibt die Tab-Leiste stabil und der Nutzer sieht, dass die Kategorie existiert, aber leer ist.

**Styling: Pill-Tabs analog zu bestehenden Mustern**  
Aktiver Tab: `bg-brand-yellow text-brand-black font-medium`. Inaktiver Tab: `text-brand-text-muted hover:text-brand-text`. Kein Unterstrich-Stil, da die Karte bereits einen gelben `border-t-4` hat.

## Risks / Trade-offs

- [Kein Risiko] Rein additive Änderung, keine bestehende Logik wird entfernt
- [Trade-off] Tab-Zustand wird bei Navigation zurückgesetzt → akzeptabel, da kein Nutzer erwartet, den Tab-Zustand nach Seitenwechsel wiederzufinden
