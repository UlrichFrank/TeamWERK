## 1. Datenbankschema

- [x] 1.1 Migration `014_kader_team_number.up.sql` erstellen: `ALTER TABLE kader ADD COLUMN team_number INTEGER NOT NULL DEFAULT 1`, `ALTER TABLE kader ADD COLUMN dedicated_birth_year INTEGER`, neuer UNIQUE Index `kader_unique ON kader(season_id, age_class, gender, team_number)`
- [x] 1.2 Migration `014_kader_team_number.down.sql` erstellen: Index löschen, neuen Index ohne team_number anlegen
- [x] 1.3 Lokale Migrationen testen: `make migrate-up` und `make migrate-down`

## 2. Backend — Jahrgangskalkulation

- [x] 2.1 `ageBracketRef2025` in `age_brackets.go` korrigieren: A=[2007,2008], B=[2009,2010], C=[2011,2012], D=[2013,2014]
- [x] 2.2 `age_brackets_test.go` aktualisieren: Testwerte auf neue Referenzjahrgänge anpassen
- [x] 2.3 Sicherstellen dass `BirthYearInBracket` für 2025 und 2026 korrekte Ergebnisse liefert (kein Jahrgang in zwei Klassen)

## 3. Backend — Handler erweitern

- [x] 3.1 `kaderRow` und `kaderDetail` Structs um `TeamNumber int` und `DedicatedBirthYear *int` erweitern
- [x] 3.2 `ListKader` und `GetKader`: `team_number`, `dedicated_birth_year` aus DB lesen; `birth_years []int` berechnen und in Response einfügen (beide Bracket-Jahre wenn NULL, nur den einen wenn gesetzt)
- [x] 3.3 `InitializeKader`: Alle 7 Standard-Kader mit `team_number=1` und `dedicated_birth_year=NULL` anlegen (Default bleibt Jahrgangsmischung)
- [x] 3.4 Neuen Einzelkader-Anlegen-Pfad in `InitializeKader` bzw. neuem Handler: `POST /api/admin/kader` mit Body `{season_id, age_class, gender, team_number, dedicated_birth_year}` — antwortet 201 mit neuem Kader-Objekt, 409 bei Duplikat
- [x] 3.5 `UpdateKader` um `dedicated_birth_year` erweitern: wenn im Body vorhanden, UPDATE auf `kader`-Tabelle ausführen
- [x] 3.6 `DELETE /api/admin/kader/{id}` implementieren: prüft ob `kader_members` leer, bei leer → 204, sonst → 409 mit `{"error":"...", "member_count":N}`
- [x] 3.7 Neuen Route in `cmd/teamwerk/main.go` registrieren: `r.Delete("/api/admin/kader/{id}", kaderH.DeleteKader)`

## 4. Backend — Filterlogik

- [x] 4.1 `suggestMembers` in `suggestions.go` anpassen: wenn `dedicated_birth_year` für den Kader gesetzt ist, `WHERE birth_year = ?` statt `BETWEEN ? AND ?` verwenden
- [x] 4.2 `autoAssignMembers` in `copy.go` anpassen: gleiche Logik für `dedicated_birth_year`
- [x] 4.3 `MemberSuggestions`-Handler: `dedicated_birth_year` aus der Kader-Abfrage mitlesen und an `suggestMembers` übergeben

## 5. Frontend — Kachel-Anzeige

- [x] 5.1 `Kader`-Interface in `AdminKaderPage.tsx` um `team_number`, `dedicated_birth_year`, `birth_years` erweitern
- [x] 5.2 Kachel-Titel anpassen: `team_number` nur anzeigen wenn für diese Altersklasse/Geschlecht mehr als ein Kader existiert (z.B. „C-Jugend 1 männlich")
- [x] 5.3 Jahrgänge auf der Kachel anzeigen: `birth_years` als kompakter Badge neben Mitgliederanzahl (z.B. „Jg. 2011" oder „Jg. 2011/12")

## 6. Frontend — Modus-Umschalter

- [x] 6.1 Toggle-Element auf jeder Kachel: „gemischt" / „dediziert" (Radio oder Toggle)
- [x] 6.2 Bei Wechsel auf „dediziert": Dropdown mit den beiden Jahrgängen der Altersklasse (berechnet aus `birth_years` des Brackets)
- [x] 6.3 Bei Auswahl: `PUT /api/admin/kader/{id}` mit `{dedicated_birth_year: <year>}` oder `{dedicated_birth_year: null}` senden, danach `loadAll()` aufrufen
- [x] 6.4 `KaderMemberSearch` erhält `birthYears: number[]` als Prop; nutzt diese zur Information, keine Logikänderung nötig (Filter bleibt serverseitig)

## 7. Frontend — Kader anlegen und löschen

- [x] 7.1 „+ Mannschaft anlegen"-Button pro Altersklasse/Geschlecht-Gruppe anzeigen, wenn weniger als 2 Kader für diese Kombination existieren
- [x] 7.2 Inline-Dialog oder Modal: `team_number` (automatisch nächste Nummer), `dedicated_birth_year` Dropdown (beide Jahrgänge oder „gemischt")
- [x] 7.3 POST an `/api/admin/kader` senden, danach `loadAll()`; bei 409 Fehler-Toast anzeigen
- [x] 7.4 Löschen-Button [×] auf jeder Kachel (nur wenn `member_count === 0`; sonst deaktiviert mit Tooltip „Erst alle Mitglieder entfernen")
- [x] 7.5 Bei Klick auf Löschen: Bestätigungsdialog → `DELETE /api/admin/kader/{id}` → `loadAll()` bei Erfolg; Toast bei Fehler

## 8. Abschluss

- [x] 8.1 Manuelle Tests: Jahrgänge in Vorschlägen prüfen (dediziert vs. gemischt), Anlegen eines zweiten C-Jugend-Kaders, Löschen eines leeren Kaders
- [x] 8.2 Sicherstellen dass Copy-Modal noch funktioniert (keine Regression durch team_number)
