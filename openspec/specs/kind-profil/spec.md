### Requirement: Eltern sehen Kind-Einträge in der Navigation

Für Nutzer mit Rolle `elternteil` MÜSSEN unterhalb des Navigations-Eintrags „Mein Profil" dynamische Einträge für jedes verknüpfte Kind angezeigt werden. Die Einträge zeigen den Vornamen des Kindes als Label (z.B. „Jannes Profil"). Kinder ohne `family_links`-Verknüpfung erscheinen nicht.

#### Scenario: Elternteil mit einem Kind

- **WHEN** ein `elternteil`-Nutzer eingeloggt ist und ein Kind via `family_links` verknüpft hat
- **THEN** erscheint unter „Mein Profil" ein Eintrag „[Vorname]s Profil" in der Sidebar-Navigation

#### Scenario: Elternteil mit mehreren Kindern

- **WHEN** ein `elternteil`-Nutzer mehrere Kinder verknüpft hat
- **THEN** erscheinen alle Kinder als separate Einträge in der Navigation, alphabetisch nach Vorname

#### Scenario: Nutzer ohne Elternteil-Rolle

- **WHEN** ein Nutzer mit Rolle `spieler`, `trainer`, `vorstand` oder `admin` eingeloggt ist
- **THEN** erscheinen keine Kind-Einträge in der Navigation

### Requirement: Strikte family_links-Autorisierung auf allen Kind-Endpunkten

Jeder Endpunkt unter `/api/profile/kind/:memberId` (GET, PUT /member, PUT /bank) MUSS als erste Operation prüfen, ob ein Eintrag in `family_links` mit `parent_user_id = <eingeloggter User>` AND `member_id = <memberId>` existiert. Fehlt dieser Eintrag, MUSS der Endpunkt sofort mit HTTP 403 antworten — unabhängig von der Rolle des anfragenden Nutzers. Die Autorisierungsprüfung DARF NICHT durch Kenntnis einer fremden memberId umgangen werden können.

#### Scenario: Elternteil greift auf eigenes Kind zu

- **WHEN** `GET /api/profile/kind/42` mit JWT von User 7 aufgerufen wird und `family_links` enthält (parent_user_id=7, member_id=42)
- **THEN** gibt der Endpunkt HTTP 200 mit den Kindprofil-Daten zurück

#### Scenario: Elternteil versucht fremdes Kind aufzurufen

- **WHEN** `GET /api/profile/kind/99` mit JWT von User 7 aufgerufen wird und `family_links` enthält KEINEN Eintrag (parent_user_id=7, member_id=99)
- **THEN** antwortet der Endpunkt mit HTTP 403, ohne Daten zu Member 99 preiszugeben

#### Scenario: Elternteil versucht fremdes Kind zu bearbeiten (PUT)

- **WHEN** `PUT /api/profile/kind/99/member` oder `PUT /api/profile/kind/99/bank` mit JWT von User 7 aufgerufen wird und keine Verknüpfung (parent_user_id=7, member_id=99) existiert
- **THEN** antwortet der Endpunkt mit HTTP 403, ohne Daten zu schreiben

#### Scenario: Nicht-Elternteil-Nutzer ruft Kind-Endpunkt auf

- **WHEN** `GET /api/profile/kind/42` von einem Nutzer mit Rolle `spieler` aufgerufen wird
- **THEN** antwortet der Endpunkt mit HTTP 403 (kein family_links-Eintrag vorhanden)

### Requirement: Route für Kindprofil

Das System MUSS eine Route `/profil/kind/:memberId` bereitstellen, die das Profil eines verknüpften Kindes anzeigt. Das Frontend MUSS bei HTTP 403-Antwort des Backends zur Startseite weiterleiten und keinen Profilinhalt anzeigen.

#### Scenario: Gültiger Aufruf mit verknüpftem Kind

- **WHEN** ein `elternteil`-Nutzer `/profil/kind/42` aufruft und `family_links` eine Verknüpfung zwischen dem Nutzer und Member 42 enthält
- **THEN** wird das Kindprofil mit den Tabs Kontakt, Mitgliedsdaten und Bankdaten angezeigt

#### Scenario: Direktaufruf einer fremden memberId im Browser

- **WHEN** ein Nutzer `/profil/kind/99` direkt im Browser aufruft und das Backend HTTP 403 zurückgibt
- **THEN** zeigt das Frontend keinen Profilinhalt und leitet zur Startseite (`/`) weiter

### Requirement: Backend-Endpunkt zum Lesen des Kindprofils

Das System MUSS einen Endpunkt `GET /api/profile/kind/:memberId` bereitstellen, der das vollständige Profil eines Kindes zurückgibt. Die Autorisierung erfolgt ausschließlich über die `family_links`-Tabelle (siehe Requirement „Strikte family_links-Autorisierung").

#### Scenario: Berechtigter Elternteil liest Kindprofil

- **WHEN** `GET /api/profile/kind/42` mit gültigem JWT eines berechtigten Elternteils aufgerufen wird
- **THEN** gibt der Endpunkt ein JSON-Objekt zurück mit Mitgliedsdaten des Kindes (name, DOB, jersey_number, position, status, address) sowie Bankdaten (IBAN, account_holder), HTTP 200

#### Scenario: Unberechtigter Nutzer liest Kindprofil

- **WHEN** `GET /api/profile/kind/42` von einem Nutzer aufgerufen wird, der nicht in `family_links` mit Member 42 verknüpft ist
- **THEN** antwortet der Endpunkt mit HTTP 403

### Requirement: Eltern können Mitgliedsdaten des Kindes bearbeiten

Das System MUSS einen Endpunkt `PUT /api/profile/kind/:memberId/member` bereitstellen. Eltern DÜRFEN folgende Felder bearbeiten: Vorname, Nachname, Geburtsdatum, Trikot-Nummer, Position, Straße, PLZ, Ort.

#### Scenario: Mitgliedsdaten erfolgreich aktualisiert

- **WHEN** `PUT /api/profile/kind/42/member` mit gültigen Feldern und Berechtigung aufgerufen wird
- **THEN** werden die Felder in der `members`-Tabelle gespeichert, Antwort HTTP 204

#### Scenario: Status kann nicht durch Elternteil geändert werden

- **WHEN** `PUT /api/profile/kind/42/member` mit einem `status`-Feld aufgerufen wird
- **THEN** wird das `status`-Feld ignoriert (nur admin/vorstand darf Status ändern)

### Requirement: Eltern können Bankdaten des Kindes bearbeiten

Das System MUSS einen Endpunkt `PUT /api/profile/kind/:memberId/bank` bereitstellen, mit dem Eltern `iban` und `account_holder` in der `members`-Tabelle setzen können.

#### Scenario: Bankdaten erfolgreich gesetzt

- **WHEN** `PUT /api/profile/kind/42/bank` mit `{ "iban": "DE...", "account_holder": "Max Muster" }` aufgerufen wird
- **THEN** werden IBAN und Kontoinhaber in `members` gespeichert, Antwort HTTP 204

#### Scenario: Leere IBAN löscht Bankdaten

- **WHEN** `PUT /api/profile/kind/42/bank` mit `{ "iban": "", "account_holder": "" }` aufgerufen wird
- **THEN** werden IBAN und Kontoinhaber auf NULL gesetzt

### Requirement: Eltern können Adresse des Kindes bearbeiten

Der Endpunkt `PUT /api/profile/kind/:memberId/member` MUSS die Felder `street`, `zip`, `city` in der `members`-Tabelle des Kindes speichern.

#### Scenario: Adresse gesetzt

- **WHEN** `PUT /api/profile/kind/42/member` mit `{ "street": "Musterstr. 1", "zip": "70173", "city": "Stuttgart" }` aufgerufen wird
- **THEN** werden die Felder in `members.street/zip/city` gespeichert

### Requirement: Kindprofil zeigt Tab Kontakt, Mitgliedsdaten, Bankdaten

Die Seite `/profil/kind/:memberId` MUSS dieselben Tab-Komponenten wie „Mein Profil" nutzen — jedoch ohne den „Konto"-Tab und den „Sonstiges"-Tab. Telefonnummern-Bearbeitung MUSS nur angezeigt werden, wenn das Kind einen verlinkten User-Account hat (`member.user_id != null`).

#### Scenario: Kind ohne User-Account

- **WHEN** das Kindprofil geladen wird und das Kind kein `user_id` hat
- **THEN** werden Tabs Kontakt (Adresse only), Mitgliedsdaten, Bankdaten angezeigt; kein Telefonnummern-Abschnitt

#### Scenario: Kind mit User-Account

- **WHEN** das Kindprofil geladen wird und das Kind ein `user_id` hat
- **THEN** werden alle drei Tabs angezeigt; der Kontakt-Tab enthält auch Telefonnummern-Verwaltung (Lesen/Schreiben via Kind-User-ID)
