## MODIFIED Requirements

### Requirement: Sitzungsgebundenes Schlüssel-Caching im Browser

Das System SHALL den entsperrten Gruppen-Privatschlüssel im Browser ausschließlich als nicht-exportierbares `CryptoKey`-Objekt halten; Schlüsselmaterial im Klartext oder als PKCS8 darf in keinem Web-Storage liegen. Der Schlüssel MUST Navigation und Reload innerhalb der Sitzung überleben (IndexedDB), MUST nach 30 Minuten Inaktivität sowie bei Abmeldung gelöscht werden und SHALL nie in `localStorage` oder an den Server gelangen. Ein aus der Zeit vor dieser Regel stammender `sessionStorage`-Eintrag MUST beim Laden entfernt werden.

#### Scenario: Inaktivitäts-Timeout
- **WHEN** 30 Minuten ohne Interaktion vergehen, nachdem der Tresor entsperrt wurde
- **THEN** wird der Schlüssel aus IndexedDB entfernt und eine erneute Eingabe verlangt

#### Scenario: Falsches Secret wird clientseitig erkannt
- **WHEN** ein falsches Gruppen-Secret eingegeben wird
- **THEN** schlägt die clientseitige Key-Check-Entschlüsselung fehl und es wird kein
  Server-Request ausgelöst

#### Scenario: Schlüssel ist nicht exportierbar
- **WHEN** Skript im Browser versucht, den gecachten Privatschlüssel zu exportieren
- **THEN** schlägt der Export fehl und kein Schlüsselmaterial ist in `sessionStorage` oder `localStorage` vorhanden

#### Scenario: Reload behält den entsperrten Tresor
- **WHEN** ein Tresor-Inhaber die Seite innerhalb des Timeouts neu lädt
- **THEN** ist der Tresor weiterhin entsperrt, ohne erneute Passphrase-Eingabe

#### Scenario: Altformat wird entsorgt
- **WHEN** beim Laden ein `sessionStorage`-Eintrag mit PKCS8-Schlüsselmaterial vorhanden ist
- **THEN** wird er entfernt und der Tresor gilt als gesperrt
