## MODIFIED Requirements

### Requirement: Sitzungsgebundenes Schlüssel-Caching im Browser

Der entschlüsselte Gruppen-Privatschlüssel MUST im Browser ausschließlich als nicht-exportierbares `CryptoKey`-Objekt gehalten werden; Schlüsselmaterial im Klartext oder als PKCS8 darf in keinem Web-Storage liegen. Der Schlüssel MUST Navigation und Reload innerhalb der Sitzung überleben (IndexedDB) und MUST nach 30 Minuten Inaktivität sowie bei Abmeldung gelöscht werden. Ein aus der Zeit vor dieser Regel stammender `sessionStorage`-Eintrag MUST beim Laden entfernt werden.

#### Scenario: Schlüssel ist nicht exportierbar
- **WHEN** Skript im Browser versucht, den gecachten Privatschlüssel zu exportieren
- **THEN** schlägt der Export fehl und kein Schlüsselmaterial ist in `sessionStorage` oder `localStorage` vorhanden

#### Scenario: Reload behält den entsperrten Tresor
- **WHEN** ein Tresor-Inhaber die Seite innerhalb des Timeouts neu lädt
- **THEN** ist der Tresor weiterhin entsperrt, ohne erneute Passphrase-Eingabe

#### Scenario: Timeout sperrt
- **WHEN** 30 Minuten seit dem Entsperren vergangen sind
- **THEN** ist der Schlüssel gelöscht und der Tresor gesperrt
