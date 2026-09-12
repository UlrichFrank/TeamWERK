## Purpose

Modale Dialoge sind mit Tastatur und Screenreader bedienbar.

## ADDED Requirements

### Requirement: Dialoge sind zugänglich

Jeder modale Dialog MUST `role="dialog"`, `aria-modal="true"` und eine Beschriftung (`aria-labelledby` oder `aria-label`) tragen. Beim Öffnen MUST der Fokus in den Dialog wandern, Tab und Shift+Tab MUST innerhalb des Dialogs zyklisch bleiben, und beim Schließen MUST der Fokus zum auslösenden Element zurückkehren. Escape schließt weiterhin.

#### Scenario: Tastaturnutzer bleibt im Dialog
- **WHEN** ein Dialog offen ist und der Nutzer wiederholt Tab drückt
- **THEN** wandert der Fokus nur zwischen den fokussierbaren Elementen des Dialogs

#### Scenario: Fokus kehrt zurück
- **WHEN** der Nutzer den Dialog schließt
- **THEN** liegt der Fokus wieder auf dem Element, das den Dialog geöffnet hat
