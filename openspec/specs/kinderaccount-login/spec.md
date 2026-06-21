# kinderaccount-login Specification

## Purpose
TBD - created by archiving change kinderaccount-ohne-email. Update Purpose after archive.
## Requirements
### Requirement: Login per Spielername als Alternative zur E-Mail
Das System SHALL einem Nutzer erlauben, sich entweder mit seiner E-Mail-Adresse oder mit seinem `login_name` (Format `Vorname.Nachname`) zu authentifizieren. Die Anmeldung MUSS case-insensitiv gegen `email` und `login_name` prüfen und nur Accounts mit `can_login=1` zulassen.

#### Scenario: Kind loggt sich mit Spielername ein
- **WHEN** ein aktiviertes Kinder-Konto (`email IS NULL`, `login_name='Lena.Schmidt'`, `can_login=1`) mit `"Lena.Schmidt"` und korrektem Passwort anmeldet
- **THEN** liefert die Login-Route Access- + Refresh-Token (HTTP 200)

#### Scenario: Spielername wird case-insensitiv erkannt
- **WHEN** dasselbe Konto sich mit `"lena.schmidt"` (Kleinschreibung) und korrektem Passwort anmeldet
- **THEN** ist der Login erfolgreich (HTTP 200)

#### Scenario: E-Mail-Login bleibt unverändert möglich
- **WHEN** ein Konto mit gesetzter E-Mail und `can_login=1` sich per E-Mail + Passwort anmeldet
- **THEN** ist der Login erfolgreich (HTTP 200)

#### Scenario: Login per Spielername bei deaktiviertem Konto schlägt fehl
- **WHEN** ein Kinder-Konto mit `can_login=0` (Passwort noch nicht gesetzt) sich mit seinem `login_name` anzumelden versucht
- **THEN** lehnt das System mit HTTP 401 ab

#### Scenario: Falsches Passwort bei Spielername-Login
- **WHEN** ein aktiviertes Kinder-Konto mit korrektem `login_name`, aber falschem Passwort anmeldet
- **THEN** lehnt das System mit HTTP 401 ab

### Requirement: Eindeutiger, normalisierter Spielername
Das System SHALL beim Anlegen eines Kinder-Kontos einen `login_name` aus `Vorname.Nachname` erzeugen, der über alle Konten hinweg eindeutig ist. Umlaute/ß MÜSSEN transliteriert (`ä→ae`, `ö→oe`, `ü→ue`, `ß→ss`), Leerzeichen innerhalb eines Namensteils durch Bindestriche ersetzt und nicht erlaubte Zeichen entfernt werden; Vergleich und Eindeutigkeit gelten case-insensitiv.

#### Scenario: Umlaute werden transliteriert
- **WHEN** ein Kinder-Konto für „Lena Müller" erzeugt wird
- **THEN** ist der `login_name` `"Lena.Mueller"`

#### Scenario: Doppelname wird mit Bindestrich verbunden
- **WHEN** ein Kinder-Konto für „Anna Lena Schmidt" mit Vorname „Anna Lena" erzeugt wird
- **THEN** ist der `login_name` `"Anna-Lena.Schmidt"`

#### Scenario: Kollision erhält ein numerisches Suffix
- **WHEN** ein Kinder-Konto für „Lena Schmidt" erzeugt wird und `Lena.Schmidt` bereits existiert (auch wenn dieses Konto noch `can_login=0` hat)
- **THEN** wird der `login_name` `"Lena.Schmidt2"` vergeben; bei weiterer Kollision `"Lena.Schmidt3"` usw., bis ein freier Name gefunden ist

### Requirement: Passwort setzen aktiviert das Kinder-Konto
Das System SHALL über einen zeitlich begrenzten Token-Link (48 h) das Setzen eines Passworts für ein Kinder-Konto erlauben. Beim erfolgreichen Setzen MUSS `can_login` auf 1 gesetzt und das Token invalidiert werden.

#### Scenario: Eltern setzen Passwort über gültigen Token
- **WHEN** über einen gültigen, nicht abgelaufenen Token ein Passwort gesetzt wird
- **THEN** speichert das System den bcrypt-Hash, setzt `can_login=1` und markiert das Token als verbraucht (HTTP 200)

#### Scenario: Abgelaufener oder verbrauchter Token
- **WHEN** ein Passwort über einen abgelaufenen oder bereits verbrauchten Token gesetzt werden soll
- **THEN** lehnt das System mit HTTP 400 ab und das Konto bleibt bei `can_login=0`

