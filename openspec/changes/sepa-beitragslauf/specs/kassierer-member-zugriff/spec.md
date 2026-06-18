## ADDED Requirements

### Requirement: Kassierer-Lesezugriff auf Mitglieder
Nutzer mit Vereinsfunktion `kassierer` SHALL die Mitgliederliste und Mitglieder-Detailseiten lesen können (`GET /api/members`, `GET /api/members/{id}`, `GET /api/members/{id}/parents`, `GET /api/members/export`). Diese Routen MUST von der bisherigen Vorstand-only-Gruppe in eine `vorstand`+`kassierer`-Gruppe wandern. Mitglieder anlegen, vollständig bearbeiten, löschen, Status ändern, importieren sowie Rollen-/Family-Verwaltung BLEIBEN `vorstand`-only (Admin-Override unverändert).

#### Scenario: Kassierer liest Mitgliederliste
- **WHEN** ein Nutzer mit `club_functions: ["kassierer"]` `GET /api/members` aufruft
- **THEN** antwortet der Server mit HTTP 200 und der Mitgliederliste

#### Scenario: Spieler bleibt ausgesperrt
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `GET /api/members` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Kassierer darf nicht vollständig bearbeiten
- **WHEN** ein Kassierer `POST /api/members` oder `DELETE /api/members/{id}` aufruft
- **THEN** antwortet der Server mit HTTP 403

### Requirement: Bankdaten-Bearbeitung durch Kassierer
Nutzer mit Vereinsfunktion `kassierer` (und `vorstand`/`admin`) SHALL via `PUT /api/members/{id}/bankdaten` ausschließlich die bankrelevanten Felder eines Mitglieds aktualisieren können: `iban`, `sepa_mandat`, `sepa_mandat_date`, `account_holder`, `street`, `zip`, `city`. Der Endpoint MUST alle übrigen Member-Felder (Name, Status, `beitragsfrei`, Rollen) unverändert lassen. Eine ungültige IBAN (Mod-97) MUST mit HTTP 400 abgelehnt werden. Das SEPA-Mandat-Hochladen/Löschen (`POST /api/upload/sepa-mandat/{id}`, `DELETE /api/members/{id}/sepa-mandat`) SHALL ebenfalls für `kassierer` erlaubt sein.

#### Scenario: Kassierer ändert nur Bankfelder
- **WHEN** ein Kassierer `PUT /api/members/{id}/bankdaten` mit neuer IBAN und Adresse aufruft
- **THEN** sind IBAN und Adresse aktualisiert und Name, Status sowie `beitragsfrei` des Mitglieds unverändert

#### Scenario: Ungültige IBAN abgelehnt
- **WHEN** ein `PUT /api/members/{id}/bankdaten` mit einer IBAN mit falscher Mod-97-Prüfsumme erfolgt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Bankdaten ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `PUT /api/members/{id}/bankdaten` aufruft
- **THEN** antwortet der Server mit HTTP 403
