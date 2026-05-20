## 1. Backend: Export erweitern

- [x] 1.1 `Export`-Handler in `internal/members/handler.go` auf 12 Felder erweitern: SQL-Query um `member_number`, `jersey_number`, `position`, `gender`, `user_id` ergänzen
- [x] 1.2 Zweiten LEFT JOIN auf `users` für `Benutzer_Email` via `members.user_id` hinzufügen
- [x] 1.3 Subquery oder zweiten JOIN für `family_links` → bis zu 2 Erziehungsberechtigten-Emails (geordnet nach `parent_user_id`)
- [x] 1.4 CSV-Header auf die 12 definierten Spaltennamen anpassen; Semikolon als Trennzeichen setzen (`cw.Comma = ';'`)
- [x] 1.5 Filter `WHERE m.status != 'ausgetreten'` entfernen — alle Mitglieder exportieren

## 2. Backend: Import-Handler

- [x] 2.1 Neue Funktion `Import` in `internal/members/handler.go` anlegen: multipart/form-data parsen (`r.FormFile("file")`), Parameter `mode` lesen (`append` oder `update`)
- [x] 2.2 CSV-Parser: Header-Zeile einlesen und validieren (Pflichtfelder `Vorname`, `Nachname`); BOM strippen; Trennzeichen auto-detect (Semikolon vs. Komma)
- [x] 2.3 Duplikat-Erkennung innerhalb der CSV: Map `lower(vorname)+lower(nachname)+dob` → Zeilennummer; bei Kollision beide Zeilen als Fehler markieren
- [x] 2.4 Pro Zeile: DB-Lookup nach Idempotenz-Schlüssel (`lower(first_name)=? AND lower(last_name)=?`, optional `AND date_of_birth=?`)
- [x] 2.5 Modus `append`: neue Mitglieder anlegen (`INSERT INTO members ...`), bestehende als `unchanged` markieren
- [x] 2.6 Modus `update`: neue Mitglieder anlegen; für bestehende Mitglieder feldweise vergleichen und nur nicht-leere geänderte Felder updaten (`UPDATE members SET ... WHERE id=?`)
- [x] 2.7 User-Link: bei nicht-leerem `Benutzer_Email`-Feld → Email in `users` suchen, wenn gefunden und `members.user_id` noch nicht gesetzt: updaten; wenn nicht gefunden: Hinweis in Bericht
- [x] 2.8 Erziehungsberechtigte: für jede nicht-leere Parent-Email → Email in `users` suchen, wenn gefunden und `family_links` noch nicht vorhanden: `INSERT INTO family_links (parent_user_id, member_id)` einfügen
- [x] 2.9 JSON-Importbericht zusammenbauen und als HTTP 200 zurückgeben

## 3. Backend: Route registrieren

- [x] 3.1 In `cmd/teamwerk/main.go` die Route `POST /api/members/import` unter der Admin-Only-Gruppe registrieren und `membH.Import` zuweisen

## 4. Frontend: Import-Modal

- [x] 4.1 In `web/src/pages/MembersPage.tsx` einen "Import CSV"-Button neben dem bestehenden "Export CSV"-Button einfügen (nur für Admins sichtbar)
- [x] 4.2 Import-Modal-State anlegen: `showImport`, `importFile`, `importMode` (`append`|`update`), `importing`, `importResult`
- [x] 4.3 Modal mit File-Input (`.csv`), Modus-Toggle ("Nur ergänzen" / "Fehlende + geänderte Felder aktualisieren") und "Import starten"-Button implementieren
- [x] 4.4 API-Call: `api.post('/members/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } })` mit `FormData` (Felder `file` und `mode`)
- [x] 4.5 Importbericht-Anzeige im Modal: Zusammenfassung (Zählwerte) + Detailliste je Zeile (created/updated/unchanged/error mit Änderungsdetails)
- [x] 4.6 Modal schließen und Mitgliederliste neu laden nach erfolgreichem Import

## 5. Manuelle Verifikation

- [x] 5.1 Export testen: CSV herunterladen, alle 12 Spalten prüfen, Erziehungsberechtigte prüfen, `ausgetreten`-Mitglieder prüfen
- [x] 5.2 Import testen (Modus `append`): CSV mit 1 neuem Mitglied hochladen → Bericht zeigt 1 created
- [x] 5.3 Import testen (Modus `update`): CSV mit geänderter Passnummer hochladen → Bericht zeigt 1 updated mit Felddetail
- [x] 5.4 Non-Destructive prüfen: leere Felder in CSV → bestehende DB-Werte unverändert
- [x] 5.5 Duplikat-in-CSV prüfen: zwei gleiche Zeilen → beide als error im Bericht
- [x] 5.6 Email-nicht-gefunden prüfen: unbekannte Benutzer-Email → Hinweis im Bericht, kein Fehler
