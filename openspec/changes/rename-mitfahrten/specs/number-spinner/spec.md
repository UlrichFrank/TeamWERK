## MODIFIED Requirements

### Requirement: Komponente wird an allen drei Einsatzorten verwendet

Die Komponente SHALL in `ProfileMiscTab.tsx` (Sitzplätze), `MitfahrtenPage.tsx` (Freie Plätze) und `AdminSettingsPage.tsx` (Halbzeit und Pause) eingesetzt werden.

#### Scenario: Einsatz mit step=5 bei Altersklassen

- **WHEN** `<NumberSpinner value={20} min={1} step={5} onChange={fn} />` in der Altersklassen-Tabelle gerendert wird
- **THEN** erhöht/verringert ein Button-Klick den Wert um 5
