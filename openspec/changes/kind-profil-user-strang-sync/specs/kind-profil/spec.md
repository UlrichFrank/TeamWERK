## MODIFIED Requirements

### Requirement: Kindprofil zeigt Tab Kontakt, Mitgliedsdaten, Bankdaten

Die Seite `/profil/kind/:memberId` MUSS dieselben Tab-Komponenten wie „Mein Profil" nutzen — jedoch ohne den „Konto"-Tab und den „Sonstiges"-Tab. Wenn das Kind einen verlinkten User-Account hat (`member.user_id != null`), MÜSSEN der Kontakt-Tab Telefonnummern-Verwaltung und Sichtbarkeitseinstellungen aus dem `user_contact`-Objekt des API-Response anzeigen und bearbeiten. Das Speichern MUSS sowohl `PUT /profile/kind/{id}/account` (User-Strang sofort) als auch `POST /members/{id}/change-request` (Member-Strang via Draft) aufrufen.

#### Scenario: Kind ohne User-Account

- **WHEN** das Kindprofil geladen wird und das Kind kein `user_id` hat
- **THEN** werden Tabs Kontakt (Adresse only), Mitgliedsdaten, Bankdaten angezeigt; kein Telefonnummern-Abschnitt

#### Scenario: Kind mit User-Account — Speichern aktualisiert User-Strang sofort

- **WHEN** ein Elternteil im Kontakt-Tab des Kindprofils Änderungen speichert und das Kind `user_id` hat
- **THEN** werden `PUT /profile/kind/{id}/account` (sofort in `users`) und `POST /members/{id}/change-request` (Draft für Vorstand) aufgerufen

#### Scenario: Kind mit User-Account — Kontakt-Tab zeigt user_contact-Daten

- **WHEN** das Kindprofil geladen wird und das Kind `user_id` hat
- **THEN** zeigt der Kontakt-Tab Name, Adresse und Telefonnummern aus `user_contact` (nicht aus `members`)

#### Scenario: Direktaufruf einer fremden memberId im Browser

- **WHEN** ein Nutzer `/profil/kind/99` direkt im Browser aufruft und das Backend HTTP 403 zurückgibt
- **THEN** zeigt das Frontend keinen Profilinhalt und leitet zur Startseite (`/`) weiter
