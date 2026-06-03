## Context

Drei Admin-Seiten haben identisches Muster: `<div className="flex items-center justify-between">` mit `<h1>` links und Controls (Suchfeld + Button oder nur Button) rechts. Auf mobilen Breiten (≤ 375px) überläuft diese Zeile horizontal, weil kein Zeilenumbruch erlaubt ist.

Das korrekte Muster ist bereits in `MembersPage.tsx` umgesetzt: `flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0`. Auf Mobile stapeln sich Titel und Controls vertikal; ab 640px gilt das Desktop-Layout.

## Goals / Non-Goals

**Goals:**
- Drei Seiten auf dasselbe Header-Muster wie `MembersPage` bringen
- Kein horizontaler Overflow auf iPhone 7 (375px) oder größer

**Non-Goals:**
- Keine neue Komponente (`PageHeader`) extrahieren — drei Stellen rechtfertigen keine Abstraktion
- Keine Kalender- oder Tabellen-Fixes (separater Change)
- Keine Backend-Änderungen

## Decisions

**Inline-Fix statt neuer Komponente:** Die drei Seiten haben leicht unterschiedliche Control-Anordnungen (AdminUsersPage: Suche + Button; AdminDutyTypesPage: nur Button; AdminDutyTemplatesPage: nur Button). Eine generische `PageHeader`-Komponente würde Props-Drilling erfordern und wäre Over-Engineering für drei Stellen.

**Exaktes Muster von MembersPage übernehmen:**
```
äußerer div:  flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0
h1:           unverändert (text-2xl font-bold)
Controls-div: flex gap-2 (ggf. flex-wrap für mehrere Buttons)
```

## Risks / Trade-offs

- Kein nennenswertes Risiko — reine CSS-Klassen-Änderung, kein Verhaltenscode berührt
- Auf Mobile nimmt der Header mehr vertikale Höhe ein (Titel + Controls stapeln sich) → akzeptabel, da Scrolling auf Mobile normal ist
