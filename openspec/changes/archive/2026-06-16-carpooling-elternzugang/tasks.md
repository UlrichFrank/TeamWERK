## 1. Backend: Hilfsfunktionen in carpooling/handler.go

- [x] 1.1 `ChildUser`-Struct und `Children []ChildUser`-Feld in `ListResponse` ergänzen
- [x] 1.2 `childUsers(ctx, parentUserID int) []ChildUser` implementieren (Query über `family_links JOIN members JOIN users`)
- [x] 1.3 `isChildOf(ctx, parentUserID, targetUserID int) bool` implementieren (COUNT-Query über `family_links JOIN members`)

## 2. Backend: List-Endpoint anpassen

- [x] 2.1 In `List()`: `childUsers()` einmalig laden und in `ListResponse.Children` setzen
- [x] 2.2 `childIDSet map[int]bool` aus den geladenen Kind-Usern ableiten und an `queryEntries` + `queryPaarungen` übergeben
- [x] 2.3 In `queryEntries()`: Signatur um `childIDSet map[int]bool` erweitern; `e.IsOwn = ownerID == currentUserID || childIDSet[ownerID]`
- [x] 2.4 In `queryPaarungen()`: Signatur erweitern; `BieteIsOwn`/`SucheIsOwn` analog mit childIDSet setzen

## 3. Backend: Upsert-Endpoint anpassen

- [x] 3.1 `ForUserID *int` als optionalen JSON-Feld in den Request-Body von `Upsert()` aufnehmen
- [x] 3.2 Wenn `ForUserID` gesetzt: `isChildOf()` prüfen, bei Fehler 403 zurückgeben; `userID = *ForUserID` für alle DB-Operationen

## 4. Backend: Delete-Endpoint anpassen

- [x] 4.1 In `Delete()`: Eintrag-Owner laden; Zugriff erlauben wenn `ownerID == claims.UserID || isChildOf(ctx, claims.UserID, ownerID)`

## 5. Backend: Paarungsanfragen anpassen

- [x] 5.1 In `RequestPairing()`: Autorisierung erweitern — User darf initiieren wenn er Bieter, Sucher, Elternteil des Bieters ODER Elternteil des Suchers ist
- [x] 5.2 In `ConfirmPairing()`: Autorisierung erweitern — User darf bestätigen wenn er die Gegenseite ist ODER Elternteil der Gegenseite
- [x] 5.3 In `RejectPairing()`: Autorisierung erweitern — User darf ablehnen wenn er Bieter, Sucher, Elternteil des Bieters ODER Elternteil des Suchers ist

## 6. Backend: Dashboard anpassen

- [x] 6.1 In `queryCarpoolingConfirmed()`: WHERE-Clause um Kind-Subquery erweitern (`OR mb.user_id IN (SELECT m.user_id FROM family_links fl JOIN members m ON m.id = fl.member_id WHERE fl.parent_user_id = ?) OR ms.user_id IN (...)`)

## 7. Frontend: MitfahrgelegenheitenPage.tsx

- [x] 7.1 `ChildUser`-Interface (`userId: number`, `name: string`) und `children: ChildUser[]` in `ListResponse`-Interface ergänzen
- [x] 7.2 `childIdSet: Set<number>` aus `response.children` ableiten
- [x] 7.3 `mineMatches()` erweitern: `childIdSet.has(e.userId)` und `childIdSet.has(p.bieteUserId || p.sucheUserId)` prüfen
- [x] 7.4 In `FormModal`: `forUserId?: number`-Prop ergänzen; beim `api.post('/mitfahrgelegenheiten', ...)` ggf. `forUserId` mitsenden
- [x] 7.5 Neues `ForWhomSelector`-Element im `FormModal`: nur rendern wenn `children.length > 0`; Optionen: „Ich" (kein forUserId) + je ein Kind; State: `selectedUserId`
- [x] 7.6 `GameCard` und `MitfahrgelegenheitenPage` die `children`-Liste an `FormModal` durchreichen

## 8. Tests

- [x] 8.1 Test: Elternteil legt Suche-Eintrag für Kind an (201), fremde userId → 403
- [x] 8.2 Test: Elternteil löscht Kind-Eintrag (204), fremden Eintrag → 403
- [x] 8.3 Test: Elternteil stellt Paarungsanfrage für Kind (204), ohne Bezug → 403
- [x] 8.4 Test: Elternteil bestätigt Paarung für Kind (204)
- [x] 8.5 Test: `GET /api/mitfahrgelegenheiten` — `isOwn=true` für Kind-Eintrag wenn Elternteil anfrägt
- [x] 8.6 Test: `GET /api/dashboard` — Kind-Paarung erscheint in `carpoolingConfirmed` des Elternteils
