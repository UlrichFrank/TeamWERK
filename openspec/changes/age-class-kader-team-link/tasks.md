## 1. Datenbank-Migration

- [ ] 1.1 Migration `011_age_class_canonical.up.sql` anlegen: `age_class_game_rules` mit Langform-PKs neu erstellen (`'A-Jugend'`–`'D-Jugend'`), Standardwerte einfügen
- [ ] 1.2 In derselben Migration `teams`-Tabelle per SQLite-Rewrite-Pattern mit FK-Constraint auf `age_class_game_rules.age_class` neu erstellen, bestehende Daten kopieren
- [ ] 1.3 Down-Migration `011_age_class_canonical.down.sql` anlegen: Kurzform-Keys wiederherstellen, FK entfernen

## 2. Backend

- [ ] 2.1 `internal/config/handler.go`: `validAgeClasses`-Map auf Langform-Keys ('A-Jugend' usw.) umstellen
- [ ] 2.2 `internal/config/handler.go`: `CreateTeam` und `UpdateTeam` validieren `age_class` gegen `age_class_game_rules` (DB-Query statt Hardcode) und antworten HTTP 422 bei unbekanntem Wert
- [ ] 2.3 `internal/games/handler.go`: Workaround-Code `[:1]`-Extraktion in `effectiveEventDuration` entfernen

## 3. Frontend

- [ ] 3.1 `AdminAgeClassRulesPage.tsx`: Klassen-Label von `{rule.age_class}-Jugend` auf `{rule.age_class}` ändern (Suffix entfernt, da Key jetzt Langform ist)
- [ ] 3.2 Teams-Admin-UI (in `AdminDutyTypesPage.tsx` oder separater Seite): `age_class`-Freitext-Input durch `<select>` ersetzen, das die Optionen aus `GET /api/admin/age-class-rules` lädt (inkl. Leer-Option für NULL)

## 4. Verifikation

- [ ] 4.1 Lokal: `make migrate-up` ausführen, prüfen ob Migration fehlerfrei durchläuft
- [ ] 4.2 Spieltag-Detailseite: Dienste neu generieren für ein B-Jugend-Heimspiel — kein Fehler mehr, korrekte Slot-Zeiten
- [ ] 4.3 Teams-Admin: Team mit 'B-Jugend' aus Dropdown anlegen und bearbeiten — Auswahl wird korrekt gespeichert und angezeigt
- [ ] 4.4 Altersklassen-Regeln-Seite: Labels zeigen 'A-Jugend' statt 'A-Jugend-Jugend'
