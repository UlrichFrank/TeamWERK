## Context

Die Profilseite zeigt Kontodaten (E-Mail, Rolle) heute nur lesend an. Name, E-Mail und Passwort liegen alle in der `users`-Tabelle und werden bisher ausschließlich durch Admins oder den Reset-Flow verändert. Das Sportprofil (`members`-Tabelle) bleibt Admin-verwaltets und ist nicht Gegenstand dieser Änderung.

## Goals / Non-Goals

**Goals:**
- Anzeigenamen sofort änderbar (einfaches PUT)
- Passwort mit Verifikation des alten Passworts änderbar; alle Refresh-Tokens danach invalidiert
- E-Mail mit Bestätigungslink an neue Adresse änderbar; altes Passwort als Verifikation; alle Refresh-Tokens nach Bestätigung invalidiert, Redirect zu /login
- Neue Tabelle `email_change_tokens` speichert ausstehende Änderungen (TTL 24h)

**Non-Goals:**
- Änderung von Rolle oder Team (Admin-only)
- Sportprofil-Felder (first_name, last_name etc.) — bleibt Admin-verwaltets
- Mehrfach-E-Mails oder E-Mail-Aliasse

## Decisions

### 1. Handler in `internal/auth/` statt neuem Package

**Entscheidung:** Die drei neuen Handler (`UpdateAccount`, `ChangePassword`, `RequestEmailChange`, `ConfirmEmailChange`) kommen in `internal/auth/handler.go`, da sie auf `users`- und Token-Tabellen operieren — exakt die Domäne des Auth-Packages.

**Alternative:** Eigenes `internal/profile/`-Package — abgelehnt, da kein eigenes Datenmodell; würde nur einen dünnen Wrapper um dieselben Tabellen legen.

### 2. `email_change_tokens` als eigene Tabelle (analog `password_reset_tokens`)

**Entscheidung:** Neue Tabelle mit `user_id`, `token` (Hash), `new_email`, `expires_at`, `used_at`. Das Muster ist identisch mit `password_reset_tokens`, benötigt aber das Extra-Feld `new_email`.

**Alternative:** `password_reset_tokens` wiederverwenden + Konvention für E-Mail-Tokens — abgelehnt, da das Feld `new_email` nicht passt und Tabellen-Semantik unklar würde.

### 3. Passwort-Verifikation bei E-Mail-Änderung

**Entscheidung:** `POST /api/profile/email` erfordert `{ new_email, password }` — das aktuelle Passwort muss korrekt sein. Schützt vor unbemerkter E-Mail-Übernahme bei offenem Browser.

### 4. Vollständige Session-Invalidierung nach Passwort- und E-Mail-Änderung

**Entscheidung:** `DELETE FROM refresh_tokens WHERE user_id = ?` nach erfolgreicher Änderung. Der Nutzer wird zu `/login` weitergeleitet.

**Warum:** Access-Token-TTL ist 15 min — vernachlässigbar. Refresh-Tokens haben 7 Tage — müssen invalidiert werden damit gestohlene Tokens nach Passwortänderung wertlos sind. Konsistent mit dem bestehenden Passwort-Reset-Flow.

### 5. E-Mail-Bestätigungslink führt direkt zum Backend

**Entscheidung:** `GET /api/profile/email/confirm?token=xyz` ist ein Backend-Endpunkt, der nach Erfolg zu `/login` redirected (HTTP 302).

**Warum:** Kein clientseitiges Token-Handling nötig; der Link funktioniert auch wenn der Nutzer bereits ausgeloggt ist oder einen anderen Browser nutzt.

## Risks / Trade-offs

**[Offenes Zeitfenster bei E-Mail-Änderung]** Zwischen Anfrage und Bestätigung gilt noch die alte E-Mail. Ein Angreifer mit Zugang zum Konto könnte eine neue E-Mail anfordern.
→ Mitigation: Passwort-Verifikation beim Anfordern; Token läuft nach 24h ab; pro User nur ein ausstehender Token (vorherige werden überschrieben).

**[Name-Änderung ohne Verifikation]** Der Anzeigename kann ohne Passwort-Check geändert werden.
→ Bewusste Entscheidung: Der Name ist kein sicherheitskritisches Feld; kein Login-Identifier.

**[JWT-Email-Claim nach E-Mail-Änderung]** Laufende Access-Tokens (max. 15 min) enthalten noch die alte E-Mail im Claim.
→ Akzeptiert: Das Zeitfenster ist kurz; nach Session-Invalidierung und Re-Login ist der Claim aktuell.

## Migration Plan

1. Migration `008_email_change_tokens.up.sql`: Tabelle `email_change_tokens` anlegen
2. Migration `008_email_change_tokens.down.sql`: Tabelle droppen
3. Kein Datenmigrations-Backfill nötig
4. Deploy via `make deploy` (Migrations embedded in Binary)
