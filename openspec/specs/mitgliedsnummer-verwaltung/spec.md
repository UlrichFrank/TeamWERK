# mitgliedsnummer-verwaltung Specification

## Requirements

### Requirement: Automatische Vergabe der Mitgliedsnummer beim Anlegen
Das System SHALL beim Anlegen eines Mitglieds die `member_number` automatisch vergeben: die höchste vorhandene **numerische** Nummer + 1. Lücken werden nicht wiederverwendet. Eine vom Client im Request mitgeschickte `member_number` MUST beim Anlegen ignoriert werden. Nicht-numerische Bestandswerte werden bei der Maximum-Bestimmung übersprungen.

#### Scenario: Erste Vergabe ohne Bestand
- **WHEN** ein Mitglied angelegt wird und noch keine numerische Mitgliedsnummer existiert
- **THEN** vergibt das System die Nummer `1`

#### Scenario: Folge-Vergabe ist höchste + 1
- **WHEN** ein Mitglied angelegt wird und die höchste vorhandene numerische Nummer `285` ist
- **THEN** vergibt das System die Nummer `286`

#### Scenario: Client-Wert wird ignoriert
- **WHEN** ein `POST /api/members` eine explizite `member_number` (z.B. `"42"`) enthält
- **THEN** ignoriert das System diesen Wert und vergibt stattdessen die nächste freie Nummer (höchste numerische + 1)

#### Scenario: Lücken werden nicht wiederverwendet
- **WHEN** Nummern `1, 2, 4` vergeben sind (Lücke bei `3`) und ein neues Mitglied angelegt wird
- **THEN** vergibt das System `5` (nicht `3`)

### Requirement: Mitgliedsnummer ist read-only mit Admin-Override
Das System SHALL die Mitgliedsnummer nach der Vergabe gegen Änderungen schützen. Über `PUT /api/members/{id}` MUST eine Änderung der `member_number` nur akzeptiert werden, wenn der anfragende Nutzer die System-Rolle `admin` hat. Für Nicht-Admins bleibt die Nummer unverändert, auch wenn das Feld im Request enthalten ist. Im Frontend wird die Nummer für Nicht-Admins nur angezeigt und für Admins editierbar dargestellt.

#### Scenario: Admin korrigiert die Nummer
- **WHEN** ein Admin `PUT /api/members/{id}` mit einer geänderten, freien `member_number` sendet
- **THEN** speichert das System die neue Nummer und antwortet mit HTTP 200

#### Scenario: Nicht-Admin kann die Nummer nicht ändern
- **WHEN** ein Nutzer ohne Rolle `admin` (z.B. Vorstand) `PUT /api/members/{id}` mit abweichender `member_number` sendet
- **THEN** bleibt die bestehende Mitgliedsnummer unverändert und die übrigen erlaubten Felder werden gespeichert

### Requirement: Eindeutigkeit der Mitgliedsnummer
Das System SHALL die Eindeutigkeit der Mitgliedsnummer erzwingen. Setzt ein Admin eine Nummer, die bereits einem anderen Mitglied gehört, MUST die Route mit HTTP 409 und einer verständlichen Fehlermeldung antworten (statt eines generischen Datenbankfehlers).

#### Scenario: Dublette wird abgelehnt
- **WHEN** ein Admin eine `member_number` setzt, die bereits einem anderen Mitglied zugeordnet ist
- **THEN** antwortet das System mit HTTP 409 und nennt die kollidierende Nummer in der Fehlermeldung

#### Scenario: Unveränderte eigene Nummer ist erlaubt
- **WHEN** ein Admin ein Mitglied speichert, ohne dessen Nummer zu ändern
- **THEN** wird kein Konflikt gemeldet und die Anfrage ist erfolgreich (HTTP 200)

### Requirement: Honorar-Mitglieder ohne Nummer
Das System SHALL für Mitglieder mit Status `honorar` keine Mitgliedsnummer führen (bestehendes Verhalten bleibt erhalten). Beim Setzen des Status `honorar` wird die `member_number` geleert. Ein Honorar-Mitglied ohne Nummer ist KEIN Konflikt.

#### Scenario: Status honorar leert die Nummer
- **WHEN** ein Mitglied auf Status `honorar` gesetzt wird
- **THEN** entfernt das System dessen `member_number`

#### Scenario: Honorar ohne Nummer ist kein Konflikt
- **WHEN** die Konflikt-Erkennung über ein Honorar-Mitglied ohne Nummer läuft
- **THEN** wird dieses Mitglied nicht als Konflikt markiert

### Requirement: Konflikt-Erkennung und -Anzeige in der Mitglieder-Übersicht
Das System SHALL Nummern-Konflikte erkennen und in der Mitglieder-Übersicht (`/mitglieder`) sichtbar machen. `GET /api/members` MUST pro Mitglied einen Konflikt-Indikator liefern. Als Konflikt gelten: (a) eine Nummer, die mehrfach vorkommt (Dublette), (b) ein nicht-numerischer `member_number`-Wert, (c) ein Nicht-`honorar`-Mitglied ohne Nummer. Das Frontend markiert betroffene Zeilen mit einem Hinweis (lucide `AlertTriangle`, `brand-*`-Tokens).

#### Scenario: Nicht-honorar-Mitglied ohne Nummer
- **WHEN** ein Mitglied mit Status `aktiv`, `passiv` oder `anwaerter` keine Mitgliedsnummer hat
- **THEN** kennzeichnet das System es als Konflikt vom Typ „fehlende Nummer"

#### Scenario: Nicht-numerischer Wert
- **WHEN** ein Mitglied eine nicht-numerische `member_number` (z.B. `"M-100"`) hat
- **THEN** kennzeichnet das System es als Konflikt vom Typ „nicht numerisch"

#### Scenario: Doppelte Nummer
- **WHEN** zwei Mitglieder dieselbe Mitgliedsnummer tragen
- **THEN** kennzeichnet das System beide als Konflikt vom Typ „Dublette"

#### Scenario: Übersicht zeigt Konflikt-Hinweis
- **WHEN** die Mitglieder-Übersicht Mitglieder mit Konflikt-Indikator lädt
- **THEN** zeigt das Frontend bei diesen Zeilen einen sichtbaren Warnhinweis an
