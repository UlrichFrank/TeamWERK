## 1. Datenbank

- [x] 1.1 Migration `013_welcome_email.up.sql` anlegen: `ALTER TABLE members ADD COLUMN welcome_email_sent_at TEXT`
- [x] 1.2 Migration `013_welcome_email.down.sql` anlegen (Tabelle ohne Spalte neu erstellen)

## 2. Anhänge einbetten

- [x] 2.1 Verzeichnis `internal/mailer/attachments/` anlegen
- [x] 2.2 PDFs aus `~/Downloads` kopieren: `25-Satzung-Neufassung-Stand_221026.pdf`, `20260424ANLAGE 1-BO.pdf`, `20260424Leitbild.pdf`
- [x] 2.3 Logo-SVG aus `web/dist/logo.svg` als `logo.svg` in `internal/mailer/attachments/` ablegen
- [x] 2.4 `//go:embed`-Direktive in einem neuen `attachments.go` in `internal/mailer/` hinzufügen, das die Dateien als `embed.FS` exportiert

## 3. Mailer erweitern

- [x] 3.1 Typ `Attachment` in `internal/mailer/` definieren: `{Filename string; Data []byte; MIMEType string}`
- [x] 3.2 Funktion `SendWithAttachments(to, subject, body string, attachments []Attachment) error` implementieren — MIME multipart/mixed manuell aufbauen, keine neue Dependency

## 4. Backend — Handler

- [x] 4.1 `welcome_email_sent_at` (als `sql.NullString`) in den `Member`-Struct und `scanMember`/`Get`-Handler von `internal/members/handler.go` aufnehmen; `GET /api/members/{id}` gibt `welcome_email_sent_at` zurück
- [x] 4.2 Handler `SendWelcomeEmail` in `internal/members/welcome_email.go` implementieren:
  - Mitglied + verknüpfte User-E-Mail laden
  - Prüfen: User vorhanden, noch kein Versand (`welcome_email_sent_at` IS NULL)
  - Anrede nach `gender` bestimmen (`m` → Lieber, `f` → Liebe, sonst Liebe/r)
  - Datum aus `join_date` (oder heute) formatieren (DD.MM.YYYY)
  - Mailtext mit Platzhaltern befüllen
  - `SendWithAttachments` aufrufen mit den 4 Anhängen (3 PDFs + Logo)
  - Bei Erfolg: `UPDATE members SET welcome_email_sent_at = ? WHERE id = ?`
- [x] 4.3 Route registrieren in `cmd/teamwerk/main.go`: `POST /api/admin/members/{id}/welcome-email` (admin only)

## 5. Frontend — MemberAdminTab

- [x] 5.1 `welcome_email_sent_at` zum `MemberDetail`-Interface in `MemberDetailPage.tsx` (oder dem relevanten Typ) hinzufügen
- [x] 5.2 `MemberAdminTab.tsx` erweitern: neuer Abschnitt „Willkommensmail"
  - Button „Willkommensmail senden" — aktiv wenn `currentUserId != null && !welcome_email_sent_at`
  - Wenn `welcome_email_sent_at` gesetzt: grüner Hinweis mit Versanddatum statt Button
  - Wenn kein User verknüpft: Button deaktiviert mit Hinweis „Bitte zuerst Nutzeraccount verknüpfen"
- [x] 5.3 Click-Handler: `POST /api/admin/members/{id}/welcome-email`, bei Erfolg State aktualisieren, bei Fehler Fehlermeldung im UI

## 6. Manuelle Verifikation

- [ ] 6.1 Lokal testen: Mitglied mit verknüpftem User-Account → Button klicken → Mail in Postfach prüfen (Anhänge, Anrede, Datum, Mitgliedsnummer)
- [ ] 6.2 Prüfen, dass Button nach Versand deaktiviert ist und Datum anzeigt
- [ ] 6.3 `make deploy` + Smoke-Test auf Produktion
