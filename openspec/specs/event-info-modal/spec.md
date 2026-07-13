# event-info-modal Specification

## Purpose

Diese Spezifikation beschreibt die Capability `event-info-modal`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Button „In Terminen öffnen" im EventInfoModal

Das `EventInfoModal` SHALL einen Button **„In Terminen öffnen"** anzeigen, der den Nutzer zur `/termine`-Seite mit Deep-Link auf den aktuellen Termin navigieren lässt. Der Button verwendet die Primary-Button-Klasse (`bg-brand-yellow text-brand-black …`) und erscheint im Footer-Bereich des Modals, links vom Schließen-Button.

- Bei einem Spieltag → Link `/termine?focus=game-<id>`
- Bei einem Training → Link `/termine?focus=training-<id>`

Beim Klick MUSS das Modal sich schließen und die Navigation via React-Router `navigate(...)` erfolgen (kein Full Page Reload).

Der Button SHALL für alle Rollen sichtbar sein, die `/termine` aufrufen dürfen — also alle eingeloggten Nutzer (gleiche Sichtbarkeit wie die Termine-Seite selbst).

#### Scenario: Button für Spieltag
- **WHEN** das `EventInfoModal` für einen Spieltag mit ID 17 geöffnet ist
- **THEN** ist ein Button „In Terminen öffnen" sichtbar
- **WHEN** der User auf den Button klickt
- **THEN** schließt sich das Modal
- **THEN** navigiert die App zu `/termine?focus=game-17`

#### Scenario: Button für Training
- **WHEN** das `EventInfoModal` für eine Trainingseinheit mit ID 42 geöffnet ist
- **WHEN** der User auf „In Terminen öffnen" klickt
- **THEN** navigiert die App zu `/termine?focus=training-42`

#### Scenario: Button nicht für Dienste oder andere Event-Typen
- **WHEN** das `EventInfoModal` einen Eintrag anzeigt, der weder Spiel noch Training ist (z.B. reiner Kalendereintrag, Dienst-Eintrag)
- **THEN** ist der Button „In Terminen öffnen" nicht sichtbar

### Requirement: Button „In Diensten öffnen" im EventInfoModal

Das `EventInfoModal` SHALL für Spiele einen Button **„In Diensten öffnen"** im Footer anzeigen, der zur Dienstbörse `/dienste` navigiert. Der Button MUSS:

- **nur für `type === 'game'`** erscheinen (nicht für Trainings, nicht für Absences),
- **für alle eingeloggten Rollen sichtbar** sein (Navigation, nicht Bearbeitung),
- **`disabled` sein**, wenn das Spiel keine Dienst-Slots hat (`slot_count === 0`),
- die Primary-Button-Klasse (`bg-brand-yellow text-brand-black …`) verwenden und **rechts** vom bestehenden „In Terminen öffnen"-Button erscheinen,
- beim Klick das Modal schließen und via React-Router `navigate('/dienste')` (kein Full Page Reload, kein Focus-Parameter) navigieren.

#### Scenario: Spiel mit Slots — Button aktiv
- **WHEN** das `EventInfoModal` für ein Spiel mit `slot_count > 0` geöffnet ist
- **THEN** ist der Button „In Diensten öffnen" sichtbar und aktiv
- **WHEN** der User auf den Button klickt
- **THEN** schließt sich das Modal
- **THEN** navigiert die App zu `/dienste`

#### Scenario: Spiel ohne Slots — Button disabled
- **WHEN** das `EventInfoModal` für ein Spiel mit `slot_count === 0` geöffnet ist
- **THEN** ist der Button „In Diensten öffnen" sichtbar, aber deaktiviert (`disabled`)

#### Scenario: Training — Button nicht sichtbar
- **WHEN** das `EventInfoModal` für ein Training geöffnet ist
- **THEN** ist der Button „In Diensten öffnen" nicht sichtbar

#### Scenario: Absence — Button nicht sichtbar
- **WHEN** das `EventInfoModal` für eine Abwesenheit geöffnet ist
- **THEN** ist der Button „In Diensten öffnen" nicht sichtbar

### Requirement: Header-Icon „Dienste" (ClipboardList) nur bei Bearbeitungsrechten

Das `EventInfoModal` SHALL das ClipboardList-Icon oben rechts (links neben dem Schließen-`X`) nur anzeigen, wenn der aufrufende Kontext einen `onDienste`-Callback übergibt. `KalenderPage` MUSS den Callback nur setzen, wenn der Nutzer Bearbeitungsrechte auf dem Spiel hat (analog zum bestehenden `onEdit`-Pfad). Für Nicht-Bearbeiter bleibt der Zugriff auf die Dienstbörse ausschließlich über den Footer-Button „In Diensten öffnen" erhalten.

#### Scenario: Admin sieht ClipboardList
- **WHEN** ein Admin ein Spiel im `EventInfoModal` öffnet
- **THEN** ist das ClipboardList-Icon oben rechts sichtbar

#### Scenario: Vorstand sieht ClipboardList
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` ein Spiel im `EventInfoModal` öffnet
- **THEN** ist das ClipboardList-Icon oben rechts sichtbar

#### Scenario: Trainer sieht ClipboardList
- **WHEN** ein Trainer eines am Spiel beteiligten Teams das `EventInfoModal` öffnet
- **THEN** ist das ClipboardList-Icon oben rechts sichtbar

#### Scenario: Spieler sieht kein ClipboardList
- **WHEN** ein Spieler ohne privilegierte Vereinsfunktion ein Spiel im `EventInfoModal` öffnet
- **THEN** ist das ClipboardList-Icon oben rechts nicht sichtbar
- **THEN** ist der Footer-Button „In Diensten öffnen" (sofern das Spiel Slots hat) trotzdem sichtbar und aktiv
