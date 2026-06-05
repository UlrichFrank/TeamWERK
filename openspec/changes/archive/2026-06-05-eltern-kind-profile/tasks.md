## 1. Backend: Autorisierungs-Hilfsfunktion

- [x] 1.1 Hilfsfunktion `isParentOf(ctx, db, parentUserID, memberID int) bool` in `internal/members/handler.go` schreiben: führt `SELECT COUNT(*) FROM family_links WHERE parent_user_id=? AND member_id=?` aus, gibt true zurück wenn Ergebnis > 0
- [x] 1.2 Sicherstellen, dass `isParentOf` als **erste Operation** in jedem Kind-Handler aufgerufen wird — vor jedem DB-Lese- oder Schreibzugriff; bei false sofort `http.Error(w, "forbidden", 403)` + return

## 2. Backend: Kindprofil lesen

- [x] 2.1 Handler `GetChildProfile` anlegen: `GET /api/profile/kind/:memberId` — ruft `isParentOf` auf (403 bei false), gibt Member-Daten (inkl. address, IBAN, account_holder) als JSON zurück
- [x] 2.2 Route `GET /api/profile/kind/{memberId}` im Router (auth-geschützt) registrieren

## 3. Backend: Kindprofil bearbeiten

- [x] 3.1 Handler `UpdateChildMember` anlegen: `PUT /api/profile/kind/:memberId/member` — ruft `isParentOf` auf (403 bei false), schreibt first_name, last_name, date_of_birth, jersey_number, position, street, zip, city in `members`; status-Feld wird nicht angenommen
- [x] 3.2 Handler `UpdateChildBank` anlegen: `PUT /api/profile/kind/:memberId/bank` — ruft `isParentOf` auf (403 bei false), schreibt iban, account_holder in `members`
- [x] 3.3 Routen `PUT /api/profile/kind/{memberId}/member` und `PUT /api/profile/kind/{memberId}/bank` im Router registrieren

## 4. Frontend: Kindprofil-Seite

- [x] 4.1 `ChildProfilePage.tsx` anlegen: lädt `GET /api/profile/kind/:memberId`, bei HTTP 403 sofort zur Startseite (`/`) weiterleiten (kein Profilinhalt rendern)
- [x] 4.2 Tab-Komponenten (`ProfileProfilTab`, `ProfileMemberTab`, `ProfileBankTab`) mit konfigurierbarem API-Pfad versehen (Prop `basePath`) oder via Props direkt mit Daten beliefern
- [x] 4.3 Im Kontakt-Tab: Adresse über `PUT /api/profile/kind/:memberId/member` speichern; Telefonnummer-Abschnitt nur anzeigen wenn `member.user_id != null`
- [x] 4.4 Im Mitgliedsdaten-Tab: Felder via `PUT /api/profile/kind/:memberId/member` speichern; Status-Feld ausblenden (kein Edit für Eltern)
- [x] 4.5 Im Bankdaten-Tab: IBAN/Kontoinhaber via `PUT /api/profile/kind/:memberId/bank` speichern
- [x] 4.6 Seitenüberschrift zeigt „[Vorname]s Profil"

## 5. Frontend: Navigation

- [x] 5.1 In `AppShell.tsx` beim Mount für `elternteil`-Nutzer `GET /api/profile/me` aufrufen und `children`-Liste im State halten — diese Liste ist die einzige Quelle für anzeigbare Kinder
- [x] 5.2 Im Nutzer-Modul der Sidebar dynamische Kind-Einträge unterhalb von „Mein Profil" rendern: Label „[Vorname]s Profil", Link zu `/profil/kind/:memberId`
- [x] 5.3 Kind-Einträge nur rendern wenn `user.role === 'elternteil'` und `children.length > 0`

## 6. Frontend: Route registrieren

- [x] 6.1 Route `/profil/kind/:memberId` in `App.tsx` unter dem `AppShell`-Outlet anlegen und `ChildProfilePage` zuweisen
