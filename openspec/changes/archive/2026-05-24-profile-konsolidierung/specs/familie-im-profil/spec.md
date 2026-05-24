## MODIFIED Requirements

### Requirement: Mitgliedsdaten-Tab zeigt Stammdaten und offene Anfragen
Der Mitgliedsdaten-Tab SHALL ausschließlich lesende Informationen enthalten: Stammdaten des verknüpften Mitglieds (read-only) und eine Übersicht offener Änderungsanfragen. Die bisherigen Editier-Sektionen „Name ändern" und „IBAN ändern" entfallen — diese Funktionalität ist in den Profil-Tab gewandert.

#### Scenario: Mitgliedsdaten-Tab ohne Editier-Sektionen
- **WHEN** ein verknüpftes Mitglied den Mitgliedsdaten-Tab öffnet
- **THEN** sind keine Eingabefelder oder „Änderung anfordern"-Buttons für Name oder IBAN vorhanden

#### Scenario: Offener Profil-Draft im Mitgliedsdaten-Tab sichtbar
- **WHEN** ein Mitglied einen offenen Profil-Bundle-Draft hat und den Mitgliedsdaten-Tab öffnet
- **THEN** wird eine Sektion „Ausstehende Anfrage" angezeigt mit einer lesbaren Auflistung aller beantragten Änderungen (Felder mit altem → neuem Wert) und einem „Zurückziehen"-Button

#### Scenario: Zurückziehen löscht den Draft
- **WHEN** ein Mitglied auf „Zurückziehen" klickt
- **THEN** wird `DELETE /members/{id}/change-drafts/{draftId}` aufgerufen, der Draft entfernt, und das Profil-Formular wieder editierbar

#### Scenario: Kein offener Draft — keine Anfragen-Sektion
- **WHEN** ein Mitglied keine offenen Drafts hat
- **THEN** zeigt der Mitgliedsdaten-Tab nur die Stammdaten-Sektion ohne Anfragen-Bereich
