## Context

Zwei Ursachen für horizontalen Overflow auf Mobile:

**Ursache 1 — AppShell-Layout:**  
Der Flex-Content-Container `<div className="flex-1 flex flex-col min-h-0">` hat `min-h-0` für vertikales Scrolling, aber kein `min-w-0`. In einem Row-Flex-Container darf ein Flex-Item ohne `min-w-0` breiter werden als der Viewport — der Root-Container clippt mit `overflow-hidden`, statt zu scrollen. Das `overflow-auto` auf `<main>` kann dann nicht mehr greifen.

**Ursache 2 — Nicht-responsive Page-Header:**  
Vier Seiten haben `<div className="flex items-center justify-between">` als Page-Header ohne Mobile-Zeilenumbruch. Das korrekte Muster ist in `MembersPage.tsx` und `TerminePage.tsx` bereits umgesetzt: `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0`.

## Goals / Non-Goals

**Goals:**
- Kein horizontaler Overflow auf iPhone SE (375px) oder größer auf den betroffenen Seiten
- AppShell-Root-Ursache beseitigen

**Non-Goals:**
- Keine neue `PageHeader`-Komponente — 4 Stellen rechtfertigen keine Abstraktion
- Kein Umbau zu Mobile-Card-Layouts für Tabellen (separater Change)
- Keine Backend-Änderungen

## Decisions

### D1: `min-w-0` in AppShell

`<div className="flex-1 flex flex-col min-h-0">` → `<div className="flex-1 flex flex-col min-h-0 min-w-0">`. Eine Klasse, behebt die Root-Ursache. Danach kann `overflow-auto` auf `<main>` sowohl vertikal als auch horizontal scrollen.

### D2: Responsive Header-Pattern für alle 4 betroffenen Seiten

```
// Vorher (alle 4 Seiten):
<div className="flex items-center justify-between mb-6">

// Nachher:
<div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0 mb-6">
```

Controls-div erhält `flex flex-wrap gap-2` damit mehrere Buttons/Felder auf Mobile umbrechen.

### D3: Tabellen-Wrapper in AdminUsersPage

AdminUsersPage hat zwei `<table>`-Elemente direkt in Tabellen-Containern ohne `overflow-x-auto`. Der Card-Container erhält `overflow-x-auto` als Wrapper-Klasse (gemäß CLAUDE.md-Konvention „Card — Tabellen-Container").

## Risks / Trade-offs

- Reine CSS-Klassen-Änderung, kein Verhaltenscode berührt → minimales Risiko
- Page-Header auf Mobile nimmt mehr vertikale Höhe ein (Stapel-Layout) → akzeptabel, Scrollen auf Mobile ist Standard
