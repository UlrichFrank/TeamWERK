## Why

Die Profil-Seite ist inkonsistent: Vorname/Nachname wird im Konto-Tab direkt editiert (users-Tabelle), während für Mitglieder dieselben Daten im Mitgliedsdaten-Tab über Change-Requests verwaltet werden. IBAN erscheint im Profil-Tab als totes Formularfeld (wird beim Speichern ignoriert) und nochmals korrekt im Mitgliedsdaten-Tab. Nutzer, die Mitglieder sind, stoßen auf doppelte Editierwege und unklare Semantik.

## What Changes

- **Konto-Tab**: Vorname/Nachname entfernt — nur noch E-Mail (read-only) + Passwort/E-Mail-Aktionen
- **Profil-Tab**: Vorname/Nachname und IBAN (nur für verknüpfte Mitglieder) neu aufgenommen
  - Kein Mitglied verknüpft → „Speichern" speichert direkt in `users`
  - Mitglied verknüpft → „Änderung anfordern" sendet einen vollständigen Bundle-Request
- **Change-Request-Format** (neu): `field_name: "profil"` mit `new_value: { first_name, last_name, street, zip, city, iban }` — ein atomarer Request für alle Profilfelder
- **Mitgliedsdaten-Tab**: „Name ändern"- und „IBAN ändern"-Sektionen entfernt; stattdessen Anzeige offener Profil-Änderungsanfragen mit Zurückziehen-Option
- **Backend `PUT /profile/me`**: nimmt zusätzlich `first_name`, `last_name` entgegen (für nicht-verknüpfte Nutzer)
- **Admin-Ansicht** (Änderungsanfragen): zeigt `field_name: "profil"`-Requests als lesbares Bundle

## Capabilities

### New Capabilities

- `profil-change-request`: Neues atomares Change-Request-Format für alle Profilfelder eines Mitglieds in einem Bundle

### Modified Capabilities

- `name-aenderung`: Vorname/Nachname wandert aus dem Konto-Tab in den Profil-Tab; für Mitglieder läuft die Änderung über das neue Bundle-Format statt über feldweise Requests
- `familie-im-profil`: Mitgliedsdaten-Tab verliert die Editier-Sektionen (Name, IBAN); zeigt stattdessen offene Bundle-Anfragen

## Impact

- **Frontend**: `ProfileAccountTab.tsx`, `ProfileProfilTab.tsx`, `ProfileMemberTab.tsx`, `ProfilePage.tsx`
- **Backend**: `internal/members/handler.go` — `UpdateProfile`, `CreateChangeRequestHandler`, `AcceptChangeRequestHandler`
- **Admin-Ansicht**: `MemberDetailPage.tsx` oder Änderungsanfragen-Komponente muss `profil`-Bundle-Typ darstellen
- **Keine DB-Migration** nötig — `change_drafts`-Tabelle speichert `new_value` als JSON, neuer `field_name`-Wert reicht
