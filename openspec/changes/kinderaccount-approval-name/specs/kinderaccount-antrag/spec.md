## ADDED Requirements

### Requirement: Approve eines Kinderantrags erzeugt Konto und Eltern-Mail

Beim Akzeptieren eines Antrags mit `is_child=1` SHALL das System in einer Transaktion **ausschließlich einen `users`-Datensatz** anlegen (`email=NULL`, generierter, eindeutiger `login_name`, `first_name`/`last_name` des Kindes aus dem Antrag, leeres Passwort, `can_login=0`) und einen Passwort-Setz-Token erzeugen. Anschließend MUSS eine E-Mail an die `parent_email` mit dem zugewiesenen Spielernamen und dem Passwort-Setz-Link versandt werden.

Es wird **KEIN `members`-Datensatz und KEIN `family_link`** angelegt (reine Korrespondenz; das Mitglied wird ggf. später separat über die Mitgliederverwaltung erfasst). Nach dem Approve DARF `users.first_name`/`last_name` NICHT leer sein. Der `login_name` gilt aufgrund von Transliteration, Zeichen-Normalisierung und Kollisions-Suffix als verlustbehaftet und SHALL NICHT als Namensquelle dienen.

#### Scenario: Kinderantrag wird akzeptiert
- **WHEN** der Vorstand einen `is_child=1`-Antrag für „Lena Schmidt" akzeptiert
- **THEN** existiert danach genau ein `users`-Datensatz mit `first_name='Lena'`, `last_name='Schmidt'`, gesetztem `login_name` und `can_login=0`; es wurde eine Mail an die `parent_email` versandt; der Antrag-Status ist `approved`

#### Scenario: Name erscheint in der Nutzerliste
- **WHEN** nach dem Approve `GET /api/users` aufgerufen wird
- **THEN** enthält der Eintrag des Kinder-Kontos den nicht-leeren Namen des Kindes

#### Scenario: Eindeutiger Spielername bei Namensgleichheit
- **WHEN** beim Akzeptieren der erzeugte `login_name` bereits vergeben ist
- **THEN** wird ein eindeutiger Name mit numerischem Suffix vergeben und im Konto gespeichert

#### Scenario: Kein Mitglied und kein automatischer Eltern-Link
- **WHEN** ein Kinderantrag akzeptiert wird
- **THEN** existiert weder ein `members`-Datensatz noch ein `family_link` für das neue Kinder-Konto (reine Korrespondenz)

#### Scenario: Fehlgeschlagener Mailversand bricht nicht die Kontoanlage ab
- **WHEN** der Mailversand an die `parent_email` fehlschlägt, nachdem das Konto erfolgreich committed wurde
- **THEN** bleibt das Konto bestehen und der Fehler wird protokolliert (kein Rollback der bereits committeten Daten)

### Requirement: Bestehende namenlose Kinder-Konten nachfüllen

Bereits vor Einführung des Namens-Fixes ohne Namen angelegte Kinder-Konten (`can_login=0`, `login_name` gesetzt, `email IS NULL`, leerer `first_name`) SHALL das System, soweit über die Antragsdaten in `membership_requests` eindeutig zuordenbar, per Migration nachfüllen. Nicht eindeutig zuordenbare Konten SHALL es unverändert lassen (kein Ratewerk über den verlustbehafteten `login_name`).

#### Scenario: Backfill füllt Bestandskonto eindeutig
- **WHEN** ein vor dem Fix angelegtes namenloses Kinder-Konto existiert, dessen `recovery_email` und `login_name` eindeutig einem `approved` `is_child=1`-Antrag zuzuordnen sind
- **THEN** setzt die Migration `first_name`/`last_name` des Kontos auf die Namen aus diesem Antrag

#### Scenario: Backfill lässt Mehrdeutiges unangetastet
- **WHEN** ein namenloses Kinder-Konto keinem eindeutigen `membership_requests`-Antrag zugeordnet werden kann
- **THEN** bleiben `first_name`/`last_name` des Kontos unverändert (leer), ohne einen falschen Namen zu raten

## REMOVED Requirements

### Requirement: Approve eines Kinderantrags erzeugt Konto, Mitglied und Eltern-Mail
**Reason**: Fehlformulierung — der Approve legt bewusst KEIN Mitglied an (nur ein `users`-Konto, „reine Korrespondenz"; `internal/auth/handler.go:493-497`). Die Requirement und ihre Szenarien beschrieben eine Mitglieds-Erzeugung, die der Code nie durchführte (Spec/Code-Drift).
**Migration**: Ersetzt durch „Approve eines Kinderantrags erzeugt Konto und Eltern-Mail" (korrigierte Formulierung: nur Konto inkl. `first_name`/`last_name`, kein `members`-Datensatz).
