## 1. Datenbank-Migration

- [ ] 1.1 Neue Migration anlegen (`internal/db/migrations/011_member_sensitive.up.sql`): Tabelle `member_sensitive` mit Spalten `member_id` (PK FK), `ciphertext`, `dek_enc_vorstand`, `dek_enc_member` (nullable), `member_salt` (nullable)
- [ ] 1.2 `clubs`-Tabelle um `vorstand_kdf_salt TEXT` und `vorstand_key_check TEXT` erweitern (ebenfalls in Migration 011)
- [ ] 1.3 Down-Migration anlegen (`011_member_sensitive.down.sql`): Tabelle droppen, clubs-Spalten droppen

## 2. Backend — Neue API-Endpunkte

- [ ] 2.1 `GET /api/admin/encryption-config` implementieren: gibt `{ vorstand_kdf_salt, vorstand_key_check, configured: bool }` zurück (Vorstand + Admin)
- [ ] 2.2 `PUT /api/admin/encryption-config` implementieren: setzt `vorstand_kdf_salt` + `vorstand_key_check` in `clubs`; lehnt ab mit 409 wenn bereits gesetzt
- [ ] 2.3 `GET /api/members/{id}/sensitive` implementieren: gibt Ciphertext-Blob zurück; Vorstand bekommt beide DEK-Felder, Mitglied nur eigenes; 403 für Fremdzugriff; 204 wenn kein Eintrag vorhanden
- [ ] 2.4 `PUT /api/members/{id}/sensitive` implementieren: Upsert in `member_sensitive`; nur Vorstand darf schreiben; alle Felder werden als Text gespeichert ohne Dekodierung
- [ ] 2.5 `PUT /api/admin/rotate-encryption` implementieren: nimmt `{ new_salt, new_key_check, entries: [{member_id, dek_enc_vorstand}] }`, updated alle Zeilen und `clubs`-Spalten in einer Transaktion
- [ ] 2.6 `GET /api/members/export-encrypted` implementieren: gibt JSON-Array aller Mitglieder zurück, jedes mit Klartextfeldern (Name, Status etc.) + `ciphertext`, `dek_enc_vorstand`; nur Vorstand

## 3. Backend — Passwort-Änderung anpassen

- [ ] 3.1 `PUT /api/auth/change-password` Request-Struct um optionale Felder `dek_enc_member` + `member_salt` erweitern
- [ ] 3.2 Wenn `dek_enc_member` mitgeschickt wird: `member_sensitive`-Eintrag des Users atomar mit Passwort-Hash-Update aktualisieren

## 4. Backend — Sensitive Felder aus bestehenden Routen entfernen

- [ ] 4.1 `Member`-Struct in `handler.go`: Felder `DateOfBirth`, `Street`, `Zip`, `City`, `IBAN`, `AccountHolder` entfernen
- [ ] 4.2 Alle `SELECT`-Abfragen in `members/handler.go` anpassen: sensitive Spalten aus `members`-Table-Queries entfernen
- [ ] 4.3 `PUT /api/members/{id}` Request-Struct bereinigen: sensitive Felder entfernen
- [ ] 4.4 Alten `GET /api/members/export`-Handler entfernen (durch `/export-encrypted` ersetzt)
- [ ] 4.5 `GET /api/profile/me` prüfen: sicherstellen, dass keine sensitiven Felder zurückgegeben werden

## 5. Frontend — Crypto-Utility

- [ ] 5.1 `web/src/lib/crypto.ts` anlegen mit Funktionen: `deriveKey(passphrase, salt)`, `wrapKey(dek, key)`, `unwrapKey(wrapped, key)`, `encrypt(payload, dek)`, `decrypt(ciphertext, dek)`, `generateDEK()`
- [ ] 5.2 Hilfsfunktionen für Base64 ↔ ArrayBuffer Konvertierung in `crypto.ts`
- [ ] 5.3 `deriveKeyFromPassword(password, salt)` für member_key ergänzen (gleiche PBKDF2-Parameter wie Vorstand)
- [ ] 5.4 Funktion `verifyVaultPassphrase(passphrase, salt, keyCheck)` implementieren (intern: ableiten + decrypt → prüfen ob "ok")

## 6. Frontend — VaultContext

- [ ] 6.1 `web/src/contexts/VaultContext.tsx` anlegen: hält `isUnlocked`, `unlockVault(passphrase)`, `lockVault()`, `vaultKey` (aus sessionStorage geladen)
- [ ] 6.2 Inaktivitäts-Timer in VaultContext: nach 30 Minuten ohne Interaktion `sessionStorage['vk']` löschen und `isUnlocked` zurücksetzen
- [ ] 6.3 VaultProvider in `App.tsx` einbinden (innerhalb AuthProvider)

## 7. Frontend — Vault-UI-Komponenten

- [ ] 7.1 `VaultPassphraseDialog`-Komponente erstellen: Modal mit Passphrase-Input, Fehleranzeige, Bestätigen-Button; ruft `VaultContext.unlockVault()` auf
- [ ] 7.2 `VaultGate`-Wrapper-Komponente erstellen: zeigt `VaultPassphraseDialog` wenn `!isUnlocked`, sonst rendert Children
- [ ] 7.3 Seite `/admin/tresor-einrichten` erstellen: einmaliges Setup-Formular (Passphrase + Bestätigung), postet Salt + Key-Check an `PUT /api/admin/encryption-config`
- [ ] 7.4 Seite `/admin/tresor-verwaltung` erstellen: zeigt Vault-Status, Link zur Rotation, Initialmigrations-Trigger
- [ ] 7.5 Rotations-Workflow in `/admin/tresor-verwaltung`: neue Passphrase eingeben → alle DEKs re-wrappen → POST an `PUT /api/admin/rotate-encryption`
- [ ] 7.6 Nav-Eintrag für Tresor-Verwaltung in `AppShell.tsx` (nur für `vorstand`-Rolle sichtbar)

## 8. Frontend — Mitglied-Detailseite: Sensitive Felder

- [ ] 8.1 Sensitive Felder (Geburtsdatum, Adresse, IBAN) aus dem regulären Member-Fetch entfernen und durch separaten `GET /api/members/{id}/sensitive`-Aufruf ersetzen
- [ ] 8.2 Sensitive Felder mit `VaultGate` schützen: nur anzeigen wenn Vault entsperrt
- [ ] 8.3 Entschlüsselung asynchron in der Komponente: `unwrapKey` + `decrypt` → Felder in lokalem State halten
- [ ] 8.4 Edit-Formular für sensitive Felder: bei Speichern `encrypt` + `wrapKey` ausführen, `PUT /api/members/{id}/sensitive` aufrufen
- [ ] 8.5 Mitglieder-Selbstzugriff: eigenes Profil (`GET /api/profile/me`) lädt `GET /api/members/{id}/sensitive` und entschlüsselt mit member_key (aus Login-Passwort abgeleitet + in sessionStorage gecacht)

## 9. Frontend — Passwort-Änderung mit DEK-Re-Wrap

- [ ] 9.1 Passwort-Änderungs-Formular (`/profil` oder Modal) um altes Passwort-Feld erweitern (benötigt für Key-Ableitung)
- [ ] 9.2 Beim Absenden: alten `member_key` ableiten → `dek_enc_member` entschlüsseln → DEK mit neuem `member_key` wrappen → `dek_enc_member` + `member_salt` im Request mitsenden
- [ ] 9.3 Wenn kein `member_sensitive`-Eintrag vorhanden: normalen Passwort-Change ohne DEK-Re-Wrap durchführen

## 10. Frontend — Verschlüsselter Export

- [ ] 10.1 Alten CSV-Export-Button durch neuen ersetzen, der `GET /api/members/export-encrypted` aufruft
- [ ] 10.2 Für jeden Datensatz: `unwrapKey(dek_enc_vorstand, vaultKey)` + `decrypt(ciphertext, dek)` → sensitive Felder zusammenführen
- [ ] 10.3 CSV-Generierung im Browser via `Blob` + dynamischem `<a download>`-Trigger
- [ ] 10.4 Export mit `VaultGate` schützen

## 11. Frontend — Initialmigration

- [ ] 11.1 Migrations-UI in `/admin/tresor-verwaltung`: Fortschrittsanzeige, Start-Button
- [ ] 11.2 Migrations-Logik: `GET /api/members` → für jeden Eintrag mit vorhandenen Klartextfeldern: `encrypt` → `PUT /api/members/{id}/sensitive`; bereits migrierte überspringen
- [ ] 11.3 Nach abgeschlossener Migration: Hinweis anzeigen dass serverseitige Klartext-Spalten manuell per Migration gedroppt werden können
