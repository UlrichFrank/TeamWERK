## Context

Neue Mitglieder erhalten heute keine standardisierte Begrüßungsmail. Der Prozess liegt außerhalb von TeamWERK. Ziel ist es, die Mail direkt aus der App auslösen zu können, sobald ein Mitglied einen verknüpften Nutzeraccount hat — mit personalisierbarem Text, 3 festen PDF-Anhängen und Protokollierung des Versands.

Bestehende Infrastruktur: `internal/mailer` mit `mailer.Send(to, subject, body)` via `net/smtp`. Der Mailer unterstützt derzeit keine Anhänge — das muss erweitert werden. Die Anhänge (3 PDFs + 1 PNG-Logo) werden per `embed.FS` in die Binary eingebettet.

## Goals / Non-Goals

**Goals:**
- Manueller Versand einer personalisierten Willkommensmail per Knopfdruck im Admin-Tab
- Mail enthält: korrekte Anrede (Lieber/Liebe nach Geschlecht), Aufnahmedatum, Mitgliedsnummer, 3 PDF-Anhänge, Vereinssignatur mit Logo-Bild
- Versand wird mit Timestamp in der DB protokolliert (`welcome_email_sent_at`)
- Button ist nur aktiv wenn Nutzeraccount verknüpft; einmal verschickt → Button deaktiviert + Datum angezeigt

**Non-Goals:**
- Automatischer Versand (kein Trigger bei Mitglied-Erstellung)
- Bearbeitung des Mailtexts im UI
- Mehrfachversand / Re-Send
- Anhänge aus der DB oder einem Upload — sie sind zur Compile-Zeit fix eingebettet

## Decisions

**1. Anhänge über embed.FS, nicht DB-Upload**

Die 3 PDFs und das Logo sind stabile Dokumente (Satzung, Gebührenordnung, Leitbild). Sie werden unter `internal/mailer/attachments/` abgelegt und per `//go:embed` in die Binary eingebettet. Bei Dokumentänderungen ist ein neues Deployment nötig — das ist akzeptabel, da diese Dokumente selten wechseln.

Alternative (Upload-Mechanismus) würde eine signifikante neue Infrastruktur erfordern, die den Scope sprengt.

**2. Mailer-Erweiterung um MIME-Multipart-Unterstützung**

`mailer.Send` wird um eine neue Funktion `SendWithAttachments(to, subject, htmlBody string, attachments []Attachment)` erweitert. Die bestehende `Send`-Funktion bleibt unverändert. MIME multipart/mixed wird manuell aufgebaut — keine neue externe Bibliothek (Constraint: RAM, Binary-Größe).

Alternative (externe Mail-Bibliothek wie `gomail`) würde eine neue Dependency einführen.

**3. Handler in `internal/members/`, kein eigenes Package**

Der neue Endpunkt `POST /api/admin/members/{id}/welcome-email` passt thematisch zu den bestehenden Member-Handlern. Ein eigenes Package wäre Over-Engineering für einen einzelnen Endpunkt.

**4. Geschlecht → Anrede: explizit per `gender`-Feld**

`gender = 'm'` → „Lieber", `gender = 'f'` → „Liebe", alles andere → „Liebe/r" (geschlechtsneutral). Das Aufnahmedatum ist `join_date` aus dem Mitgliedsdatensatz; fehlt es, wird das aktuelle Datum verwendet.

## Risks / Trade-offs

- **Binary-Größe**: 3 PDFs + 1 PNG erhöhen die Binary um ~1–2 MB. Akzeptabel für VPS mit ausreichend Speicher.
- **Kein Re-Send**: Einmal versendet, kein zweiter Versand möglich (per Design). Wenn die Mail verloren geht, muss direkt aus dem E-Mail-Client gesendet werden.
- **Hardcodierter Mailtext**: Änderungen am Mailtext erfordern Code-Änderung und Deployment.
- **SMTP-Fehler**: Wenn der SMTP-Server nicht erreichbar ist, schlägt der Endpunkt mit 500 fehl und der Timestamp wird nicht gesetzt. Der Admin sieht die Fehlermeldung im UI.

## Migration Plan

1. Migration 013: `ALTER TABLE members ADD COLUMN welcome_email_sent_at TEXT` (nullable)
2. PDFs + PNG unter `internal/mailer/attachments/` ablegen
3. Mailer um `SendWithAttachments` erweitern
4. Handler `SendWelcomeEmail` in `internal/members/handler.go`
5. Route registrieren: `POST /api/admin/members/{id}/welcome-email` (admin only)
6. `MemberAdminTab.tsx` um Button + Status erweitern
7. `GET /api/members/{id}` gibt `welcome_email_sent_at` zurück

Rollback: Migration 013 rückgängig (`ALTER TABLE` in SQLite nicht direkt umkehrbar → Down-Migration erstellt neue Tabelle ohne Spalte). Der neue Endpunkt ist additiv, kein Breaking Change.

## Open Questions

- Soll die Mail an die E-Mail-Adresse des verknüpften `users`-Accounts gehen, oder gibt es eine separate Mitglieds-E-Mail? → Annahme: E-Mail des verknüpften User-Accounts (`users.email`).
- Soll das Logo inline (CID-Attachment) oder als normaler Anhang eingebunden sein? → Annahme: normaler Anhang, da einfacher zu implementieren und von den meisten Mail-Clients korrekt dargestellt.
