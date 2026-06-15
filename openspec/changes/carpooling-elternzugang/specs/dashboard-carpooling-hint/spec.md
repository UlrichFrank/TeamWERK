## MODIFIED Requirements

### Requirement: Personalisierten Fahrtgemeinschafts-Status anzeigen

Das System SHALL in der Dashboard-Antwort (`GET /api/dashboard`) unter `carpoolingConfirmed` auch bestätigte Paarungen der Kinder des eingeloggten Nutzers anzeigen — zusätzlich zu den eigenen Paarungen.

`carpoolingConfirmed` enthält die nächsten max. 3 Auswärtsspiele, bei denen entweder der Nutzer selbst ODER eines seiner Kinder eine `confirmed`-Paarung hat. `partnerName` zeigt die Gegenseite aus Sicht des Nutzers/Kindes.

#### Scenario: User mit bestätigter Paarung

- **WHEN** der User eine Paarung mit `status='confirmed'` für ein kommendes Auswärtsspiel hat
- **THEN** enthält `carpoolingConfirmed` dieses Spiel mit der entsprechenden Paarung

#### Scenario: Kind mit bestätigter Paarung

- **WHEN** ein Kind des eingeloggten Elternteils eine `confirmed`-Paarung für ein kommendes Auswärtsspiel hat
- **THEN** erscheint diese Paarung ebenfalls in `carpoolingConfirmed` des Elternteils

#### Scenario: Kein Auswärtsspiel mit Paarung

- **WHEN** weder der Nutzer noch ein Kind eine bestätigte Paarung für kommende Auswärtsspiele haben
- **THEN** ist `carpoolingConfirmed` ein leeres Array

#### Scenario: Nutzer ohne Kinder — unverändert

- **WHEN** ein Nutzer ohne `family_links`-Einträge das Dashboard lädt
- **THEN** enthält `carpoolingConfirmed` ausschließlich seine eigenen Paarungen (Verhalten wie bisher)
