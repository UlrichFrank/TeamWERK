## 1. Backend — Copy-Logik korrigieren

- [ ] 1.1 `ageClassBefore` in `internal/kader/copy.go` umbenennen zu `ageClassBelow` und Richtung umkehren: A-Jugend→B-Jugend, B-Jugend→C-Jugend, C-Jugend→D-Jugend, D-Jugend→""
- [ ] 1.2 `copyMembersFromKader` mit Jahrgangsfilter erweitern: Parameter `bracketMin, bracketMax int` hinzufügen, SQL-WHERE um `AND CAST(strftime('%Y', m.date_of_birth) AS INTEGER) BETWEEN ? AND ?` ergänzen
- [ ] 1.3 `copyKader` auf Smart-Copy umstellen: für `member_source == "smart-copy"` (oder leer/default) beide Quellen kombinieren — gleiche Klasse + `ageClassBelow`-Klasse — beide gefiltert nach Ziel-Bracket; `same-age-previous` und `age-before-previous` durch `smart-copy` ersetzen
- [ ] 1.4 Ziel-Bracket im `copyKader`-Aufruf berechnen: `ComputeAgeBrackets(targetStartYear)[a.AgeClass]` und an beide `copyMembersFromKader`-Aufrufe übergeben

## 2. Backend — Auto-Assign Endpunkt

- [ ] 2.1 Neuen Handler `AutoAssign(w, r)` in `internal/kader/handler.go` anlegen: `POST /api/admin/kader/auto-assign`, Body `{ "kader_ids": [1, 2, 3] }`, ruft für jede ID `autoAssignMembers` auf
- [ ] 2.2 Route in `cmd/teamwerk/main.go` registrieren (admin/vorstand Rolle)

## 3. Frontend — CopyKaderModal vereinfachen

- [ ] 3.1 `auto-assign` Option aus Schritt 3 (Member-Assignment) in `CopyKaderModal.tsx` entfernen
- [ ] 3.2 Schritt 3 entfernen: da nur noch `smart-copy` und `empty` existieren, reicht Schritt 2 (Kader-Auswahl) — alle ausgewählten Kader bekommen `member_source: "smart-copy"` als Default; Option "Nur Struktur" pro Kader als Toggle/Checkbox beibehalten
- [ ] 3.3 `memberSourceLabel`-Funktion und `ageBefore`-Hilfsfunktion entfernen oder vereinfachen

## 4. Frontend — AutoAssignModal neu anlegen

- [ ] 4.1 `web/src/components/AutoAssignModal.tsx` erstellen: lädt alle Kader der aktiven Saison, zeigt Checkboxen mit `{age_class} {gender} (Jg. {yr1}/{yr2})`, alle standardmäßig ausgewählt
- [ ] 4.2 Bestätigungs-Button ruft `POST /api/admin/kader/auto-assign` mit den IDs der ausgewählten Kader auf
- [ ] 4.3 Nach Erfolg: Modal schließen, Kader-Seite neu laden, Toast „Auto-Assign abgeschlossen"

## 5. Frontend — AdminKaderPage anpassen

- [ ] 5.1 „Auto-Assign"-Button neben „Aus vorheriger Saison kopieren" im Seitenkopf hinzufügen
- [ ] 5.2 `AutoAssignModal` einbinden: State `showAutoAssignModal`, Props `activeSeason`, `onDone`, `onClose`
