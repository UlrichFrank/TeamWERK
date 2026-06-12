# Spec: roster-section-tabs

## Overview
Die Mannschaftskarten auf der Mein-Team-Seite zeigen eine Tab-Navigation mit drei Kategorien: **Team**, **Trainer**, **Eltern**. Jede Karte verwaltet ihren Tab-Zustand unabhängig.

---

## Requirements

### Requirement: RosterSection zeigt Tab-Navigation
Jede Mannschaftskarte in der Mein-Team-Seite SHALL eine Tab-Leiste mit drei Tabs anzeigen: **Team**, **Trainer**, **Eltern**. Der aktive Tab ist beim Laden der Karte immer „Team".

#### Scenario: Standard-Tab beim Öffnen
- **WHEN** die Mein-Team-Seite geladen wird
- **THEN** zeigt jede Mannschaftskarte den Tab „Team" als aktiven Tab

#### Scenario: Tab-Wechsel
- **WHEN** der Nutzer auf einen anderen Tab klickt
- **THEN** wechselt der Inhalt der Karte auf die entsprechende Kategorie (Trainer-Liste oder Eltern-Liste)

#### Scenario: Unabhängiger Tab-Zustand bei mehreren Karten
- **WHEN** der Nutzer bei Karte A auf „Trainer" wechselt
- **THEN** bleibt Karte B auf ihrem eigenen Tab-Zustand (keine Synchronisation)

### Requirement: Leere Tabs zeigen Leertext
Ist eine Tab-Kategorie für ein Team leer (keine Einträge), SHALL der Tab trotzdem angezeigt und auswählbar sein. Der Inhalt zeigt dann den Text `— keine Einträge —`.

#### Scenario: Leerer Trainer-Tab
- **WHEN** ein Team keine Trainer hat und der Nutzer auf „Trainer" klickt
- **THEN** zeigt die Karte den Text `— keine Einträge —`

#### Scenario: Leerer Eltern-Tab
- **WHEN** ein Team keine Eltern hat und der Nutzer auf „Eltern" klickt
- **THEN** zeigt die Karte den Text `— keine Einträge —`
