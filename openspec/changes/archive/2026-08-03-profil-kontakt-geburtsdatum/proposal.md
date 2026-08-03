## Why

Mitglieder mit einem direkt verknüpften Nutzer-Account (eigener Login, oder Kind mit eigenem Account) können im Profil unter „Kontakt" bereits Name und Adresse selbst pflegen — die Änderung läuft als Änderungsantrag (`member_change_drafts`, `field_name='profil'`) über eine Vorstands-Freigabe. Das Geburtsdatum fehlt in diesem Formular komplett; es ist aktuell nur unter „Mitgliedsdaten" sichtbar und dort ausschließlich vom Vorstand direkt editierbar. Mitglieder, die ein fehlerhaftes oder fehlendes Geburtsdatum bemerken, haben keinen Weg, das selbst anzustoßen.

## What Changes

- Neues Eingabefeld „Geburtsdatum" im Kontakt-Tab (`ProfileProfilTab`), direkt unter PLZ/Ort, für Nutzer mit direkt verknüpftem Mitgliederaccount (`ownMember !== null`, bei Kind-Profilen zusätzlich `userContact !== null`).
- Verhalten identisch zu Straße/PLZ/Ort: sofortiger Write auf eine neue Spalte `users.date_of_birth` (Self-Service-Kopie) **und** Aufnahme in den bestehenden `profil`-Änderungsantrag ans Mitglieder-Record (`members.date_of_birth`, Vorstands-Freigabe).
- Default-Anzeige: solange `users.date_of_birth` noch nie explizit gesetzt wurde (NULL), zeigt das Feld das aktuelle Geburtsdatum aus dem Mitgliederbereich (`members.date_of_birth`) vor. Nach dem ersten expliziten Speichern gilt ausschließlich der eigene `users.date_of_birth`-Wert.
- Die „Ausstehende Anfrage"-Anzeige im Mitgliedsdaten-Tab (`ProfileMemberTab`) zeigt die Geburtsdatum-Änderung automatisch mit an (generische Diff-Liste über `FIELD_LABELS`).
- Keine neue Formatvalidierung — Geburtsdatum verhält sich wie Adresse: kein Mindestalter- oder Plausibilitäts-Check auf dieser Route (das native `<input type="date">` erzwingt bereits ein gültiges Kalenderdatum).

## Capabilities

### New Capabilities
- `profil-kontakt-geburtsdatum`: Selbstauskunfts-Geburtsdatum im Kontakt-Tab (eigenes Profil) — neue `users.date_of_birth`-Spalte, `GET/PUT /api/profile/me`-Erweiterung, Aufnahme in den `profil`-Änderungsantrag (`extractFieldValue`/`applyDraftToMember`), Default-aus-Mitgliederbereich-Regel.

### Modified Capabilities
- `kind-profil`: Kontakt-Tab des Kindprofils zeigt/bearbeitet Geburtsdatum zusätzlich zu Name/Adresse, wenn das Kind einen eigenen User-Account hat.
- `kind-profil-user-strang`: `GET /api/profile/kind/:memberId` (`user_contact`) und `PUT /api/profile/kind/:memberId/account` um `date_of_birth` erweitert.

## Impact

- **Migration**: `internal/db/migrations/039_users_date_of_birth.up/down.sql` — `ALTER TABLE users ADD COLUMN date_of_birth TEXT;`
- **Backend**: `internal/members/handler.go` (`ProfileResponse`, `GetProfile`, `UpdateProfile`, `GetChildProfile`/`userContactEntry`, `UpdateChildAccount`), `internal/members/drafts.go` (`extractFieldValue`, `applyDraftToMember`, Case `"profil"`).
- **Frontend**: `web/src/components/profile/ProfileProfilTab.tsx` (neues Feld + Diffing + Save), `web/src/pages/ChildProfilePage.tsx` (`UserContact`-Interface), `web/src/components/profile/ProfileMemberTab.tsx` (`FIELD_LABELS`).
- **Tests**: Happy-Path + Fehlerfall für `UpdateProfile`, `UpdateChildAccount`, `POST /members/:id/change-request` (Feld `profil` inkl. `date_of_birth`), `AcceptDraft`/`applyDraftToMember`.
- Keine Breaking Changes, keine neuen externen Abhängigkeiten.
