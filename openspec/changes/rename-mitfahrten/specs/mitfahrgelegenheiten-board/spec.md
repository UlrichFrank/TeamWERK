## MODIFIED Requirements

### Requirement: Mitfahrten als chronologische Liste

Die Mitfahrten-Seite SHALL alle zukünftigen Spiele und Events in einer einzigen, fortlaufenden Liste anzeigen. Eine Aufteilung in Tabs nach Event-Typ SHALL NICHT mehr existieren.

#### Scenario: Liste zeigt alle Event-Typen zusammen

- **WHEN** ein Nutzer die Mitfahrten-Seite öffnet
- **THEN** sieht er alle zukünftigen Spiele und Events seines Teams in einer durchgehenden Liste — unabhängig vom Event-Typ (heim, auswärts, generisch)

#### Scenario: Keine Tab-Navigation vorhanden

- **WHEN** ein Nutzer die Seite öffnet
- **THEN** existieren keine Tab-Schaltflächen "Auswärtsspiele", "Heimspiele" oder "Events" — alle Filterung erfolgt über Pill-Buttons
