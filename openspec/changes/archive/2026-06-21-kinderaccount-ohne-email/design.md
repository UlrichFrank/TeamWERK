## Context

Login in TeamWERK läuft heute ausschließlich über die E-Mail: `internal/auth/handler.go` führt `SELECT id, password, role FROM users WHERE LOWER(email)=LOWER(?) AND can_login=1` aus. Der Unique-Index `users_email_login_unique ON users(email) WHERE can_login=1 AND email IS NOT NULL` erzwingt, dass jeder login-fähige Account eine eindeutige E-Mail hat. Kinder ohne eigene E-Mail können damit keinen eigenen login-fähigen Account bekommen.

`members` und `users` sind getrennte Entitäten (`members.user_id` nullable). Der Beitrittsantrag (`membership_requests`) erfasst heute `first_name`, `last_name`, `email`, `comment`; beim Akzeptieren wird ein `invitation_token` erzeugt und ein Registrierungslink per Mail versandt. Proxy-Accounts (`can_login=0`, leeres Passwort) existieren bereits als Muster für „noch nicht aktivierte" Nutzer.

## Goals / Non-Goals

**Goals:**
- Login-fähige Kinder-Accounts ohne E-Mail, identifiziert über einen eindeutigen `login_name` = `Vorname.Nachname`.
- Beitrittsantrag-Variante „Kinderaccount" mit Erfassung von Kindname + verwaltender Eltern-E-Mail.
- Approve erzeugt User + Member und versendet einen Passwort-Setz-Link an die Eltern; Passwort-Setzen aktiviert den Account.
- Bestehender E-Mail-Login bleibt unverändert funktionsfähig.

**Non-Goals:**
- Kein automatischer `family_link` zwischen Eltern-E-Mail und Kind (reine Korrespondenz). Eltern-Verknüpfung bleibt der bestehenden Familien-Verwaltung überlassen.
- Kein Geschlechtsfeld im Antrag; `members.gender` pflegt der Vorstand bei Bedarf nach.
- Keine Selbstbedienungs-Registrierung für Kinder ohne Vorstand-Approve.
- Kein Umstellen vorhandener E-Mail-Accounts auf `login_name`.

## Decisions

### D1: Zweite Login-Spalte `users.login_name` statt E-Mail-Missbrauch
Neue nullable Spalte `users.login_name`. Login-Query wird zu `WHERE (LOWER(email)=? OR LOWER(login_name)=?) AND can_login=1`. Beide Parameter erhalten denselben (lowercased) Eingabewert, sodass ein Nutzer im selben Feld E-Mail _oder_ Spielername eingeben kann.

*Alternative verworfen:* Den Spielernamen in die `email`-Spalte schreiben — würde E-Mail-Semantik (Versand, Validierung, Unique-Logik) verschmutzen und ist fehleranfällig.

### D2: Eindeutigkeit über partiellen Unique-Index + Generierungs-Helper
Neuer Index analog zur E-Mail: `CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name)) WHERE can_login=1 AND login_name IS NOT NULL`. Da der Account beim Anlegen `can_login=0` hat, greift die DB-Uneindeutigkeit erst bei Aktivierung; deshalb prüft der Generierungs-Helper die Eindeutigkeit **zusätzlich im Code** gegen _alle_ vorhandenen `login_name` (unabhängig von `can_login`), um Doppelvergaben zwischen zwei noch inaktiven Kinder-Accounts zu verhindern.

*Hinweis:* SQLite unterstützt Ausdrucks-Indizes (`LOWER(login_name)`). Damit ist der Vergleich case-insensitiv ohne zusätzliche Spalte.

### D3: Normalisierung des `login_name`
Helper `normalizeLoginName(first, last string) string`:
1. Trim, Mehrfach-Leerzeichen kollabieren.
2. Umlaute/ß transliterieren (`ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`, plus Großbuchstaben-Varianten) und Akzente entfernen.
3. Verbleibende Leerzeichen innerhalb eines Namensteils → Bindestrich (`Anna Lena` → `Anna-Lena`).
4. Auf erlaubte Zeichen `[A-Za-z0-9-]` reduzieren.
5. Zusammensetzen als `Vorname.Nachname` (genau ein Punkt als Trenner).

Gespeichert wird die so erzeugte Schreibweise; der **Vergleich** erfolgt case-insensitiv (`LOWER`). Kein eigenes Sprach-Paket nötig — eine kleine `strings.NewReplacer`-Tabelle reicht (kein RAM-Risiko).

### D4: Kollisions-Strategie „Suffix bis frei"
Ist `Lena.Schmidt` belegt, wird `Lena.Schmidt2`, `Lena.Schmidt3` … geprüft, bis ein freier Name gefunden ist (Schleife mit Obergrenze als Sicherheitsnetz, z. B. 1000 Versuche → danach Fehler). Das Suffix hängt an den Nachnamen-Teil, der Punkt-Trenner bleibt erhalten.

*Alternative verworfen:* Profi-Namens-Katalog — vom Nutzer explizit verworfen; der echte Kindname ist erkennbarer und kollisionsärmer durch unbegrenzten Suffix-Pool.

### D5: Approve-Flow in einer Transaktion
`ApproveMembershipRequest` unterscheidet `is_child`:
- **Standard (unverändert):** wie bisher invitation_token + Registrierungslink.
- **Kind:** in einer DB-Transaktion (a) `login_name` generieren, (b) `INSERT INTO users (login_name, email, password, role, can_login) VALUES (?, NULL, '', 'standard', 0)`, `LastInsertId()`, (c) `INSERT INTO members (first_name, last_name, user_id, …)`, (d) Passwort-Setz-Token (wie `password_reset_tokens`/`invitation_tokens`, 48 h) anlegen. Nach erfolgreichem Commit: Mail an `parent_email` mit Spielername + `/set-password?token=…`. Mailversand außerhalb der Transaktion; `h.hub.Broadcast("membership-event")` nach Erfolg.

### D6: Set-Password aktiviert den Account
Die Passwort-Setz-Route (bestehenden Reset-Flow wiederverwenden oder dünn erweitern) setzt `password=<bcrypt>` und `can_login=1` für den zugehörigen User und invalidiert das Token. Erst danach greift der Unique-Index und der Login per `login_name` ist möglich.

## Risks / Trade-offs

- **Doppelvergabe zwischen zwei inaktiven Accounts (can_login=0)** → Mitigation: Code-seitige Eindeutigkeitsprüfung gegen alle `login_name` (D2), nicht nur DB-Index.
- **Login-Feld-Mehrdeutigkeit** (Nutzer gibt etwas ein, das sowohl wie E-Mail als auch Name aussieht) → Mitigation: `OR`-Query mit identischem lowercased Wert; ein Spielername enthält kein `@`, eine E-Mail enthält keine zwei Namensteile mit Punkt-Konvention — praktische Kollision unwahrscheinlich.
- **Transliteration verfälscht Namen** (`Søren`, kyrillisch) → Mitigation: Restzeichen werden entfernt; falls der erzeugte `login_name` leer/zu kurz ist, Fehler an den Vorstand mit Bitte um manuelle Vergabe (Open Question).
- **Suffix-Endlosschleife** bei massenhaft gleichen Namen → Mitigation: harte Obergrenze (z. B. 1000) → Fehler statt Hänger.
- **Eltern-Mail erreicht falsche Person** (Tippfehler im Antrag) → Mitigation: Eltern-E-Mail wird im Antrag erfasst und ist beim Approve für den Vorstand sichtbar/prüfbar.

## Migration Plan

1. Neue Migration `00N_kinderaccount_login.up.sql`/`.down.sql` (nächste freie Nummer):
   - `ALTER TABLE users ADD COLUMN login_name TEXT;`
   - `CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name)) WHERE can_login=1 AND login_name IS NOT NULL;`
   - `ALTER TABLE membership_requests ADD COLUMN is_child INTEGER NOT NULL DEFAULT 0;`
   - `ALTER TABLE membership_requests ADD COLUMN parent_email TEXT;`
   - `.down.sql`: Index droppen; Spalten via Tabellen-Rebuild entfernen (SQLite kann `DROP COLUMN` ab 3.35 — sonst Rebuild).
2. Deploy via `make deploy` (führt `migrate up` aus). Rollback: `make migrate-down` (eine Stufe).
3. Backwards-kompatibel: bestehende E-Mail-Accounts unverändert; `login_name` bleibt NULL.

## Open Questions

- Wenn die Transliteration einen leeren/unbrauchbaren `login_name` ergibt (exotische Zeichen): harter Fehler an den Vorstand oder Fallback auf einen generischen Schlüssel? **Vorschlag:** Fehler mit Hinweis, Vorstand vergibt manuell.
- Soll die Login-Seite das Feld-Label sichtbar zu „E-Mail oder Spielername" ändern, oder bleibt „E-Mail" mit Tooltip? **Vorschlag:** Label anpassen.
