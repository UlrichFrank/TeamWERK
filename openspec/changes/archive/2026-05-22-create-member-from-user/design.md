## Context

Die `users`-Tabelle und die `members`-Tabelle sind über `members.user_id` (nullable FK) lose gekoppelt. Nach einer Einladungsregistrierung existiert ein User-Account, aber kein Member-Datensatz. Admins legen Mitglieder aktuell separat an (`/admin/mitglieder → Neu`) und verknüpfen sie danach manuell über `PUT /api/admin/members/{id}/user`. Dieser Zweischrittfluss ist fehleranfällig und aufwändig.

## Goals / Non-Goals

**Goals:**
- Mitglied mit einem Klick aus einem bestehenden User-Account erstellen
- Name des Accounts als Vorname/Nachname übernehmen (Best-Effort-Split)
- `members.user_id` sofort setzen — keine manuelle Verknüpfung nötig
- Button nur anzeigen, wenn noch kein Mitglied verknüpft ist

**Non-Goals:**
- Vollständiges Erfassungsformular im Dialog (Geburtsdatum, Passnummer etc. — das erfolgt danach in der Mitgliederverwaltung)
- Automatische Kader-Zuweisung
- Anlegen für Trainer- oder Vorstand-Rollen (nur Admin)

## Decisions

### 1. Minimal-Insert ohne Pflichtfelder-Validierung

`members` hat `date_of_birth` und `pass_number` UNIQUE, aber beide sind nullable in der DB. Das Mitglied wird mit `status='aktiv'` und leerem `pass_number` (NULL) angelegt. Das reicht für eine sofort nutzbare Verknüpfung; Admin füllt Rest über normale Mitgliedsbearbeitung.

**Alternative verworfen:** Inline-Formular im Dialog mit Pflichtfeldern → erhöht Reibung, wäre schnell redundant zur bestehenden Mitgliedsseite.

### 2. Neuer Endpoint `POST /api/admin/users/{id}/create-member`

Eigener Endpoint statt Erweiterung von `POST /api/members`, weil der User-ID-Kontext hier die treibende Eingabe ist und die Autorisierung direkt daran hängt. Gibt `{ member_id }` zurück.

### 3. Name-Split per Leerzeichen (erstes Wort = Vorname, Rest = Nachname)

`users.name` ist ein Freitext-Feld. Split bei erstem Leerzeichen ist pragmatisch und deckt den Normalfall ab. Enthält der Name kein Leerzeichen, wird alles als `first_name` gesetzt und `last_name` bleibt leer.

### 4. Kein Modal — direkter POST mit Bestätigungs-Toast

Da keine Pflichtfelder abgefragt werden, reicht ein direkter Button-Klick ohne Dialog. Erfolg: Toast + Button verschwindet. Fehler: Fehlermeldung inline.

## Risks / Trade-offs

- **Doppelte Mitglieder**: Wenn ein Admin den Button versehentlich klickt, obwohl das Mitglied bereits manuell existiert und nur nicht verknüpft ist → Mitigation: Button nur rendern wenn `member_id == null` in der User-Liste; Endpoint prüft ebenfalls ob User bereits ein Mitglied hat.
- **Unvollständige Daten**: Neu angelegte Mitglieder haben kein Geburtsdatum/Passnummer → kein funktionales Problem, aber Admin muss nacharbeiten. Akzeptabel, da expliziter Folgeschritt.
- **Name-Split-Qualität**: Arabische/asiatische Namen oder Einzelnamen erzeugen ggf. seltsame Aufteilungen → Admin korrigiert im Anschluss; kein Blocking-Problem.
