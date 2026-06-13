## Context

`MeinTeamPage` ist sowohl eine primäre Navigationsseite (Sidebar: `/mein-team`) als auch eine fokussierte Detail-Ansicht (Dashboard-Link: `/mein-team?team=X`). Im zweiten Fall fehlt ein Rückweg.

Bestehendes Muster: `TermineDetailPage` verwendet `navigate(-1)` mit `ChevronLeft`-Icon.

## Goals / Non-Goals

**Goals:**
- Zurück-Button auf MeinTeamPage wenn `focusTeamId != null` (d.h. `?team=X` in URL)
- Visuell und funktional konsistent mit TermineDetailPage

**Non-Goals:**
- Kein Zurück-Button wenn User direkt via Sidebar navigiert
- Keine Änderung an anderen Seiten

## Decisions

**`navigate(-1)` statt hartem `/dashboard`-Link:** Funktioniert unabhängig vom Herkunftspfad. Wenn künftig weitere Stellen auf `/mein-team?team=X` verlinken, passt sich der Rückweg automatisch an.

## Risks / Trade-offs

- [navigate(-1) bei direktem URL-Aufruf (z.B. neuer Tab)] → führt in Browser-History-Sackgasse. Da der Button nur bei `focusTeamId != null` erscheint, ist dieses Szenario marginal — der User kann trotzdem zur Sidebar greifen.
