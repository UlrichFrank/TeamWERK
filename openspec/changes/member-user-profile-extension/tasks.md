## 1. DB-Migrationen

- [ ] 1.1 `internal/db/migrations/017_member_extended.up.sql`: `ALTER TABLE members ADD COLUMN street TEXT`, zip TEXT, city TEXT, join_date DATE, iban TEXT, photo_path TEXT, dsgvo_verarbeitung INTEGER DEFAULT 0, dsgvo_verarbeitung_date DATE, dsgvo_weitergabe INTEGER DEFAULT 0, dsgvo_weitergabe_date DATE, sepa_mandat INTEGER DEFAULT 0, sepa_mandat_date DATE, sepa_mandat_path TEXT
- [ ] 1.2 `internal/db/migrations/017_member_extended.down.sql`: SQLite-Rebuild von `members` ohne die neuen Felder (PRAGMA foreign_keys=OFF, CREATE members_new, INSERT SELECT, DROP, RENAME)
- [ ] 1.3 `internal/db/migrations/018_user_contact.up.sql`: `ALTER TABLE users ADD COLUMN street TEXT, zip TEXT, city TEXT, photo_path TEXT`; `CREATE TABLE user_phones (id INTEGER PK AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, label TEXT NOT NULL, number TEXT NOT NULL, sort_order INTEGER DEFAULT 0)`; `CREATE TABLE user_visibility (user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, phones_visible INTEGER DEFAULT 0, address_visible INTEGER DEFAULT 0, photo_visible INTEGER DEFAULT 0)`
- [ ] 1.4 `internal/db/migrations/018_user_contact.down.sql`: DROP TABLE user_visibility; DROP TABLE user_phones; SQLite-Rebuild von `users` ohne street/zip/city/photo_path

## 2. Backend: File-Upload-Package

- [ ] 2.1 Neues Package `internal/upload/`: `Handler struct{ db *sql.DB; uploadDir string }`, `NewHandler(db, uploadDir)`
- [ ] 2.2 Hilfsfunktion `saveFile(r *http.Request, subdir string, allowedTypes []string, maxBytes int64) (filename string, err error)`: liest Multipart-Form, prüft Content-Type gegen allowedTypes, Größe gegen maxBytes, generiert UUID-Dateiname, erstellt Subdir via `os.MkdirAll`, schreibt Datei
- [ ] 2.3 `UploadMemberPhoto(w, r)`: Auth-required, Admin-only; liest `{id}`, ruft `saveFile("member-photos/", imageTypes, 5 MB)`, löscht altes Foto wenn vorhanden (`os.Remove`), setzt `members.photo_path`, gibt `{photo_url}` zurück
- [ ] 2.4 `UploadUserPhoto(w, r)`: Auth-required, eigener Nutzer; ruft `saveFile("user-photos/", imageTypes, 5 MB)`, löscht altes Foto, setzt `users.photo_path`
- [ ] 2.7 `UploadSepaMandat(w, r)`: Auth-required, Admin-only; liest `{id}`, ruft `saveFile("sepa-mandats/", pdfAndImageTypes, 10 MB)`, löscht altes Dokument wenn vorhanden, setzt `members.sepa_mandat_path`, gibt `{sepa_mandat_url}` zurück
- [ ] 2.5 `ServeUpload(w, r)`: Auth-required; liest `{*}` Wildcard-Pfad, prüft auf Path-Traversal (`strings.Contains(path, "..")`), liefert Datei via `http.ServeFile`
- [ ] 2.6 Env-Variable `UPLOAD_DIR` in `internal/config/` ergänzen (Default `./storage/uploads/`)

## 3. Backend: Members-API erweitern

- [ ] 3.1 `Member`-Struct in `internal/members/handler.go` um alle neuen Felder erweitern (`*string` für nullable Text, `*bool`+`*string` für DSGVO/SEPA Flags+Dates, `PhotoURL *string`, `SepaMandat_url *string`, `AddressSource *string`)
- [ ] 3.2 `scanMember` um alle neuen Felder erweitern (sql.NullString, sql.NullInt64 für bool-Felder, sepa_mandat_path)
- [ ] 3.3 `GetMember`: Adress-Fallback-Logik implementieren (LEFT JOIN users für Adresse; wenn members.street NULL → user-Adresse verwenden, address_source setzen); IBAN + sepa_mandat_url nur für Admin; Adresse/DSGVO/SEPA-bool nur wenn Admin ODER eigener Nutzer; photo_url immer für alle authentifizierten
- [ ] 3.4 `UpdateMember`: neue Felder aus Request lesen und in UPDATE schreiben; IBAN-Update nur wenn Admin
- [ ] 3.5 `GetMemberParents` (Family-Links): wenn `claims.Role == "elternteil"` → WHERE parent_user_id = claims.UserID; sonst alle Links

## 4. Backend: User/Profile-API erweitern

- [ ] 4.1 `GetProfile` (`GET /api/profile/me`): `street`, `zip`, `city`, `photo_url`, `phones`-Array, `visibility`-Objekt zurückgeben; JOIN user_phones + LEFT JOIN user_visibility
- [ ] 4.2 `UpdateProfile` (`PUT /api/profile/me`): `street`, `zip`, `city` aus Body lesen und in `users` schreiben
- [ ] 4.3 Neuer Handler `AddPhone` (`POST /api/profile/phones`): INSERT in user_phones, gibt neue ID zurück
- [ ] 4.4 Neuer Handler `DeletePhone` (`DELETE /api/profile/phones/{id}`): prüft `user_phones.user_id == claims.UserID`, löscht Eintrag
- [ ] 4.5 Neuer Handler `UpdatePhone` (`PUT /api/profile/phones/{id}`): prüft Ownership, updated label/number
- [ ] 4.6 Neuer Handler `UpdateVisibility` (`PUT /api/profile/visibility`): INSERT OR REPLACE in user_visibility
- [ ] 4.7 `GetMember` für Nicht-Admin: wenn `phones_visible=true` beim verknüpften Nutzer → phones-Array anhängen; analog für address/photo

## 5. Backend: Router-Registrierung

- [ ] 5.1 In `cmd/teamwerk/main.go` upload.Handler initialisieren (mit uploadDir aus Config)
- [ ] 5.2 Routen registrieren: `POST /api/upload/member-photo/{id}` (Admin), `POST /api/upload/user-photo` (Auth), `POST /api/upload/sepa-mandat/{id}` (Admin), `GET /api/uploads/*` (Auth)
- [ ] 5.3 Routen registrieren: `POST /api/profile/phones`, `PUT /api/profile/phones/{id}`, `DELETE /api/profile/phones/{id}`, `PUT /api/profile/visibility`

## 6. Frontend: MemberDetailPage erweitern

- [ ] 6.1 Member-Interface um alle neuen Felder erweitern (`street`, `zip`, `city`, `join_date`, `iban`, `photo_url`, `dsgvo_verarbeitung`, `dsgvo_verarbeitung_date`, `dsgvo_weitergabe`, `dsgvo_weitergabe_date`, `sepa_mandat`, `sepa_mandat_date`)
- [ ] 6.2 Neuen Abschnitt "Adresse & Kontakt" in Stammdaten: Felder street, zip, city (Admin-editierbar); wenn `address_source === "user"` → Hinweis "Übernommen vom Nutzerprofil" anzeigen
- [ ] 6.3 Feld "Eintrittsdatum" (date-Input, Admin-editierbar)
- [ ] 6.4 Feld "IBAN" (Text-Input, nur sichtbar wenn `user.role === 'admin'`)
- [ ] 6.5 Abschnitt "DSGVO & SEPA": drei Toggle-Checkboxen + Datumsfelder (Admin-editierbar); verknüpfter Nutzer sieht SEPA-bool + Datum read-only (kein Dokument); Admin sieht zusätzlich SEPA-Dokument-Upload-Button + Link zum Dokument (`sepa_mandat_url`)
- [ ] 6.6 Foto-Bereich: zeigt `photo_url` als `<img>` wenn vorhanden; Admin sieht Upload-Button (`<input type="file">` → POST /api/upload/member-photo/{id})

## 7. Frontend: Profil-Seite (Nutzer-Kontaktdaten)

- [ ] 7.1 Neue Seite `web/src/pages/ProfilePage.tsx` oder bestehende Profilseite erweitern um Abschnitt "Kontaktdaten"
- [ ] 7.2 Telefonnummern-Liste: zeigt vorhandene Nummern mit Label; "Hinzufügen"-Button öffnet Inline-Formular (Label-Select mit Vorschlägen + Freitext + Nummer-Input → POST /api/profile/phones); × löscht Eintrag
- [ ] 7.3 Adresse: drei Felder (Straße, PLZ, Ort) → PUT /api/profile/me
- [ ] 7.4 Profilbild: zeigt photo_url oder Platzhalter; Upload-Button → POST /api/upload/user-photo
- [ ] 7.5 Sichtbarkeits-Toggles: drei Checkboxen (Telefon / Adresse / Foto sichtbar für Teammitglieder) → PUT /api/profile/visibility

## 8. Deploy & Infrastruktur

- [ ] 8.1 `deploy/setup-vps.sh` ergänzen: `mkdir -p /var/lib/teamwerk/storage/uploads/{member-photos,user-photos,sepa-mandats} && chown -R www-data:www-data /var/lib/teamwerk/storage/`
- [ ] 8.2 `.env.example` um `UPLOAD_DIR=/var/lib/teamwerk/storage/uploads` ergänzen
- [ ] 8.3 `make deploy` — Migrationen 017 + 018 laufen automatisch; Upload-Verzeichnis auf VPS anlegen (einmalig manuell oder via setup-vps.sh)
