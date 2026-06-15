## ADDED Requirements

### Requirement: Kader aus Vorsaison kopieren
Das System SHALL beim Saisonwechsel eine Kader-Struktur aus der Vorsaison in die neue Saison kopieren können. Mit `member_source=same-age-previous` werden die Mitglieder der gleichen Altersklasse übernommen.

#### Scenario: Kader kopieren mit same-age-previous
- **WHEN** POST /api/kader/copy-from-season mit from_season_id, to_season_id und assignments [{age_class, gender, member_source: "same-age-previous"}]
- **THEN** HTTP 200, neuer Kader in Ziel-Saison angelegt, Mitglieder der gleichen Altersklasse aus Quell-Saison übernommen

#### Scenario: Kader kopieren ohne Mitglieder
- **WHEN** POST mit member_source="" (leer)
- **THEN** HTTP 200, Kader-Struktur angelegt, keine kader_members-Einträge
