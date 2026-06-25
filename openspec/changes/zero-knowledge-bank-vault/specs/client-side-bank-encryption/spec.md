## ADDED Requirements

### Requirement: Clientseitige Verschlüsselung — Server speichert nur Ciphertext

Das System SHALL Bank-/SEPA-PII (Mitglieds-IBAN/Kontoinhaber, `member_change_drafts`
mit `field_name='bankdaten'`, Vereins-SEPA-Stammdaten, SEPA-Mandat-PDFs) ausschließlich
als **clientseitig** erzeugten Ciphertext entgegennehmen und speichern. Der Server SHALL
zu keinem Zeitpunkt einen Entschlüsselungsschlüssel für diese Felder besitzen und SHALL
keinen Endpoint anbieten, der Klartext dieser Felder zurückgibt. Schreib-Endpoints SHALL
einen mitgelieferten Klartextwert für diese Felder ablehnen.

#### Scenario: Bankdaten werden als Ciphertext + Wraps gespeichert
- **WHEN** ein berechtigter Nutzer über `PUT /api/members/{id}/bank-details` Bankdaten setzt
- **THEN** enthält der Request einen AES-GCM-Ciphertext-Blob und mindestens einen
  gewrappten Data-Key (Group-Wrap); der Server speichert beides unverändert und sieht
  keinen Klartext

#### Scenario: Server bietet keinen Klartext-Lesepfad
- **WHEN** irgendein Client Bank-/SEPA-Felder eines Mitglieds liest
- **THEN** liefert der Server nur Ciphertext + gewrappte Data-Keys, nie entschlüsselte Werte

#### Scenario: Klartext-Schreibversuch wird abgewiesen
- **WHEN** ein Request für ein Bank-/SEPA-Feld einen Wert ohne gültiges Envelope-Format
  (Ciphertext + Wrap) sendet
- **THEN** antwortet der Server mit HTTP 400 und speichert nichts

### Requirement: Envelope-Encryption mit Data-Key pro Mitglied

Das System SHALL pro Mitglied einen zufälligen 256-bit Data-Key (DEK) verwenden, mit dem
die Bankdaten AES-GCM-verschlüsselt werden. Der DEK SHALL mit dem Schlüssel der
Finance-Gruppe (`vorstand`/`kassierer`) gewrappt und als `dek_enc_vorstand` gespeichert
werden. Ein Eigentümer-Wrap (`dek_enc_member`) SHALL **nicht** existieren — alle
Mitglieder (mit und ohne Nutzerkonto) erhalten ausschließlich einen Group-Wrap. Die
DEK-Schicht dient der Entkopplung der Passphrase-Rotation vom Datenbestand (Rotation =
DEK neu wrappen, kein Blob-Neuencrypt).

#### Scenario: Jedes Mitglied erhält genau einen Group-Wrap
- **WHEN** Bankdaten eines beliebigen Mitglieds gespeichert werden
- **THEN** wird genau ein `dek_enc_vorstand` gespeichert; kein Eigentümer-Wrap existiert

#### Scenario: Eingabe ohne Rücklese-Möglichkeit
- **WHEN** ein Mitglied oder Elternteil seine/die Bankdaten des Kindes einträgt
- **THEN** werden sie an den Gruppenschlüssel gewrappt und sind für den Eintragenden
  anschließend **nicht** im Klartext zurücklesbar

### Requirement: Lesen ausschließlich durch die Finance-Gruppe

Das System SHALL die Entschlüsselung so gestalten, dass **ausschließlich** Inhaber des
Finance-Gruppen-Schlüssels (`vorstand`, `kassierer`) die Bankdaten entschlüsseln können.
Kein anderer Nutzer — **einschließlich `admin` und einschließlich des Eigentümers/eines
Elternteils** — und nicht der Server SHALL die Daten entschlüsseln können.

#### Scenario: Trainer kann nicht entschlüsseln
- **WHEN** ein Nutzer ohne `vorstand`/`kassierer`-Funktion die Bank-Blobs anfordert
- **THEN** erhält er zwar ggf. Ciphertext, aber keinen für ihn entschlüsselbaren Wrap

#### Scenario: Admin ohne Gruppenschlüssel kann nicht entschlüsseln
- **WHEN** ein `admin` ohne Kenntnis des Finance-Gruppen-Secrets Bankdaten anfordert
- **THEN** kann er die Daten nicht entschlüsseln (kein Server-seitiger Decrypt, kein Wrap)

#### Scenario: Eigentümer kann eigene Daten nicht zurücklesen
- **WHEN** das verknüpfte Mitglied selbst seine Bankdaten anfordert
- **THEN** erhält es keinen für sich entschlüsselbaren Wrap (nur die Finance-Gruppe liest)

### Requirement: Einrichtung des Finance-Gruppen-Schlüssels (einmalig)

Bevor Bankdaten geschrieben werden können, SHALL die Finance-Gruppe einmalig ein
asymmetrisches Gruppen-Keypair einrichten. Bei der Einrichtung SHALL clientseitig ein
Keypair erzeugt werden; gespeichert werden der **öffentliche** Schlüssel (im Klartext,
nicht geheim), der mit `PBKDF2(passphrase, salt)` **verschlüsselte private** Schlüssel, der
Salt und ein Key-Check-Wert. Die Passphrase und der private Schlüssel im Klartext SHALL den
Browser nie verlassen. Eine erneute Einrichtung bei bereits vorhandener Konfiguration SHALL
mit HTTP 409 abgewiesen werden (stattdessen Rotation).

#### Scenario: Einrichtung speichert öffentlichen + verschlüsselten privaten Schlüssel
- **WHEN** ein berechtigter Nutzer die Einrichtung abschließt
- **THEN** werden der öffentliche Schlüssel, der passphrase-verschlüsselte private Schlüssel,
  Salt und Key-Check gespeichert; Passphrase und Klartext-Privatschlüssel nicht

#### Scenario: Doppelte Einrichtung abgewiesen
- **WHEN** bereits eine Gruppen-Konfiguration existiert
- **THEN** antwortet der Einrichtungs-Endpoint mit HTTP 409 und verweist auf die Rotation

### Requirement: Passphrase- und Keypair-Rotation bei Rollenwechsel

Da alle `vorstand`/`kassierer` dieselbe Tresor-Passphrase teilen, SHALL die Aufnahme einer
neuen Person ohne Server-/Krypto-Operation erfolgen (Weitergabe der Passphrase
out-of-band). Beim Ausscheiden einer Person SHALL ein **aktueller** Halter die Passphrase
**rotieren** können: clientseitig den privaten Schlüssel mit der alten Passphrase
entschlüsseln, mit der neuen Passphrase (+ neuem Salt + Key-Check) neu verschlüsseln und an
den Server schreiben — **ohne** die Mitglieds-DEKs anzufassen (öffentlicher Schlüssel
unverändert). Der Server erfährt **weder** alte **noch** neue Passphrase. Zusätzlich SHALL
bei Kompromittierung des privaten Schlüssels eine **Keypair-Rotation** möglich sein (neues
Keypair, alle DEKs neu wrappen). Nach jeder Rotation SHALL der Bestand vollständig lesbar
bleiben und mit der alten Passphrase **nicht** mehr entschlüsselbar sein.

#### Scenario: Aufnahme ohne Krypto-Operation
- **WHEN** eine neue `kassierer`-Person aufgenommen wird
- **THEN** genügt die Weitergabe der bestehenden Passphrase; es erfolgt keine Server-Änderung

#### Scenario: Passphrase-Rotation lässt DEKs unangetastet
- **WHEN** ein aktueller Halter mit entsperrtem Tresor eine neue Passphrase bestätigt
- **THEN** wird nur der verschlüsselte private Schlüssel (+ Salt + Key-Check) ersetzt; die
  Mitglieds-DEKs und der öffentliche Schlüssel bleiben unverändert; der Server sieht keine
  Passphrase

#### Scenario: Alte Passphrase nach Rotation wertlos
- **WHEN** nach einer Rotation die alte Passphrase eingegeben wird
- **THEN** schlägt die Key-Check-Entschlüsselung fehl und der private Schlüssel lässt sich
  nicht mehr entschlüsseln

### Requirement: Sitzungsgebundenes Schlüssel-Caching im Browser

Das System SHALL den entsperrten Gruppen-/Eigentümer-Schlüssel nur im flüchtigen
Browser-Speicher (`sessionStorage`) halten, nach 30 Minuten Inaktivität verwerfen und beim
Schließen des Tabs nicht persistieren. Der Schlüssel SHALL nie in `localStorage` oder an
den Server gelangen.

#### Scenario: Inaktivitäts-Timeout
- **WHEN** 30 Minuten ohne Interaktion vergehen, nachdem der Tresor entsperrt wurde
- **THEN** wird der Schlüssel aus `sessionStorage` entfernt und eine erneute Eingabe verlangt

#### Scenario: Falsches Secret wird clientseitig erkannt
- **WHEN** ein falsches Gruppen-Secret eingegeben wird
- **THEN** schlägt die clientseitige Key-Check-Entschlüsselung fehl und es wird kein
  Server-Request ausgelöst
