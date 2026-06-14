## MODIFIED Requirements

### Requirement: Maps-Deep-Link anzeigen
Überall wo ein Venue mit Adresse angezeigt wird, SHALL ein anklickbarer Maps-Link erscheinen.

#### Scenario: Link öffnet Maps-App (Präferenz-abhängig)
- **WHEN** Nutzer klickt auf den Maps-Link eines Venues
- **THEN** öffnet der Browser die URL zum konfigurierten Kartendienst des Nutzers (google, apple oder auto-erkannt) mit der kodierten Adresse in einem neuen Tab

#### Scenario: Kein Venue vorhanden
- **WHEN** Event hat venue_id = null
- **THEN** Kein Maps-Link wird angezeigt; Ort-Bereich bleibt leer oder zeigt „Kein Ort angegeben"
