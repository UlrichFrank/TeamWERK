## 1. Datenbank-Migration

- [x] 1.1 Migration `018_honorar_member_status.up.sql` schreiben: `members`-Tabelle umbenennen → neue Tabelle mit erweitertem CHECK (`'aktiv','verletzt','pausiert','ausgetreten','honorar'`) anlegen → Daten kopieren → alte Tabelle droppen
- [x] 1.2 Migration `018_honorar_member_status.down.sql` schreiben: Rückwärts-Migration (ohne `honorar`; bestehende `honorar`-Rows werden auf `aktiv` gesetzt)
- [x] 1.3 Migration lokal mit `make migrate-up` testen und verifizieren

## 2. Backend — Query-Audit

- [x] 2.1 Grep nach `status != 'ausgetreten'`, `status = 'aktiv'`, `status <> 'ausgetreten'` in `internal/` — Liste aller betroffenen Queries erstellen
- [x] 2.2 Alle Queries für „aktive Vereinsmitglieder" (Mitgliederliste, Dienst-Soll, RSVP-Einladungen) auf `status NOT IN ('ausgetreten', 'honorar')` umstellen
- [x] 2.3 Queries in `internal/members/` für Trainer-/Kader-Ansichten prüfen: `honorar`-Mitglieder dort explizit einschließen

## 3. Backend — Validierung & API

- [x] 3.1 Status-Whitelist in `internal/members/` um `honorar` erweitern (falls hardkodiert)
- [x] 3.2 `GET /api/members` sicherstellen: Honorar-Mitglieder für Admin/Trainer sichtbar, kein ungewolltes Ausblenden
- [x] 3.3 Duty-Account-Query in `internal/duties/` prüfen: `honorar`-Mitglieder dürfen keinen Soll-Eintrag erhalten
- [x] 3.4 Trainer-Zugriff (`member_club_functions`-Prüfung) sicherstellen: Honorar-Trainer mit Vereinsfunktion `trainer` und `role=standard` haben identische Berechtigungen wie reguläre Trainer-Mitglieder

## 4. Frontend — Status-Erweiterung

- [x] 4.1 Status-Dropdown in Mitglied-Anlegen/Bearbeiten-Formular um Option `Honorar` erweitern
- [x] 4.2 Status-Filter in Mitgliederliste (`MembersPage`) um `honorar` ergänzen
- [x] 4.3 Status-Badge für `honorar` mit passendem Label und Farbe anlegen (analog zu bestehenden Status-Badges)
- [x] 4.4 Trainer-Zuweisungsansicht (falls vorhanden): Honorar-Mitglieder klar sichtbar und als „Honorar" markiert anzeigen
