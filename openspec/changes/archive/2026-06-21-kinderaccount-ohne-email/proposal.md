## Why

Kinder haben oft keine eigene E-Mail-Adresse, brauchen aber einen eigenen Login (für Profil, Dienstbörse, RSVP, Chat). Heute ist die E-Mail der einzige Login-Schlüssel (`users.email`, Unique-Index nur bei `can_login=1`) — ohne E-Mail kein login-fähiger Account. Wir wollen Kinder-Accounts ohne E-Mail anlegen können und stattdessen den Namen des Kindes (`Vorname.Nachname`) als eindeutigen Login-Schlüssel nutzen. Die verwaltenden Eltern erhalten den Schriftverkehr per Mail.

## What Changes

- **Zweiter Login-Schlüssel**: Neue Spalte `users.login_name` (nullable, case-insensitiv eindeutig solange `can_login=1`). Die Login-Query akzeptiert künftig **E-Mail _oder_ `login_name`**: `WHERE (LOWER(email)=? OR LOWER(login_name)=?) AND can_login=1`.
- **Beitrittsantrag-Variante „Kinderaccount"**: Das öffentliche Antragsformular bekommt eine Auswahl „Kinderaccount anlegen". Aktiviert man sie, werden statt der eigenen E-Mail die **verwaltende Eltern-E-Mail** und der **Kindname (Vorname/Nachname)** erfasst. Kein Geschlechtsfeld im Antrag.
- **`membership_requests`** bekommt die Felder `is_child` (Flag) und `parent_email`.
- **Approve-Flow für Kinderanträge**: Beim Akzeptieren wird (1) ein eindeutiger `login_name` aus `Vorname.Nachname` generiert (Kollision → Suffix `.2`, `.3` … bis frei), (2) ein `users`-Datensatz mit `email=NULL`, gesetztem `login_name` und `can_login=0` angelegt, (3) ein `members`-Datensatz mit dem echten Kindnamen erzeugt und über `user_id` verknüpft, (4) eine Mail an die **`parent_email`** mit dem zugewiesenen Spielernamen und einem Passwort-Setz-Link (Token, 48 h, analog bestehendem invitation/reset-Flow) versandt.
- **Passwort setzen aktiviert den Account**: Sobald die Eltern über den Token-Link ein Passwort setzen, wird `can_login=1`. Danach loggt sich das Kind mit `Vorname.Nachname` + Passwort ein.
- **Normalisierung** des `login_name`: case-insensitiver Vergleich (`LOWER`), Leerzeichen → Bindestrich, Umlaute transliteriert (`Müller`→`Mueller`), Format strikt `Vorname.Nachname`.
- Die Eltern-E-Mail ist **reine Korrespondenz** — es wird **kein** automatischer `family_link` angelegt. Eltern können unabhängig ein eigenes Konto besitzen/verknüpfen lassen.

## Capabilities

### New Capabilities
- `kinderaccount-login`: Login per `users.login_name` (`Vorname.Nachname`) als Alternative zur E-Mail; Generierung, Eindeutigkeit/Normalisierung und Account-Aktivierung über Passwort-Setz-Token.
- `kinderaccount-antrag`: Beitrittsantrag-Variante „Kinderaccount" (Erfassung Kindname + Eltern-E-Mail) und der zugehörige Approve-Flow (Account + Member + Eltern-Mail).

### Modified Capabilities
<!-- Keine bestehende Spec-Capability ändert ihre Requirements auf Spec-Ebene; die Login-Erweiterung wird als neue Capability kinderaccount-login geführt. -->

## Impact

- **Migration** (neue Nummer in `internal/db/migrations/`): `users.login_name` + partieller Unique-Index; `membership_requests.is_child` + `parent_email`.
- **Backend** `internal/auth/handler.go`: `Login` (Query um `login_name` erweitern), `RequestMembership` (Kinder-Variante + Validierung), `ApproveMembershipRequest` (Namensgenerierung, User+Member-Anlage, Eltern-Mail); neue/erweiterte Set-Password-Route. Neuer Helper für `login_name`-Normalisierung & Eindeutigkeit.
- **Frontend** `web/src/pages/` (öffentliches Antragsformular) + ggf. Login-Seite (Label „E-Mail oder Spielername").
- **SSE**: Mutationen (Approve) rufen weiterhin `h.hub.Broadcast(...)`.
- **Tests**: Happy-Path + Fehlerfälle für Login per `login_name`, Antrag-Kindervariante, Approve mit Kollisions-Suffix, Set-Password-Aktivierung.
- Keine neuen externen Dienste, kein zusätzlicher RAM-Footprint.
