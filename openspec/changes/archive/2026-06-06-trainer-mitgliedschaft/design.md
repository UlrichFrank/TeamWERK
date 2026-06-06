## Context

TeamWERK unterscheidet zwischen **Benutzern** (`users`, Login-Accounts) und **Mitgliedern** (`members`, Vereinsmitgliedschaft mit Profil). Trainer-Berechtigung kommt nicht von der Nutzer-Rolle, sondern von der Vereinsfunktion `trainer` in `member_club_functions`. Ein Trainer hat `role=standard` in `users` + Vereinsfunktion `trainer` in `member_club_functions` + einen Eintrag in `team_trainers`.

Externe/Honorartrainer trainieren ein Team, sind aber keine Vereinsmitglieder. Bisher gibt es keinen sauberen Weg, diese Personen mit einem `members`-Profil zu versehen (z. B. für Adresskontakt, Vereinsfunktionen-Anzeige) ohne sie fälschlicherweise als aktive Mitglieder zu zählen.

## Goals / Non-Goals

**Goals:**
- `honorar` als neuen gültigen `members.status`-Wert einführen
- Honorar-Mitglieder von Pflichten, RSVP-Anfragen und Soll-Berechnung ausschließen
- Bestehende Trainer-Zuweisung (`team_trainers`) bleibt unverändert

**Non-Goals:**
- Kein neues Rollen-Modell oder neues Tabellen-Schema
- Keine separaten „Trainer-Profile" — bestehende `members`+`users`-Architektur bleibt
- Kein eigener UI-Bereich nur für Honorar-Mitglieder

## Decisions

### Status-Erweiterung statt neue Tabelle

**Entscheidung:** `members.status` CHECK-Constraint um `'honorar'` erweitern.

**Alternativen erwogen:**
- *Neue Tabelle `external_persons`*: Overhead, dupliziert Profil-Felder, kein Gewinn
- *Neues Boolean-Feld `is_honorary`*: Semantisch schwächer, erfordert trotzdem Filteranpassungen überall

**Begründung:** Alle bestehenden Member-CRUD-Pfade, Formulare und die CSV-Export-Logik können `honorar` als weiteren Status behandeln — genau wie `ausgetreten`. SQLite lässt einen CHECK-Constraint-Drop+Recreate nicht direkt zu; stattdessen wird die Tabelle per `ALTER TABLE … RENAME`, `CREATE TABLE … CHECK(… OR 'honorar')`, `INSERT INTO … SELECT` und `DROP TABLE` migriert (Standard-Pattern bei golang-migrate + SQLite).

### Filter-Semantik: zwei Klassen von Ausschlüssen

`ausgetreten` = ehemalige Mitglieder (abgetreten)  
`honorar` = externe Mitarbeiter (nie vollwertige Mitglieder)

Bestehende Queries filtern auf `status != 'ausgetreten'`. Diese Logik wird zu `status NOT IN ('ausgetreten', 'honorar')` überall dort erweitert, wo „aktive Vereinsmitglieder" gemeint sind. Explizite Kontexte wie Trainer-Listen oder Kader-Ansichten behalten `honorar` sichtbar.

## Risks / Trade-offs

- **Stille Query-Vergessen** → Mitigation: Grep nach `status != 'ausgetreten'` und `status = 'aktiv'` deckt alle Stellen auf; Migrations-Checkliste enthält diesen Schritt
- **SQLite ALTER TABLE Workaround** → Standard-Pattern, gut dokumentiert; golang-migrate führt Transaction-Wrapping durch, Rollback im Fehlerfall ist sauber
- **CSV-Export zählt Honorar-Mitglieder** → Gewollt: Export soll alle Personen enthalten, die im System gepflegt sind; Filter-Option kann später ergänzt werden

## Migration Plan

1. Migration schreiben (`00N_honorar_member_status.up.sql`): Tabelle umbenennen → neu anlegen mit erweitertem CHECK → Daten kopieren → alte Tabelle droppen
2. Down-Migration: analog (ohne `honorar`-Wert; bestehende `honorar`-Rows bekämen `aktiv`)
3. Backend-Queries anpassen (Grep-gestützt)
4. Frontend: Status-Dropdown und Badges erweitern
5. Deploy: `make deploy` führt `migrate up` automatisch aus
