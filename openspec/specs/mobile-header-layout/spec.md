# mobile-header-layout Specification

## Capability: mobile-header-layout

Seitenheader (h1 + Controls) auf Admin-Seiten müssen auf allen Mobilbreiten (≥ 320px) ohne horizontalen Overflow darstellbar sein.

## Purpose

Diese Spezifikation beschreibt die Capability `mobile-header-layout`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Vertikale Stapelung auf Mobile

Auf Viewports < 640px (`sm:`-Breakpoint) stapeln sich `<h1>` und die Controls-Gruppe vertikal (`flex-col`). Ab 640px gilt das horizontale Desktop-Layout (`flex-row justify-between`).

**Betroffene Seiten:** AdminUsersPage, AdminDutyTypesPage, AdminDutyTemplatesPage

**Referenz-Implementierung:** `MembersPage` — bereits korrekt umgesetzt.
