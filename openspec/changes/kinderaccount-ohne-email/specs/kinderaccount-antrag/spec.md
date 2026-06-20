## ADDED Requirements

### Requirement: Beitrittsantrag-Variante „Kinderaccount"
Das öffentliche Beitrittsantrag-Formular SHALL eine Variante „Kinderaccount anlegen" anbieten. Ist sie aktiv, MUSS der Antrag den Vor- und Nachnamen des Kindes sowie eine verwaltende Eltern-E-Mail erfassen (statt einer eigenen E-Mail des Antragstellers) und mit `is_child=1` sowie `parent_email` gespeichert werden. Ein Geschlechtsfeld wird NICHT erfasst.

#### Scenario: Kinderantrag wird angelegt
- **WHEN** ein Kinderantrag mit Vorname, Nachname und gültiger `parent_email` abgeschickt wird
- **THEN** legt das System einen `membership_requests`-Eintrag mit `is_child=1`, gesetztem `parent_email` und `status='pending'` an (HTTP 201/200)

#### Scenario: Kinderantrag ohne Eltern-E-Mail wird abgelehnt
- **WHEN** ein Kinderantrag ohne `parent_email` (oder mit ungültiger E-Mail) abgeschickt wird
- **THEN** lehnt das System mit HTTP 400 ab

#### Scenario: Standard-Antrag bleibt unverändert
- **WHEN** ein Antrag ohne Kinderaccount-Variante (eigene E-Mail) abgeschickt wird
- **THEN** wird er wie bisher mit `is_child=0` gespeichert

### Requirement: Approve eines Kinderantrags erzeugt Konto, Mitglied und Eltern-Mail
Beim Akzeptieren eines Antrags mit `is_child=1` SHALL das System in einer Transaktion einen `users`-Datensatz (`email=NULL`, generierter `login_name`, leeres Passwort, `can_login=0`), einen verknüpften `members`-Datensatz mit dem echten Kindnamen anlegen und einen Passwort-Setz-Token erzeugen. Anschließend MUSS eine E-Mail an die `parent_email` mit dem zugewiesenen Spielernamen und dem Passwort-Setz-Link versandt werden. Es wird KEIN `family_link` automatisch angelegt.

#### Scenario: Kinderantrag wird akzeptiert
- **WHEN** der Vorstand einen `is_child=1`-Antrag akzeptiert
- **THEN** existieren danach ein `users`-Datensatz (`login_name` gesetzt, `can_login=0`) und ein über `user_id` verknüpfter `members`-Datensatz, und es wurde eine Mail an die `parent_email` versandt; der Antrag-Status ist `approved`

#### Scenario: Eindeutiger Spielername bei Namensgleichheit
- **WHEN** beim Akzeptieren der erzeugte `login_name` bereits vergeben ist
- **THEN** wird ein eindeutiger Name mit numerischem Suffix vergeben und im Konto gespeichert

#### Scenario: Kein automatischer Eltern-Link
- **WHEN** ein Kinderantrag akzeptiert wird
- **THEN** existiert kein `family_link` zwischen der `parent_email` und dem neuen Kind-Mitglied (reine Korrespondenz)

#### Scenario: Fehlgeschlagener Mailversand bricht nicht die Kontoanlage ab
- **WHEN** der Mailversand an die `parent_email` fehlschlägt, nachdem Konto und Mitglied erfolgreich committed wurden
- **THEN** bleiben Konto und Mitglied bestehen und der Fehler wird protokolliert (kein Rollback der bereits committeten Daten)
