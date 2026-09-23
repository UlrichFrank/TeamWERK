## ADDED Requirements

### Requirement: Kein personenbezogenes Datum im öffentlichen Repo
Der öffentliche Repository MUST frei von echten personenbezogenen Daten sein — sowohl im aktuellen Tree als auch in der gesamten Git-Historie. Dies umfasst Namen realer Personen, Adressen, Geburtsdaten, Kontaktdaten, IBAN/BIC/Kontoinhaber, SEPA-Mandatsreferenzen sowie hochgeladene Mitglieds-Dokumente.

#### Scenario: Audit findet keinen Echt-Datensatz
- **WHEN** das PII-Audit über `git log --all` und den aktuellen Tree läuft
- **THEN** finden sich keine realen Mitgliederdaten, DB-Dumps oder hochgeladenen Dokumente
- **AND** das Audit-Ergebnis ist als Checkliste im Repo dokumentiert

#### Scenario: Verdächtige Datei wird vor Veröffentlichung verifiziert
- **WHEN** eine Datei wie `teamwerk_dump.sql` oder `testdata/*.csv` potenziell reale Daten enthält
- **THEN** wird ihr Inhalt manuell geprüft und entweder entfernt oder durch nachweislich synthetische Daten ersetzt, bevor der Public-Repo entsteht

### Requirement: Bereinigte History ohne PII
Der öffentliche Repository MUST so erzeugt werden, dass keine PII aus früheren Commits rekonstruierbar ist. Der History-Rewrite (`git-filter-repo`) MUST alle bekannten PII-Blobs aus allen Commits entfernen, und ein Pattern-Scan über die neue History MUST leer ausgehen, bevor öffentlich gepusht wird.

#### Scenario: PII-Blobs aus allen Commits entfernt
- **WHEN** `git-filter-repo` mit der dokumentierten Pfadliste gelaufen ist
- **THEN** referenziert kein Commit der neuen History `teamwerk_dump.sql`, `storage/files/*` oder `internal/mailer/attachments/*.pdf`

#### Scenario: Pattern-Scan über neue History ist leer
- **WHEN** `git log --all -p` der neuen History gegen die PII-Patterns (IBAN, reale Mail-Domains, `INSERT INTO members`) gescannt wird
- **THEN** liefert der Scan keinen Treffer, andernfalls wird nicht öffentlich gepusht

### Requirement: Mechanischer Guard gegen künftige PII-Commits
Ein Pre-Commit-Schritt MUST Commits ablehnen, die bekannte PII-Datenklassen einführen (DB-Dateien, `*_dump.sql`, CSV mit personenbezogenen Spaltenköpfen, IBAN-Muster im Diff).

#### Scenario: DB-Dump wird geblockt
- **WHEN** eine Datei `*.db` oder `*_dump.sql` gestaget und committet werden soll
- **THEN** bricht der Pre-Commit-Hook mit einer erklärenden Fehlermeldung ab

#### Scenario: IBAN im Diff wird geblockt
- **WHEN** ein Commit-Diff eine Zeichenkette enthält, die dem IBAN-Muster entspricht und nicht als Test-IBAN markiert ist
- **THEN** bricht der Pre-Commit-Hook ab und nennt die betroffene Zeile
