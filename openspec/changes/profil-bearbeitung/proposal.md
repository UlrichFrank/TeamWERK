## Why

Eingeloggte Nutzer können ihren Anzeigenamen, ihre E-Mail-Adresse und ihr Passwort nicht selbst ändern — das Profil ist bisher rein lesend. Das zwingt Admins zu manuellen Eingriffen bei trivialen Konto-Änderungen.

## What Changes

- **Neu:** Anzeigenamen ändern (`PUT /api/profile/account`) — sofort wirksam
- **Neu:** Passwort ändern (`POST /api/profile/password`) — altes Passwort erforderlich, alle Sessions danach invalidiert
- **Neu:** E-Mail-Adresse ändern mit Bestätigungs-Workflow (`POST /api/profile/email` + `GET /api/profile/email/confirm`) — altes Passwort erforderlich, Bestätigungslink an neue Adresse, alle Sessions nach Bestätigung invalidiert
- **Neu:** Tabelle `email_change_tokens` für ausstehende E-Mail-Änderungen
- **Geändert:** Profilseite erhält editierbare Formularfelder für Name, E-Mail und Passwort

## Capabilities

### New Capabilities

- `name-aenderung`: Anzeigenamen des eigenen Kontos ändern
- `passwort-aenderung`: Passwort mit Verifikation des aktuellen Passworts ändern; invalidiert alle Refresh-Tokens
- `email-aenderung`: E-Mail-Adresse mit zweistufigem Bestätigungs-Workflow ändern; invalidiert alle Refresh-Tokens nach Bestätigung

### Modified Capabilities

_(keine bestehenden Specs betroffen)_

## Impact

- **Backend:** 3 neue Handler in `internal/auth/` (oder eigenes `internal/profile/`-Package), 1 neue Migration
- **Datenbank:** neue Tabelle `email_change_tokens` (analog zu `password_reset_tokens`)
- **Frontend:** `ProfilePage.tsx` erhält drei editierbare Sektionen
- **Auth-Flow:** Passwort- und E-Mail-Änderung erzwingen Re-Login (alle Refresh-Tokens gelöscht)
- **E-Mail-Versand:** Bestätigungs-Mail für E-Mail-Änderung via bestehendem Mailer
