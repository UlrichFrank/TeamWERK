## ADDED Requirements

### Requirement: Feld `beitragsfrei_grund`

Das System SHALL ein optionales Textfeld `beitragsfrei_grund` (`TEXT NULL`) auf der Tabelle `members` führen. Es speichert die Begründung, weshalb das Mitglied beitragsfrei gestellt ist (z. B. `kein aktiver Sportler mehr`, `Zweitspielrecht`). Das Feld wird via `GET /api/members/{id}` ausgeliefert, via `PUT /api/members/{id}` (Vorstand/Admin) und via `PUT /api/members/{id}/bank-details` (Kassierer, siehe Capability `kassierer-member-zugriff`) geschrieben.

#### Scenario: Feld in der Detail-Response

- **WHEN** ein Vorstand `GET /api/members/{id}` aufruft, dessen Mitglied `beitragsfrei_grund = "kein aktiver Sportler mehr"` hat
- **THEN** enthält die Response das Feld `beitragsfrei_grund` mit diesem Wert

#### Scenario: Feld setzbar durch Vorstand

- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: true, beitragsfrei_grund: "Zweitspielrecht"` aufruft
- **THEN** persistiert das System beide Felder und `GET /api/members/{id}` liefert sie zurück

### Requirement: Kopplung `beitragsfrei` und `beitragsfrei_grund`

Das System SHALL die Invariante durchsetzen: wenn `members.beitragsfrei = 0`, dann `members.beitragsfrei_grund IS NULL`. Diese Kopplung MUST auf Applikationsebene in jedem schreibenden Pfad (`PUT /api/members/{id}`, `PUT /api/members/{id}/bank-details`) erzwungen werden. Ein DB-CHECK-Constraint wird BEWUSST nicht eingeführt, damit fehlerhafte Eingaben mit HTTP 204 bereinigt werden statt 500 zu liefern.

#### Scenario: Deaktivieren räumt den Grund auf

- **GIVEN** ein Mitglied mit `beitragsfrei=true, beitragsfrei_grund="Zweitspielrecht"`
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: false` aufruft (Grund-Feld nicht oder beliebig mitgesendet)
- **THEN** speichert das System `beitragsfrei=false` und `beitragsfrei_grund=NULL`
- **AND** `GET /api/members/{id}` liefert `beitragsfrei_grund` als leer/null

#### Scenario: Aktivieren ohne Grund ist erlaubt

- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: true` und leerem `beitragsfrei_grund` aufruft
- **THEN** speichert das System `beitragsfrei=true` und `beitragsfrei_grund=NULL`
- **AND** liefert HTTP 204

### Requirement: UI-Darstellung im Bankdaten-Block

Das Frontend SHALL im `MemberKontaktTab` unterhalb der Checkbox „Beitragsfrei" ein Textinput „Grund für Beitragsfreiheit" anzeigen, das NUR sichtbar ist, wenn die Checkbox aktiv ist. Beim Deselektieren der Checkbox MUST das Form-Feld lokal geleert werden, damit der Server beim Speichern `NULL` schreibt.

#### Scenario: Grund-Feld sichtbar, wenn beitragsfrei aktiv

- **WHEN** die Bankdaten-Sektion mit `form.beitragsfrei === true` gerendert wird
- **THEN** ist das Textinput „Grund" sichtbar und editierbar

#### Scenario: Grund-Feld verschwindet beim Toggle

- **GIVEN** `form.beitragsfrei === true` und `form.beitragsfrei_grund === "kein aktiver Sportler mehr"`
- **WHEN** der Nutzer die Checkbox abwählt
- **THEN** verschwindet das Grund-Input und der Form-State setzt `beitragsfrei: false, beitragsfrei_grund: ''`
