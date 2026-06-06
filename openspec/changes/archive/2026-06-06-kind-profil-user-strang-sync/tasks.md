## 1. Backend: GET /profile/kind/{memberId} erweitern

- [x] 1.1 `GetChildProfile` liest `user_id` des Members; wenn vorhanden: `users.first_name`, `last_name`, `street`, `zip`, `city` des Kindes laden
- [x] 1.2 `user_phones` des Kind-Users laden (Query auf `user_phones WHERE user_id = child.user_id`)
- [x] 1.3 `user_visibility` des Kind-Users laden (Query auf `user_visibility WHERE user_id = child.user_id`, Fallback alle `false`)
- [x] 1.4 Response-Struct um `UserContact`-Objekt erweitern (nullable) und befüllen

## 2. Backend: Neuer Endpoint PUT /profile/kind/{memberId}/account

- [x] 2.1 Handler `UpdateChildAccount` in `internal/members/handler.go` anlegen
- [x] 2.2 `isParentOf`-Check + Prüfung ob Kind `user_id` hat (sonst HTTP 404)
- [x] 2.3 `UPDATE users SET first_name=?, last_name=?, street=?, zip=?, city=? WHERE id = child.user_id`
- [x] 2.4 Route `r.Put("/api/profile/kind/{memberId}/account", membH.UpdateChildAccount)` in `main.go` registrieren

## 3. Backend: Phones-Endpunkte auf user_phones umstellen

- [x] 3.1 `AddChildPhone`: Kind-`user_id` laden; wenn NULL → HTTP 403; wenn vorhanden → `INSERT INTO user_phones (user_id, ...)`
- [x] 3.2 `DeleteChildPhone`: Kind-`user_id` laden; wenn NULL → HTTP 403; wenn vorhanden → `DELETE FROM user_phones WHERE id=? AND user_id=?`

## 4. Backend: Visibility-Endpoint auf user_visibility umstellen

- [x] 4.1 `UpdateChildVisibility`: Kind-`user_id` laden; wenn NULL → HTTP 403; wenn vorhanden → UPSERT auf `user_visibility (user_id, phones_visible, address_visible, photo_visible, email_visible)`
- [x] 4.2 Sicherstellen dass die `user_visibility`-Tabelle UPSERT unterstützt (`INSERT OR REPLACE` oder `ON CONFLICT DO UPDATE`)

## 5. Frontend: ChildProfilePage und ProfileProfilTab anpassen (nur Datenbindung — kein Layout-Umbau)

- [x] 5.1 `ChildProfilePage`: `member.user_id` aus API-Response auslesen und als Prop weitergeben; Layout und Tabs bleiben unverändert
- [x] 5.2 `ProfileProfilTab` (child mode): Wenn `user_contact` im Response vorhanden, Initialwerte für Name/Adresse/Phones/Visibility daraus laden statt aus `ownMember`
- [x] 5.3 `ProfileProfilTab` (child mode, handleSave): `PUT /profile/kind/{id}/account` aufrufen wenn `user_id` vorhanden (zusätzlich zum bestehenden change-request); kein neuer Button oder UI-Element
- [x] 5.4 Phones in child mode: `user_contact.phones` verwenden wenn vorhanden; Telefonnummern-Abschnitt ausblenden wenn kein `user_contact` (Kind ohne Account)
- [x] 5.5 Visibility in child mode: Initialwerte aus `user_contact.visibility` laden wenn vorhanden; Sichtbarkeits-Abschnitt ausblenden wenn kein `user_contact`

## 6. Verifikation

- [ ] 6.1 Kind mit User-Account: Speichern aktualisiert `users`-Tabelle sofort und erstellt Change-Draft
- [ ] 6.2 Kind mit User-Account: Telefonnummer hinzufügen landet in `user_phones`
- [ ] 6.3 Kind mit User-Account: Visibility-Änderung landet in `user_visibility`
- [ ] 6.4 Kind ohne User-Account: Phones- und Visibility-Endpoints geben HTTP 403 zurück; UI blendet diese Abschnitte aus
