## ADDED Requirements

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
