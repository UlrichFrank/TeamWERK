## 1. Impersonation NULL-sicher machen (Backend)

- [x] 1.1 `Impersonate` in `internal/auth/handler.go`: SELECT auf `COALESCE(NULLIF(email,''), login_name, '')` umstellen (Identität in eine `string`-Variable scannen, kein NULL-Scan mehr)
- [x] 1.2 Aufgelöste Identität als Identitäts-Parameter an `IssueAccessToken(...)` übergeben (statt roher `email`)
- [x] 1.3 Test `TestImpersonate_ChildAccountWithoutEmail`: Kinder-Konto (`email NULL`, `login_name` gesetzt, `can_login=1`, role=standard) → 200, JWT-Identitäts-Claim = `login_name`
- [x] 1.4 Regression-Tests: `TestImpersonate_RegularUser` (Konto mit E-Mail → 200, Claim = E-Mail) und `TestImpersonate_AdminRejected` (role=admin → 400)

## 2. Lösch-Mutation broadcasten (Backend)

- [x] 2.1 `DeleteUser` in `internal/auth/handler.go`: nach `tx.Commit()` `h.hub.Broadcast("users")` aufrufen
- [x] 2.2 Sicherstellen, dass der `Handler` Zugriff auf `h.hub` hat (bereits vorhanden — verifizieren, nicht erweitern)
- [x] 2.3 Test `TestDeleteUser_Broadcast`: erfolgreiche Löschung → 204 und genau ein `Broadcast("users")` (Hub-Spy/Fake im Test)
- [x] 2.4 Test `TestDeleteUser_ChildAccount`: Kinder-Konto mit verknüpftem `members`-Datensatz → 204; `members`-Zeile bleibt mit `user_id = NULL`
- [x] 2.5 Regression-Test `TestDeleteUser_SelfRejected`: eigene userId → 400, kein Broadcast

## 3. Nutzerverwaltung aktualisiert sich (Frontend)

- [x] 3.1 `handleDeleteUser` in `web/src/pages/AdminUsersPage.tsx`: nach `await api.delete('/users/${u.id}')` `refreshUsers()` aufrufen (Fehlerfall sichtbar machen statt stiller Promise-Rejection)
- [x] 3.2 `useLiveUpdates`-Callback (Z. 130) erweitern: bei `event === 'users'` `refreshUsers()` aufrufen (bestehender `members`-Zweig bleibt unverändert)
- [ ] 3.3 Manuelle Verifikation: Kinder-Account anlegen/aktivieren → "Testen als" liefert die Kind-Session; "Löschen" entfernt die Zeile sofort

## 4. Abschluss

- [x] 4.1 `/verify-change` ausführen (Build/Test/Lint + Invarianten: Mutation→Broadcast, Route→Tests, brand-Tokens, lucide-Icons)
- [x] 4.2 `openspec validate fix-kinderaccount-nutzerverwaltung --strict`
- [x] 4.3 Conventional Commits je Task; abschließend Proposal archivieren
