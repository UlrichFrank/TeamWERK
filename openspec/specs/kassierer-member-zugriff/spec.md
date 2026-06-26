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

Nutzer mit Vereinsfunktion `kassierer` (und `vorstand`/`admin`) SHALL via `PUT /api/members/{id}/bank-details` ausschließlich die bankrelevanten Felder eines Mitglieds aktualisieren können: `iban`, `sepa_mandat`, `sepa_mandat_date`, `account_holder`, `street`, `zip`, `city`, `beitragsfrei`, `beitragsfrei_grund`. Der Endpoint MUST alle übrigen Member-Felder (Name, Status, Rollen, Geburtsdatum, …) unverändert lassen. Eine ungültige IBAN (Mod-97) MUST mit HTTP 400 abgelehnt werden. Das SEPA-Mandat-Hochladen/Löschen (`POST /api/upload/sepa-mandat/{id}`, `DELETE /api/members/{id}/sepa-mandat`) SHALL ebenfalls für `kassierer` erlaubt sein.

Das Feld-Paar `beitragsfrei` und `beitragsfrei_grund` wird bewusst gemeinsam freigegeben: der Grund ist Teil des Bankdaten-Tabs (UI-Sicht des Kassierers), und die Kopplung `beitragsfrei=false ⇒ beitragsfrei_grund=NULL` MUST serverseitig erzwungen werden — unabhängig vom Wert, den der Client für `beitragsfrei_grund` sendet.

#### Scenario: Kassierer ändert Bankfelder inkl. Beitragsfrei-Grund
- **WHEN** ein Kassierer `PUT /api/members/{id}/bank-details` mit `iban`, `beitragsfrei: true` und `beitragsfrei_grund: "kein aktiver Sportler mehr"` aufruft
- **THEN** sind IBAN, `beitragsfrei` und `beitragsfrei_grund` aktualisiert
- **AND** Name, Status und Rollen des Mitglieds sind unverändert

#### Scenario: Deaktivieren räumt Grund auf (Bankdaten-Endpoint)
- **GIVEN** ein Mitglied mit `beitragsfrei=true, beitragsfrei_grund="Zweitspielrecht"`
- **WHEN** ein Kassierer `PUT /api/members/{id}/bank-details` mit `beitragsfrei: false` (Grund beliebig oder weggelassen) aufruft
- **THEN** speichert das System `beitragsfrei=false` und `beitragsfrei_grund=NULL`
- **AND** antwortet HTTP 204

#### Scenario: Ungültige IBAN abgelehnt
- **WHEN** ein `PUT /api/members/{id}/bank-details` mit einer IBAN mit falscher Mod-97-Prüfsumme erfolgt
- **THEN** antwortet der Server mit HTTP 400

#### Scenario: Bankdaten ohne Berechtigung
- **WHEN** ein Nutzer mit ausschließlich `club_functions: ["spieler"]` `PUT /api/members/{id}/bank-details` aufruft
- **THEN** antwortet der Server mit HTTP 403

#### Scenario: Status bleibt unverändert
- **GIVEN** ein Mitglied mit `members.status='aktiv'`
- **WHEN** ein Kassierer `PUT /api/members/{id}/bank-details` mit beliebigem Body aufruft
- **THEN** bleibt `members.status='aktiv'` (Status ist nicht in der Whitelist)
