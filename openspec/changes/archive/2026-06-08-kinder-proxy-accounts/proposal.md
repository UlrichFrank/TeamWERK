## Why

Kinder spielen im Verein, haben aber keine eigene E-Mail-Adresse — insbesondere Jüngere. Ohne einen eigenen Nutzeraccount kann ein Kind weder Trainings- noch Spielzusagen machen, und sein Elternteil kann das auch nicht stellvertretend tun. Noch gravierender: das Dienstsystem (`duty_assignments`, `duty_accounts`) ist vollständig nutzer-zentrisch — ohne `user_id` existiert das Kind im Pflichtsystem nicht.

Das aktuelle Datenmodell setzt für jeden Mitspieler einen vollwertigen Account mit gültiger E-Mail-Adresse voraus. Das ist für Erwachsene sinnvoll, schließt aber Kinder ohne eigenen E-Mail-Account strukturell aus.

## What Changes

**Proxy-Accounts (Option D):** Jedes Kind erhält einen `users`-Datensatz mit `can_login = 0`. Dieser Account hat keine Login-Funktion — er dient als Ankerpunkt für das Dienstsystem und die RSVP-Logik. Die E-Mail-Adresse ist optional; ist sie gesetzt, kann sie mit dem Eltern-Account übereinstimmen.

**Variante 2 — Elternteil wählt beim Dienst-Claim:** Wenn ein Elternteil auf der Dienstbörse einen Dienst übernimmt, erscheint ein „Für wen?"-Selektor. Optionen: sich selbst oder jedes verknüpfte Kind mit Proxy-Account. Der Dienst wird dem gewählten `user_id` zugebucht — das Kind baut damit sein eigenes Dienstkonto auf.

### Konkrete Änderungen

- `users` erhält Spalte `can_login INTEGER NOT NULL DEFAULT 1`; `email` wird nullable und nicht mehr global UNIQUE
- Neuer partieller Unique-Index: `UNIQUE ON users(email) WHERE can_login = 1 AND email IS NOT NULL`
- Login-Query, Passwort-Reset und E-Mail-Eindeutigkeitsprüfungen filtern auf `can_login = 1`
- Neuer Admin-Flow: Proxy-Account für ein Mitglied anlegen (aus dem Familie-Tab oder Mitglieds-Admin)
  - Setzt `members.user_id` auf den neuen Proxy-Account
  - Damit greift die bestehende `family_links`-Logik (RSVP für Kinder) sofort
- Dienstbörse (`DutyPage`): Claim-Button öffnet bei Elternteilen einen „Für wen?"-Dialog
  - Default: eigener Account
  - Weitere Optionen: verknüpfte Kinder mit aktivem Proxy-Account
- Aktivierungsflow: Wenn das Kind älter wird, kann ein Admin `can_login = 1` + E-Mail setzen und eine Einladung verschicken → das Kind übernimmt seinen Account selbst

## Capabilities

### New Capabilities

- `kinder-proxy-account`: Anlegen und Verwalten von Login-losen Nutzeraccounts für Kinder ohne E-Mail
- `dienst-fuer-familienmitglied`: Elternteil beansprucht Dienst stellvertretend für ein Kind; Dienstkonto des Kindes wird belastet

### Modified Capabilities

- `auth-login`: Login schlägt für Proxy-Accounts fehl (`can_login = 0`); E-Mail-Eindeutigkeit nur unter login-fähigen Accounts
- `kinder-rsvp`: Funktioniert weiterhin über `family_links` + `member_id`; durch Proxy-Account ist `members.user_id` nun auch für Kinder gesetzt, was die Sichtbarkeitsfilter stabiler macht
- `dienst-anmeldung`: Claim-Flow erweitert um optionalen Familienmitglied-Selektor

## Impact

- **DB:** Migration — `users.can_login`-Spalte, nullable `email`, partieller Unique-Index (SQLite: Tabellen-Rebuild nötig)
- **Backend:** `internal/auth/handler.go` — Login, Reset, Invite, E-Mail-Checks; `internal/duties/handler.go` — Claim-Endpoint prüft Familienmitglied-Berechtigung; `internal/members/handler.go` — neuer Endpoint für Proxy-Account-Erstellung
- **Frontend:** `web/src/components/admin/MemberFamilieTab.tsx` — Proxy-Account anlegen; `web/src/pages/DutyPage.tsx` — „Für wen?"-Dialog; `web/src/pages/ProfilePage.tsx` / `AdminUsersPage.tsx` — Proxy-Accounts kenntlich machen
- **Keine neuen Dependencies**
