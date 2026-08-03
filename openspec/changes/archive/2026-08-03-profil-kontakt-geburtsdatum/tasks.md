## 1. Migration

- [x] 1.1 `internal/db/migrations/039_users_date_of_birth.up.sql`: `ALTER TABLE users ADD COLUMN date_of_birth TEXT;`
- [x] 1.2 `internal/db/migrations/039_users_date_of_birth.down.sql`: Spalte wieder entfernen (SQLite-Table-Rebuild ohne `date_of_birth`, analog vorheriger `users`-Spalten-Migrationen)

## 2. Backend — eigenes Profil

- [x] 2.1 `internal/members/handler.go`: `ProfileResponse` um `DateOfBirth string \`json:"date_of_birth,omitempty"\`` ergänzen
- [x] 2.2 `GetProfile`: Query um `users.date_of_birth` erweitern (COALESCE zu leerem String), `resp.DateOfBirth` befüllen
- [x] 2.3 `UpdateProfile`: Request-Struct um `DateOfBirth string \`json:"date_of_birth"\`` erweitern, `UPDATE users SET ...` um `date_of_birth=?` (via `nullableString`) ergänzen

## 3. Backend — Kind-Profil (User-Strang)

- [x] 3.1 `GetChildProfile`/`userContactEntry`: Struct und Query um `date_of_birth` erweitern (COALESCE zu leerem String)
- [x] 3.2 `UpdateChildAccount`: Request-Struct um `DateOfBirth` erweitern, `UPDATE users SET ...` um `date_of_birth=?` ergänzen

## 4. Backend — Änderungsantrag (profil-Draft)

- [x] 4.1 `internal/members/drafts.go`, `extractFieldValue` Case `"profil"`: `date_of_birth` aus `m.DateOfBirth` in die Map aufnehmen
- [x] 4.2 `internal/members/drafts.go`, `applyDraftToMember` Case `"profil"`: Struct und `UPDATE members SET ...` um `date_of_birth=?` erweitern

## 5. Frontend — Typen

- [x] 5.1 `web/src/pages/ChildProfilePage.tsx`: `UserContact`-Interface um `date_of_birth: string` erweitern

## 6. Frontend — Kontakt-Tab

- [x] 6.1 `web/src/components/profile/ProfileProfilTab.tsx`: neuen State `dateOfBirth` einführen
- [x] 6.2 Own-Mode-Init-Effekt: `setDateOfBirth((r.data?.date_of_birth || r.data?.own_member?.date_of_birth || '').slice(0, 10))`
- [x] 6.3 Child-Mode-Init-Effekt (`userContact`-Zweig): `setDateOfBirth((userContact.date_of_birth || ownMember?.date_of_birth || '').slice(0, 10))`; (`ownMember`-Fallback-Zweig ohne `userContact`): Feld wird nicht gerendert, State wird aber aus `ownMember.date_of_birth` mitgeführt (round-trip-sicher fürs Draft-Payload)
- [x] 6.4 Input-Feld `type="date"` unter dem PLZ/Ort-Grid in der Karte „Persönliche Daten" ergänzen
- [x] 6.5 Sichtbarkeit des neuen Feldes an die bestehende Bedingung `(mode !== 'child' || userContact)` UND `ownMember !== null` koppeln (`showDateOfBirth`)
- [x] 6.6 `childChanged`/`isChanged`-Diffing um `dateOfBirth` erweitern (Vergleich gegen `userContact?.date_of_birth` bzw. `ownMember?.date_of_birth`)
- [x] 6.7 `handleSave`: `date_of_birth` in `PUT /profile/me` bzw. `PUT /profile/kind/:id/account` UND in den `change-request`-Payload (`field_name: 'profil'`) aufnehmen

## 7. Frontend — Mitgliedsdaten-Tab (Diff-Anzeige)

- [x] 7.1 `web/src/components/profile/ProfileMemberTab.tsx`: `FIELD_LABELS` um `date_of_birth: 'Geburtsdatum'` ergänzen

## 8. Tests

- [x] 8.1 `internal/members/profil_geburtsdatum_test.go`: `UpdateProfile` — Happy-Path (Geburtsdatum wird in `users` übernommen) + Fehlerfall (401 ohne Token)
- [x] 8.2 `internal/members/profil_geburtsdatum_test.go`: `UpdateChildAccount` — Happy-Path (Geburtsdatum wird in `users` des Kindes übernommen) + Fehlerfall (403 ohne family_links-Eintrag)
- [x] 8.3 `internal/members/profil_geburtsdatum_test.go`: `field_name=profil`-Draft inkl. `date_of_birth` via `CreateOrUpdateDraft`/`AcceptDraft` — Wert wird auf `members.date_of_birth` übernommen
- [x] 8.4 `internal/members/profil_geburtsdatum_test.go`: `GetProfile`/`GetChildProfile` — Response enthält `date_of_birth` (Self-Service- und Mitglieder-Record-Wert)
- [x] 8.5 `web/src/components/profile/ProfileProfilTab.test.tsx` (neu angelegt): Default-Vorbelegung aus `ownMember.date_of_birth`, Save ruft beide Endpunkte mit `date_of_birth` auf, Gating (eigenes Profil ohne Mitglied / Kind ohne Account) geprüft

## 9. Verifikation

- [x] 9.1 `make test` (Go inkl. Architektur-/Broadcast-Gate) und `pnpm -C web build/test/lint` grün
- [x] 9.2 `/verify-change` durchlaufen (Route→Tests, Migrationsnummer, brand-Tokens, lucide-Icons)
