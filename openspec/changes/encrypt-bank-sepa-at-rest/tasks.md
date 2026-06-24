## 1. Krypto-Kern (`internal/crypto`)

- [x] 1.1 Package `internal/crypto` anlegen: `Encrypt(plaintext string) (string, error)` → `"v1:" + base64(nonce ‖ ciphertext)` (AES-256-GCM, zufälliges 12-Byte-Nonce)
- [x] 1.2 `Decrypt(value string) (string, error)`: ohne `"v1:"`-Prefix unverändert zurückgeben; mit Prefix entschlüsseln; bei gebrochener GCM-Auth Fehler statt Klartext
- [x] 1.3 Datei-Varianten `EncryptBytes`/`DecryptBytes` mit Magic-Header für PDF-Inhalte (Erkennung „bereits verschlüsselt")
- [x] 1.4 Schlüssel-Laden aus `FIELD_ENCRYPTION_KEY` (base64, 32 Byte) + `MustLoadKey`/Validierung
- [x] 1.5 Unit-Tests: Roundtrip, falscher Schlüssel schlägt fehl, Klartext-Passthrough, manipulierter Ciphertext → Fehler, idempotenter Re-Encrypt-Skip, Datei-Roundtrip
- [x] 1.6 Architektur-Test (`internal/arch`): `internal/crypto` als Foundation klassifizieren (importiert keine Domain)

## 2. Subcommands & Startup (`cmd/teamwerk`)

- [ ] 2.1 Subcommand `gen-encryption-key`: gibt base64-kodierten 32-Byte-Schlüssel aus
- [ ] 2.2 Startup-Check in `main.go`: fehlender/ungültiger `FIELD_ENCRYPTION_KEY` → Start abbrechen mit klarer Fehlermeldung
- [ ] 2.3 Subcommand `encrypt-pii` (idempotent): verschlüsselt Bestand der vier Speicher (Werte ohne `"v1:"`-Prefix bzw. PDFs ohne Magic-Header), Dateien via atomic rename
- [ ] 2.4 Spiegelbildliches `decrypt-pii` (Rollback/Rotation), ebenfalls idempotent
- [ ] 2.5 Tests für `encrypt-pii`/`decrypt-pii`: Bestand wird verschlüsselt, zweiter Lauf ist idempotent, Roundtrip encrypt→decrypt stellt Klartext wieder her

## 3. Zentrale Autorisierung (`internal/policy`)

- [ ] 3.1 `CanDecryptBankData(db, p *Principal, memberUserID int) bool` = `admin ∨ IsVorstandLike ∨ IsKassiererLike ∨ Eigentümer (p.UserID==memberUserID) ∨ isParentOf`
- [ ] 3.2 Eltern-Prüfung gegen `family_links` (DB-gestützt, analog `FolderAccess`)
- [ ] 3.3 Tests für jede Kombination: admin, vorstand, kassierer, Eigentümer, Elternteil → erlaubt; Trainer, fremdes Mitglied, fremdes Elternteil → verweigert

## 4. Schreibpfade (Encrypt einziehen)

- [ ] 4.1 `members.UpdateBankdaten` (`PUT /api/members/{id}/bank-details`): `iban` + `account_holder` vor dem Schreiben verschlüsseln
- [ ] 4.2 `members.UpdateChildBank` (`PUT /api/profile/kind/{id}/bank`): `iban` + `account_holder` verschlüsseln
- [ ] 4.3 change-request-Create (`POST /api/members/{id}/change-request`, `field_name='bankdaten'`): `new_value` verschlüsselt ablegen
- [ ] 4.4 `config.UpdateClub` (`PUT /api/club`): `iban`/`bic`/`glaeubiger_id`/`kontoinhaber` verschlüsseln (Validierung weiterhin auf dem Klartext **vor** dem Verschlüsseln)
- [ ] 4.5 `upload`-SEPA-Upload (`POST /api/upload/sepa-mandat/{id}`): PDF-Inhalt verschlüsselt speichern
- [ ] 4.6 `upload`-SEPA-Bulk-Import (`POST /api/members/sepa-mandates/import`): importierte PDFs verschlüsselt speichern

## 5. Lesepfade (Decrypt hinter `CanDecryptBankData`)

- [ ] 5.1 `members.Get` (`GET /api/members/{id}`): IBAN/Kontoinhaber entschlüsseln, nur wenn berechtigt
- [ ] 5.2 `beitragslauf.Export` (`POST /api/fee-run/export`): Mitglieds- und Vereins-Felder zur Laufzeit entschlüsseln (XML-Builder unverändert)
- [ ] 5.3 `config.GetClub` (`GET /api/club`): Vereins-SEPA-Felder entschlüsseln
- [ ] 5.4 SEPA-Download (`GET /api/members/{id}/sepa-mandat/download`): PDF-Inhalt beim Ausliefern entschlüsseln
- [ ] 5.5 change-drafts-Anzeige (`GET /api/members/{id}/change-drafts`): `bankdaten`-`new_value` entschlüsseln, nur wenn berechtigt
- [ ] 5.6 NEU `members.GetProfile` (`GET /api/profile/me`): eigene IBAN/Kontoinhaber entschlüsselt zurückgeben
- [ ] 5.7 NEU `members.GetChildProfile` (`GET /api/profile/kind/{id}`): IBAN/Kontoinhaber des Kindes entschlüsselt zurückgeben (403 bei fremdem Kind)

## 6. Frontend (Eigentümer-/Eltern-Lesen)

- [ ] 6.1 Profilseite (`web/src/pages/ProfilePage.tsx`): eigene Bankdaten anzeigen (brand-Tokens, lucide-Icons)
- [ ] 6.2 Kind-Profilseite: Bankdaten des Kindes für Eltern anzeigen
- [ ] 6.3 `pnpm -C web build` + lint grün

## 7. Tests pro Route (Happy + Fehlerfall)

- [ ] 7.1 `bank-details`: 200 (vorstand/kassierer) · 403 (trainer) · gespeicherter Wert trägt `"v1:"`-Prefix
- [ ] 7.2 `members.Get`: berechtigt → Klartext-IBAN · trainer/fremdes Mitglied → keine Bankdaten
- [ ] 7.3 `profile/me`: Eigentümer → eigene IBAN entschlüsselt
- [ ] 7.4 `profile/kind/{id}`: Elternteil → 200 mit IBAN · fremdes Kind → 403
- [ ] 7.5 `fee-run/export`: erzeugt korrektes XML aus entschlüsselten Feldern
- [ ] 7.6 `club` GET/PUT: Roundtrip mit Verschlüsselung, Validierung unverändert
- [ ] 7.7 SEPA-Upload/Download: Upload speichert verschlüsselt, Download liefert Original-PDF
- [ ] 7.8 Invarianten-Test: nicht-berechtigte Rolle erhält nie entschlüsselte Bankdaten; DB-Spalten enthalten nach Schreibzugriff nie den Klartext

## 8. SSE / Live-Updates

- [ ] 8.1 Prüfen, dass berührte Mutations-Routen weiterhin `h.hub.Broadcast(...)` aufrufen (bank-details, club, change-request, sepa-upload) und Frontend-Seiten `useLiveUpdates` nutzen

## 9. Deployment & Doku

- [ ] 9.1 `docs/agent/10-deployment.md`: `FIELD_ENCRYPTION_KEY` in `/etc/teamwerk/env` (chmod 600), Rollout-Sequenz (gen-key → deploy → encrypt-pii), Backup-Regel (Key ≠ DB-Backup; Schlüsselverlust = Datenverlust)
- [ ] 9.2 `docs/agent/03-go.md` o.ä.: Hinweis „Bank-/SEPA-Felder immer via `internal/crypto` schreiben/lesen, Lesen nur hinter `policy.CanDecryptBankData`"
- [ ] 9.3 `.env.example`/Setup-Skripte um `FIELD_ENCRYPTION_KEY` ergänzen

## 10. Spec-Bereinigung & Abschluss

- [ ] 10.1 `openspec validate encrypt-bank-sepa-at-rest --strict` grün
- [ ] 10.2 Volles Gate (`/verify-change`): build/test/lint, Route→Tests, Mutation→Broadcast, brand-Tokens, lucide-Icons
- [ ] 10.3 Beim Archivieren: Specs `member-encryption` und `vorstand-vault` werden via Delta entfernt; neue Capability `bank-data-at-rest-encryption` übernommen
