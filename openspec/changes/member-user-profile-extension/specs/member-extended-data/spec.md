## ADDED Requirements

### Requirement: Mitglied hat Adressfelder mit Fallback auf Nutzer-Adresse
Das System SHALL für jedes Mitglied die Felder `street`, `zip`, `city` speichern (nullable). Wenn diese Felder leer sind und das Mitglied einen verknüpften Nutzer hat, SHALL die Adresse des Nutzers als Fallback verwendet werden. Das Feld `address_source` im Response zeigt `"member"` oder `"user"` an.

#### Scenario: Admin speichert eigene Mitglieds-Adresse
- **WHEN** Admin `PUT /api/members/{id}` mit `street`, `zip`, `city` aufruft
- **THEN** werden die Felder in der DB gespeichert; `GET /api/members/{id}` gibt `address_source: "member"` zurück

#### Scenario: Fallback auf Nutzer-Adresse wenn Mitglied keine hat
- **WHEN** `members.street` ist NULL, Mitglied hat verknüpften Nutzer mit gesetzter Adresse, Admin ruft `GET /api/members/{id}` auf
- **THEN** Response enthält die Adresse des Nutzers und `address_source: "user"`

#### Scenario: Kein Fallback wenn kein Nutzer verknüpft
- **WHEN** `members.street` ist NULL und `members.user_id` ist NULL
- **THEN** Response enthält `street: null, zip: null, city: null, address_source: null`

#### Scenario: Verknüpfter Nutzer liest effektive Adresse seines Mitglieds
- **WHEN** Nutzer `GET /api/members/{id}` aufruft und `members.user_id == claims.UserID`
- **THEN** werden effektive Adresse + `address_source` im Response mitgeliefert (read-only)

#### Scenario: Anderer Nutzer sieht keine Adresse
- **WHEN** ein Nutzer `GET /api/members/{id}` aufruft und `members.user_id != claims.UserID` und Rolle ist nicht `admin` oder `trainer`
- **THEN** fehlen `street`, `zip`, `city`, `address_source` im Response

### Requirement: Mitglied hat Eintrittsdatum
Das System SHALL ein `join_date`-Feld (DATE, nullable) auf dem Mitglied speichern.

#### Scenario: Admin setzt Eintrittsdatum
- **WHEN** Admin `PUT /api/members/{id}` mit `join_date: "2020-09-01"` aufruft
- **THEN** wird das Datum gespeichert und im `GET`-Response zurückgegeben

### Requirement: Mitglied hat IBAN (nur Admin)
Das System SHALL ein `iban`-Feld (TEXT, nullable) speichern. Die IBAN DARF NUR für Nutzer mit Rolle `admin` im API-Response erscheinen.

#### Scenario: Admin liest IBAN
- **WHEN** ein Admin `GET /api/members/{id}` aufruft
- **THEN** ist `iban` im Response enthalten (auch wenn null)

#### Scenario: Nicht-Admin bekommt keine IBAN
- **WHEN** ein Trainer, Spieler oder Elternteil `GET /api/members/{id}` aufruft
- **THEN** fehlt das Feld `iban` im Response vollständig

### Requirement: Mitglied hat DSGVO-Einwilligungen
Das System SHALL zwei DSGVO-Flags speichern: `dsgvo_verarbeitung` (Einwilligung zur Datenverarbeitung) und `dsgvo_weitergabe` (Einwilligung zur Weitergabe von Name + Foto). Jeweils als Boolean (0/1) + optionales Datum.

#### Scenario: Admin setzt DSGVO-Einwilligung
- **WHEN** Admin `PUT /api/members/{id}` mit `dsgvo_verarbeitung: true, dsgvo_verarbeitung_date: "2024-01-15"` aufruft
- **THEN** werden Flag und Datum gespeichert

#### Scenario: Verknüpfter Nutzer sieht eigene DSGVO-Flags
- **WHEN** ein Nutzer sein eigenes Mitglied abruft
- **THEN** sind `dsgvo_verarbeitung` und `dsgvo_weitergabe` (jeweils + date) im Response

### Requirement: Mitglied hat SEPA-Mandat mit Dokument-Upload
Das System SHALL `sepa_mandat` (Boolean), `sepa_mandat_date` (DATE) und `sepa_mandat_path` (Pfad zum hochgeladenen Mandat-Dokument) speichern. Upload via `POST /api/upload/sepa-mandat/{id}`. Erlaubte Typen: PDF, JPEG, PNG, WEBP. Maximale Dateigröße: 10 MB. Das Dokument (z.B. scan des unterschriebenen Mandats) ist nur für Admin sichtbar.

#### Scenario: Admin setzt SEPA-Mandat-Flag
- **WHEN** Admin `PUT /api/members/{id}` mit `sepa_mandat: true, sepa_mandat_date: "2024-02-01"` aufruft
- **THEN** werden Flag und Datum gespeichert und im Admin-Response zurückgegeben

#### Scenario: Admin lädt SEPA-Dokument hoch
- **WHEN** Admin `POST /api/upload/sepa-mandat/{id}` mit PDF-Datei (≤ 10 MB) aufruft
- **THEN** Datei gespeichert unter `sepa-mandats/`, `members.sepa_mandat_path` gesetzt, Response enthält `sepa_mandat_url`

#### Scenario: Nicht-Admin sieht kein SEPA-Dokument
- **WHEN** Nicht-Admin `GET /api/members/{id}`
- **THEN** fehlt `sepa_mandat_url` im Response; `sepa_mandat` (bool) kann für verknüpften Nutzer sichtbar sein
