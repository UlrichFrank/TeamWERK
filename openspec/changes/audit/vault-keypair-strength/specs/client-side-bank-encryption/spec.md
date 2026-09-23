## ADDED Requirements

### Requirement: Mindest-Schlüsselstärke des Gruppen-Keypairs

Das bei der Tresor-Einrichtung clientseitig erzeugte Vereins-Gruppen-Keypair, das die Mitglieds-DEKs wrappt, SHALL mindestens RSA-3072 (≈128-Bit-Sicherheit) verwenden ODER ein gleichwertiges Verfahren (z.B. X25519/ECDH-ES + HKDF zum DEK-Wrapping). Bestehende RSA-2048-Installationen SHALL über den vorhandenen Keypair-Rotations-Pfad auf die Mindeststärke angehoben werden können, ohne Datenverlust und ohne dass der Server Klartext sieht. Beim Lesen SHALL der Client weiterhin auch ältere (schwächere) Envelopes entschlüsseln können, bis eine Rotation erfolgt ist.

#### Scenario: Neues Tresor-Setup erzeugt ein Keypair der Mindeststärke
- **WHEN** ein Tresor-Inhaber die Einrichtung („Tresor") neu durchführt
- **THEN** wird ein Gruppen-Keypair mit RSA-3072 (oder X25519-äquivalent) erzeugt und gespeichert

#### Scenario: Rotation hebt ein bestehendes 2048-Keypair an
- **WHEN** eine Keypair-Rotation auf einer Installation mit RSA-2048 ausgeführt wird
- **THEN** entsteht ein neues Keypair der Mindeststärke, alle Mitglieds-DEKs werden neu gewrappt und der Bestand bleibt vollständig lesbar

#### Scenario: Lesen älterer Envelopes vor der Rotation
- **WHEN** vor einer Rotation ein mit RSA-2048 gewrappter Datensatz gelesen wird
- **THEN** kann der Client ihn weiterhin entschlüsseln (Abwärtskompatibilität)
