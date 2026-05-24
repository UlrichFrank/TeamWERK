## MODIFIED Requirements

### Requirement: Duty account per family
Das Duty-Account-System bleibt unverändert — Ist-Wert, Claim-Logik und Export bleiben identisch. Geändert wird ausschließlich die Berechnung des `soll`-Werts für die Rolle `elternteil` im Dashboard-Endpoint.

**Vorher:** `soll = 5 × COUNT(family_links WHERE parent_user_id = user_id)`

**Nachher:** Dynamische Formel basierend auf Kader-Daten (siehe Capability `dienstkonto-dynamische-soll-formel`). Der in der `duty_accounts`-Tabelle gespeicherte Wert bleibt davon unberührt — der `/api/dashboard`-Endpoint berechnet den Wert live.

#### Scenario: Family views own duty account (updated)
- **WHEN** ein `elternteil` das Dashboard aufruft
- **THEN** sieht er `soll` basierend auf der dynamischen Formel (Kader-Spielanzahl, Templates, Spielerzahl, Elternanzahl)
- **AND** der Erklärtext lautet „Ziel: {soll} Dienste (Saison {name})" ohne Formel-Details
