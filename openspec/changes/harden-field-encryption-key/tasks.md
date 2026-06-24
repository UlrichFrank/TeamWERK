## 1. Schlüsselquelle mit Fallback (`internal/crypto`)

- [ ] 1.1 `LoadKeyFromSources()`: liest bevorzugt Datei `$CREDENTIALS_DIRECTORY/field_key`, sonst `FIELD_ENCRYPTION_KEY` aus der Umgebung; gibt Schlüssel + Quellenname zurück
- [ ] 1.2 `InitFromEnv()` → `InitFromSources()` umstellen; Validierung (base64, 32 Byte) unverändert wiederverwenden
- [ ] 1.3 Startup-Check/Log in `cmd/teamwerk serve`: aktive Quelle eindeutig protokollieren (NIE den Schlüssel selbst); Boot-Abbruch ohne nutzbare Quelle bleibt
- [ ] 1.4 Unit-Tests: Credential-Datei hat Vorrang vor Env; Fallback auf Env; keine Quelle → Fehler; ungültiger Wert → Fehler

## 2. Versioniertes Format + Schlüsselregister (`internal/crypto`)

- [ ] 2.1 Schlüsselregister: aktiver Schlüssel (Schreiben) + optional Alt-Schlüssel (nur Entschlüsseln); Konstante für aktuelles Schreibformat (`"v2:"`)
- [ ] 2.2 `Decrypt`/`DecryptBytes`: Version am Präfix/Magic-Header erkennen und passenden Schlüssel wählen; unbekanntes/kein Präfix → Klartext-Passthrough; gebrochene Auth → Fehler
- [ ] 2.3 `Encrypt`/`EncryptBytes`: schreiben im aktuellen Format (`"v2:"`)
- [ ] 2.4 Unit-Tests: v1-Lesen mit Alt-Schlüssel, v2-Roundtrip, gemischter Bestand, falscher/fehlender Alt-Schlüssel → Fehler

## 3. Subcommand `rotate-key` (`cmd/teamwerk` + `internal/crypto`)

- [ ] 3.1 Alt-Schlüssel-Übergabe definieren (zweite Credential `field_key_old` bzw. `FIELD_ENCRYPTION_KEY_OLD`) und laden
- [ ] 3.2 `RotatePII(db, uploadDir)`: iteriert die vier Speicher, entschlüsselt `"v1:"`→ schreibt `"v2:"` (Dateien atomic rename), idempotent (überspringt `"v2:"`)
- [ ] 3.3 Abbruch ohne Teil-Schreibvorgang, wenn Alt-Schlüssel fehlt oder ein Wert nicht entschlüsselbar ist
- [ ] 3.4 Subcommand `rotate-key` verdrahten (Config/DB/UploadDir laden, Report loggen)
- [ ] 3.5 Tests: Bestand `"v1:"`→`"v2:"`, zweiter Lauf idempotent, fehlender Alt-Schlüssel → Abbruch ohne Änderung

## 4. systemd-Credential (Deployment)

- [ ] 4.1 `deploy/teamwerk.service`: `LoadCredentialEncrypted=field_key:/etc/teamwerk/field_key.cred`; `EnvironmentFile` für den Key entlasten (Env-Fallback dokumentiert belassen)
- [ ] 4.2 `deploy/setup-vps.sh`: Credential via `systemd-creds encrypt --with-key=auto` erzeugen (TPM2 sonst host-key); systemd-Versions-/TPM-Verfügbarkeit prüfen und protokollieren
- [ ] 4.3 Verhalten ohne TPM/zu altes systemd: sauberer Fallback auf Env-Quelle, klare Meldung

## 5. Deploy-Skript-Integration (`deploy/deploy-encryption.sh`)

- [ ] 5.1 Schritt „Credential sicherstellen" ergänzen (idempotent: nur erzeugen, wenn `field_key.cred` fehlt); `--dry-run` zeigt es an
- [ ] 5.2 `rotate-key`-Pfad als optionaler Modus (mit erzwungenem `make backup` davor)

## 6. Dokumentation

- [ ] 6.1 `docs/agent/10-deployment.md`: Credential-Provisionierung, Quellen-Reihenfolge, Host-Bindung ⇒ rohen Schlüssel weiterhin separat sichern, Rotations-Runbook
- [ ] 6.2 `docs/agent/03-go.md`: Schlüsselquelle (Credential vor Env), `"v2:"`-Format, `rotate-key`-Hinweis
- [ ] 6.3 `.env.example`: Kommentar, dass `FIELD_ENCRYPTION_KEY` lokaler Fallback ist (Prod nutzt systemd-Credential)

## 7. Test-Anforderungen

- [ ] 7.1 `crypto` Quellen-Auswahl: Credential-Datei > Env (Vorrang) · nur Env (Fallback) · keine Quelle → Init-Fehler · ungültig → Init-Fehler
- [ ] 7.2 `crypto` Versionierung: `Decrypt("v1:")` mit Alt-Schlüssel = Klartext · `Encrypt` erzeugt `"v2:"` · gemischter Bestand lesbar · manipuliert → Fehler
- [ ] 7.3 `rotate-key`: Bestand wird `"v2:"`, Roundtrip korrekt; zweiter Lauf idempotent; fehlender Alt-Schlüssel → Abbruch ohne Schreibzugriff (Invariante: keine Teil-Rotation)
- [ ] 7.4 Architektur-Test bleibt grün (`internal/crypto` Foundation, keine Domain-Importe)

## 8. Validierung & Abschluss

- [ ] 8.1 `openspec validate harden-field-encryption-key --strict` grün
- [ ] 8.2 Volles Gate (`/verify-change`): build/test/lint (Go-Tooling mit ungesetztem `GOROOT`), keine raw-Tailwind/Unicode-Icons (kein Frontend betroffen)
- [ ] 8.3 Rollout verifizieren: Service startet aus Credential-Quelle (Log zeigt Quelle); Env-Fallback funktioniert; `rotate-key` auf DB-Kopie getestet
- [ ] 8.4 Beim Archivieren: Delta auf `bank-data-at-rest-encryption` anwenden (MODIFIED Schlüssel-Quelle, ADDED Rotation)
