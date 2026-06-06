## Why

Die Nutzerverwaltung erfordert bisher das manuelle Einladen jedes Nutzers per E-Mail. Da die Vereinsmitgliederliste als CSV vorliegt, soll ein Bulk-Import direkt aus dieser Datei möglich sein — ohne automatischen E-Mail-Versand, damit der Admin kontrolliert, wer wann eingeladen wird.

## What Changes

- **Neu:** CSV-Import-Button in der Nutzerverwaltung ersetzt den bisherigen „+ Einladung"-Button
- **Neu:** CSV-Upload-Modal mit Preview (angelegt / übersprungen) und Ergebnisanzeige
- **Neu:** Backend-Endpoint `POST /api/admin/invitations/import-csv` — liest `Email` und `Email 2` aus CSV, legt `invitation_tokens` ohne E-Mail-Versand an
- **Neu:** „Einladung senden"-Aktion im ActionMenu pro Einladungs-Zeile (ersetzt bisherigen automatischen Versand)
- **Neu:** `invitation_tokens.member_id` (nullable FK) — ermöglicht Vorab-Verknüpfung mit einem Mitglied vor der Registrierung
- **Neu:** „Mit Mitglied verknüpfen"-Aktion im ActionMenu pro Einladungs-Zeile
- **Neu:** Register-Handler verknüpft beim Registrieren automatisch `members.user_id`, wenn der Token einen `member_id`-Eintrag hat
- **Neu:** `users.last_login_at` — wird bei jedem Login gesetzt und in der Nutzerverwaltung angezeigt
- **Entfernt:** Bisheriges „+ Einladung"-Modal (Einzel-Einladung per E-Mail als primärer Einstiegspunkt)

## Capabilities

### New Capabilities

- `csv-import`: CSV-Upload legt Einladungs-Tokens für alle unique E-Mails an (Email + Email 2), überspringt bereits vorhandene Nutzer und Einladungen, sendet keine E-Mails
- `invitation-member-link`: Einladungen können vor der Registrierung mit einem Mitglied verknüpft werden; beim Registrieren wird die Verknüpfung automatisch auf den neuen User übertragen
- `last-login-tracking`: Login-Zeitpunkt wird auf `users` gespeichert und in der Nutzerverwaltung angezeigt

### Modified Capabilities

## Impact

- **Backend:** Neuer Handler in `internal/auth/` (`ImportCSV`, `SendInvitation`), Änderung an `Register`-Handler (member-link), Änderung am Login-Handler (`last_login_at` setzen)
- **Migrationen:** Zwei neue `.up.sql`-Dateien — `invitation_tokens.member_id` + `users.last_login_at`
- **Frontend:** `AdminUsersPage.tsx` — Button-Ersatz, CSV-Modal, ActionMenu-Erweiterungen, neue Spalte
- **API:** Neuer Endpoint `POST /api/admin/invitations/import-csv`, neuer Endpoint `POST /api/admin/invitations/{id}/send`
