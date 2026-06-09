## 1. Backend: Token-Logik für SEPA-Download

- [x] 1.1 `internal/upload/sepa_token.go` erstellen: `sepaDownloadTokenPayload` (memberID, userID, exp), `generateSepaToken`, `validateSepaToken` — analog zu `internal/files/token.go`

## 2. Backend: Neue Endpoints in upload/handler.go

- [x] 2.1 `Handler`-Struct um `secret string` erweitern; `NewHandler` Signatur anpassen und in `main.go` übergeben
- [x] 2.2 `GET /api/members/{id}/sepa-mandat/download-token` implementieren: Berechtigung prüfen (isOwn, isParent via `family_links`, vorstand-Funktion, admin), Token generieren und zurückgeben
- [x] 2.3 `GET /api/members/{id}/sepa-mandat/download?token=...` implementieren: Token validieren, `sepa_mandat_path` aus DB lesen, Datei servieren (nicht über `/api/uploads/`)
- [x] 2.4 `DELETE /api/members/{id}/sepa-mandat` implementieren: Berechtigung prüfen (isOwn, isParent via `family_links`, vorstand, admin), Datei von Disk löschen, `sepa_mandat_path = NULL` setzen

## 3. Backend: Neue Endpoints in main.go registrieren

- [x] 3.1 `GET /api/members/{id}/sepa-mandat/download` als **öffentliche** Route registrieren (wie `/api/files/{id}/download`) — token-auth wird intern geprüft
- [x] 3.2 `GET /api/members/{id}/sepa-mandat/download-token` und `DELETE /api/members/{id}/sepa-mandat` im auth-geschützten Block registrieren

## 4. Backend: sepa_mandat_url Sichtbarkeit erweitern

- [x] 4.1 In `internal/members/handler.go` (ca. Zeile 424): Bedingung von `if isAdmin` auf `if isAdmin || isOwn || isParent || claims.HasFunction("vorstand")` erweitern
- [x] 4.2 URL von `/api/uploads/...` auf `/api/members/{id}/sepa-mandat/download` ändern (kein Token in der URL — der Client holt ihn separat)
- [x] 4.3 `isParent` berechnen: `SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?` (analog zu `isParentOf` in handler.go:1811)

## 5. Frontend: MemberDatenschutzTab — Prop und Upload

- [x] 5.1 `Props`-Interface um `memberId: number` erweitern; Funktionsparameter ergänzen
- [x] 5.2 In `handleSepaUpload`: `FormData` befüllen, `api.post(\`/upload/sepa-mandat/\${memberId}\`, formData)` aufrufen
- [x] 5.3 Bei Erfolg: `sepa_mandat_url` via `onFormChange({ sepa_mandat_url: url })` aktualisieren
- [x] 5.4 Fehlerstate `sepaUploadError` als `useState<string>` hinzufügen, bei Upload-Fehler setzen

## 6. Frontend: MemberDatenschutzTab — Token-Download und Löschen

- [x] 6.1 `openSepaMandat`-Funktion: `window.open('about:blank', '_blank')` synchron, dann `api.get(\`/members/\${memberId}/sepa-mandat/download-token\`)`, dann `tab.location.href` setzen; bei Fehler Tab schliessen + Fehlermeldung
- [x] 6.2 Öffnen-Button anzeigen wenn `form.sepa_mandat_url` gesetzt (Lucide `<ExternalLink>`-Icon)
- [x] 6.3 Löschen-Button anzeigen wenn `form.sepa_mandat_url` gesetzt UND Nutzer ist Admin, Vorstand, isOwn oder Elternteil (user-Kontext via `useAuth` prüfen; isParent wird analog zum Öffnen-Button bestimmt); nach Bestätigung `api.delete(\`/members/\${memberId}/sepa-mandat\`)` aufrufen und `sepa_mandat_url` via `onFormChange` leeren
- [x] 6.4 Fehler-Alert für Token-Fehler und Lösch-Fehler anzeigen

## 7. Frontend: MemberDetailPage anpassen

- [x] 7.1 `memberId={Number(id)}` an `<MemberDatenschutzTab>` übergeben

## 8. Frontend: DocumentsPage iOS-Fix

- [x] 8.1 In `DocumentsPage.tsx` `openFile`: `const tab = window.open('about:blank', '_blank')` synchron **vor** dem `await` einfügen
- [x] 8.2 `window.open(url, '_blank')` ersetzen durch `if (tab) tab.location.href = url`
- [x] 8.3 Im Catch-Block: `if (tab) tab.close()` vor `setFileError`
