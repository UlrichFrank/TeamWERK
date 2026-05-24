## 1. Backend — Change-Request Bundle-Typ

- [x] 1.1 `allowedFields` in `CreateChangeRequestHandler` um `"profil"` erweitern
- [x] 1.2 `extractFieldValue` in `drafts.go` um `case "profil"` erweitern: liest `first_name`, `last_name`, `street`, `zip`, `city`, `iban` aus dem Member-Struct
- [x] 1.3 `applyDraftToMember` in `drafts.go` um `case "profil"` erweitern: schreibt alle Felder in einem `UPDATE members SET … WHERE id=?`

## 2. Backend — PUT /profile/me erweitern

- [x] 2.1 `UpdateProfile`-Handler: Request-Struct um `FirstName`, `LastName` erweitern
- [x] 2.2 UPDATE-Query in `UpdateProfile` um `first_name=?, last_name=?` erweitern

## 3. Frontend — Konto-Tab bereinigen

- [x] 3.1 `ProfileAccountTab.tsx`: Vorname/Nachname-Felder, State-Variablen, `GET /profile/account`-Call und Speichern-Button entfernen
- [x] 3.2 Speichern-Button und zugehörige Logik entfernen (nur noch Sicherheit-Aktionen bleiben)

## 4. Frontend — Profil-Tab umbauen

- [x] 4.1 `ProfileProfilTab.tsx`: Props um `ownMember: Member | null` erweitern (aus `ProfilePage`)
- [x] 4.2 State für `firstName`, `lastName`, `iban` hinzufügen; Initialwerte aus `profile/me` (`own_member`) bzw. `profile/account` laden
- [x] 4.3 Vorname/Nachname-Felder ins Formular aufnehmen (oberhalb Adresse)
- [x] 4.4 IBAN-Sektion konditionell rendern: nur wenn `ownMember !== null`; toten `iban`-State aus bisherigem Bankdaten-Block entfernen
- [x] 4.5 Speichern-Logik aufteilen:
      - Kein Mitglied: `PUT /profile/me` mit `{ first_name, last_name, street, zip, city }`
      - Mitglied: `POST /members/{id}/change-request` mit `{ field_name: "profil", new_value: { first_name, last_name, street, zip, city, iban } }`
- [x] 4.6 Button-Label dynamisch: „Speichern" (kein Mitglied) vs. „Änderung anfordern" (Mitglied)
- [x] 4.7 Bei offenem Draft (`profil`-Draft in Drafts vorhanden): Formular schreibgeschützt + Hinweis-Banner; Formular-Felder mit `new_value` des Drafts vorbelegen

## 5. Frontend — Mitgliedsdaten-Tab bereinigen

- [x] 5.1 `ProfileMemberTab.tsx`: „Name ändern"-Sektion vollständig entfernen (State, Handler, JSX)
- [x] 5.2 „IBAN"-Sektion vollständig entfernen (State, Handler, JSX)
- [x] 5.3 Sektion „Ausstehende Anfrage" hinzufügen: zeigt Drafts mit `field_name === "profil"` als lesbare Feldliste (alt → neu) mit „Zurückziehen"-Button
- [x] 5.4 `cancelError`-State vereinfachen (nur noch für Profil-Draft-Zurückziehen)

## 6. Frontend — ProfilePage koordinieren

- [x] 6.1 `ownMember` als Prop an `ProfileProfilTab` weitergeben
- [x] 6.2 Nach erfolgreichem Zurückziehen im Mitgliedsdaten-Tab: Profil-Tab-Formular wieder editierbar machen (Draft-State neu laden)
