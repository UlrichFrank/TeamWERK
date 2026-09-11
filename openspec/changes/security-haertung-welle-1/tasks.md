## 1. Upload-Härtung (kritisch)

- [x] 1.1 `internal/upload`: Sniffing immer, Endung aus Typ-Map, Client-`Content-Type` nur noch informativ; PDF-Pfad analog.
- [x] 1.2 `streamFile`: `Content-Type` aus gespeichertem Typ/Sniffing, `Content-Disposition: attachment` für alles außer `image/*`, `X-Content-Type-Options` bleibt.
- [x] 1.3 Tests: HTML mit `image/png`-Header → 400; PNG mit falscher Endung `.html` im Dateinamen → gespeichert als `.png`; Auslieferung eines Nicht-Bilds trägt `attachment`.

## 2. Tresor-Schlüssel nicht exportierbar (kritisch, Folge)

- [x] 2.1 `VaultContext`: Privatschlüssel als nicht-exportierbarer `CryptoKey` in IndexedDB, Timeout-Zeitstempel weiter in `sessionStorage`, Alt-Eintrag `vk` beim Mount entfernen.
- [x] 2.2 `lib/crypto.ts`: `decryptPrivateKey` liefert `extractable=false`; Setup/Rotation-Pfade, die exportieren müssen, bekommen einen expliziten `exportable`-Parameter.
- [x] 2.3 Vitest: Schlüssel im Kontext ist nicht exportierbar (`exportKey` wirft); Reload stellt ihn aus IndexedDB wieder her; Timeout löscht ihn.

## 3. Objekt-Gates (hoch)

- [x] 3.1 `media.Serve`: Sichtbarkeit über Uploader / Konversations-Mitgliedschaft / `broadcast_reads`, sonst 404; Matrix-Eintrag korrigieren.
- [x] 3.2 `trainings.GetSession` → `hasKaderAccess`.
- [x] 3.3 `games.SaveLineup` → Trainer eines beteiligten Teams (Muster `canRecordGameAttendance`), admin/sL vereinsweit.
- [x] 3.4 `chat.MarkRead`, `chat.LeaveConversation` → `isActiveMember`, sonst 403.
- [x] 3.5 `duties.Fulfill`, `duties.CashSubstitute` → policy.CanFulfillAssignment (admin/trainer/sL, Bestandsrecht), `RowsAffected == 0` → 404, Exec-Fehler → 500.
- [x] 3.6 Tests je Route: berechtigt → wie bisher, fremdes Objekt → 403/404 (müssen ohne Fix rot sein).

## 4. HSTS und JWT_SECRET (hoch)

- [x] 4.1 `config.Load`: `JWT_SECRET` ≥ 32 Byte und ≠ `change-me-to-a-random-secret`, sonst Startabbruch mit Hinweis.
- [x] 4.2 `.env.example`: leerer Wert + Generator-Kommentar; `Makefile deploy`: `HSTS_ENABLED=true` idempotent ergänzen und `JWT_SECRET`-Länge auf dem Server vor dem Restart prüfen.
- [x] 4.3 Tests: Config lehnt kurzes und Beispiel-Secret ab; Security-Header-Test erwartet HSTS bei `hstsEnabled=true`.

## 5. Impersonation-Audit (hoch)

- [x] 5.1 Migration `060`: Kategorie `admin` im `user_events.category`-CHECK (Rebuild nach Muster `049`), down entfernt `admin`-Zeilen und stellt den alten CHECK her.
- [x] 5.2 `Claims.ImpersonatedBy` (`impersonated_by`, omitempty) in Token und Parsing; `Impersonate` setzt ihn, loggt `slog.Warn` und schreibt `eventlog.Record` an den Admin.
- [x] 5.3 Tests: Token trägt Claim; Event-Log-Zeile existiert; normales Login trägt keinen Claim.

## 6. Abhängigkeiten und CI (hoch)

- [x] 6.1 npm-Alerts schließen (`react-router` 7.18, `vitest` 4.1.11, `undici`, `fast-uri`, `browserslist`, `baseline-browser-mapping`, `postcss-selector-parser`), `pnpm build && pnpm test` grün.
- [x] 6.2 CI-Job `audit`: `govulncheck ./...` + `pnpm audit --audit-level=high`.

## 7. Verifikation

- [x] 7.1 `make test`, `make lint`, `pnpm -C web build/test/lint`, `make test-e2e` grün.
- [x] 7.2 `/verify-change`; Gotcha-Absatz „Upload-Pfad und Objektrechte" in `docs/agent/06-gotchas.md`; Deploy-Hinweis (`JWT_SECRET`, Tresor neu entsperren) in `10-deployment.md`.
