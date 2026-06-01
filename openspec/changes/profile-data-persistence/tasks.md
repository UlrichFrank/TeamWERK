## 1. Backend: UpdateProfile reparieren

- [x] 1.1 In `internal/members/handler.go` → `UpdateProfile`: `nullableString` für `first_name` und `last_name` entfernen, Werte direkt übergeben (leerer String ist valide für NOT-NULL-Spalte)
- [x] 1.2 `ExecContext`-Fehler nicht mehr ignorieren: bei Fehler `500` zurückgeben statt lautlos `204`

## 2. Frontend: Ladelogik vereinfachen

- [x] 2.1 In `ProfileProfilTab.tsx` → zweites `useEffect`: `ownMember`-Zweig (der `setFirstName`, `setLastName`, `setAddress` aus `ownMember` setzt) entfernen; immer aus `/profile/account` laden; Dependency von `[ownMember?.id]` auf `[]` ändern
- [x] 2.2 Drittes `useEffect` (Change-Drafts): `setFirstName`, `setLastName`, `setAddress` und `setChanged(false)` aus dem Draft-Zweig entfernen; nur noch `setProfilDraft` setzen

## 3. Frontend: Save-Logik anpassen

- [x] 3.1 In `handleSave`: `if (ownMember) … else …` auflösen; `PUT /profile/me` wird immer als erster Call ausgeführt
- [x] 3.2 Für Nutzer mit verlinktem Mitglied: `POST change-request` bleibt erhalten, läuft aber nach `PUT /profile/me` als zweiter Call
- [x] 3.3 Button-Text: konditionelles `ownMember ? 'Änderung anfordern' : 'Speichern'` → immer `'Speichern'`
- [x] 3.4 Success-Text: konditionelles `ownMember ? 'Anfrage gesendet' : 'Gespeichert'` → immer `'Gespeichert'`
