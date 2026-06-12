## 1. Backend — POST /api/absences

- [x] 1.1 Request-Struct um `MemberIDs []int` erweitern; Normalisierung: `member_ids` hat Vorrang, sonst `[member_id]`, sonst Spieler-Fallback
- [x] 1.2 Phase-1-Loop: für jedes `mid` `resolveMemberID(ctx, claims, mid)` → bei Fehler sofort abbrechen (403/400 wie heute, Body identifiziert den Fehler-Member)
- [x] 1.3 Phase-1-Konflikt-Check: pro Member-ID `SELECT COUNT(*) FROM member_absences WHERE member_id=? AND type=? AND start_date<=? AND end_date>=?`; sammle alle Konflikte
- [x] 1.4 Bei einem oder mehr Konflikten: 409 mit `{"error":"overlap","conflicts":[{"member_id":…,"member_name":"…"}]}`. Für member_name: `SELECT first_name||' '||last_name FROM members WHERE id=?`. Kein Insert wird angestoßen
- [x] 1.5 Phase-2-Transaktion: für jeden resolvedMember Insert `member_absences` + bestehendes Auto-Decline-Block (Trainings + Spiele) — alles im selben `tx`
- [x] 1.6 Erfolg: 201 mit Body `{"absence_ids":[…]}` für Multi-Member-Aufruf; weiterhin `{"id":…}` für Legacy-`member_id`-Aufruf (Backwards-Compat)
- [x] 1.7 Hub-Broadcast (`h.hub.Broadcast("absences")`) nach Commit, einmal — nicht pro Member

## 2. Backend — GET /api/absences/preview

- [x] 2.1 Query-Param `member_ids` (CSV) parsen, in `[]int` umwandeln; bei fehlendem Param Fallback auf `member_id` (single)
- [x] 2.2 Berechtigungs-Check pro Member-ID (gleicher Pfad wie POST)
- [x] 2.3 Bestehende Preview-SQLs (`training_responses`, „Kein-Response-Trainings", `game_responses`) als Schleife über `member_ids` ausführen und die Ergebnisse über eine Map dedupen (Key: `event_type+event_id`)
- [x] 2.4 Response unverändert: `[{event_id, event_type, name, date}, …]` sortiert nach Datum

## 3. Backend — Tests

- [x] 3.1 `internal/absences/handler_test.go`: Test `TestCreateAbsence_MultiChild_Success` — Eltern legt für 2 Kinder Urlaub an → 2 `member_absences`-Zeilen, beide mit korrektem Zeitraum
- [x] 3.2 Test `TestCreateAbsence_MultiChild_AllOrNothing` — eines der Kinder hat überlappende Abwesenheit → 409 mit beiden conflicts gelistet, **keine** Zeile inserted
- [x] 3.3 Test `TestCreateAbsence_Legacy_SingleMemberID` — Aufruf mit altem `member_id`-Feld funktioniert wie heute (201, eine Zeile)
- [x] 3.4 Test `TestPreview_MultiChild_Union` — Preview für 2 Kinder mit überlappenden Trainings liefert dedupliziert

## 4. Frontend

- [x] 4.1 Form-State umstellen: `member_id: 0` → `member_ids: number[]`. Beim Öffnen des Wizards mit `eventType==='abwesenheit'`: wenn `children.length === 1`, `member_ids` direkt mit `[children[0].id]` initialisieren
- [x] 4.2 Wizard Step 2 Rendering:
  - `children.length === 0` (Spieler-Pfad): keine Auswahl, wie bisher
  - `children.length === 1`: keine Auswahl rendern
  - `children.length > 1`: Checkbox-Liste, mindestens 1 muss aktiv sein; Validierung in `handleAbsencePreview`
- [x] 4.3 `handleAbsencePreview`: `member_ids` als CSV in den Query-Param packen (`member_ids=1,2`); bei `children.length === 0` Fallback wie heute
- [x] 4.4 `doSaveAbsence`: Body mit `member_ids` (oder Single bei Spieler-Pfad — dann gar nicht senden)
- [x] 4.5 Konflikt-Fehler-Handling im Error-State: bei 409 mit `conflicts[]` Aufzählung anzeigen („Eintragung abgebrochen — {Namen-Liste} hat/haben in diesem Zeitraum bereits eine Abwesenheit.")
- [x] 4.6 Touch-Targets der Checkbox-Liste: `py-2.5` Mobile gemäß CLAUDE.md

## 5. Verifikation

- [ ] 5.1 Manuell als Elternteil mit 2 Kindern: Urlaub 14.–21.06. für beide eintragen → zwei Zeilen in `member_absences`, beide Kalender-Banner sichtbar
- [ ] 5.2 Manuell: gleicher Urlaub, ein Kind hat schon was im Zeitraum → 409, Fehlertext nennt das Kind, **keine** Zeile in DB inserted
- [ ] 5.3 Manuell als Elternteil mit nur 1 Kind: Auswahl-UI ist weg, Speichern funktioniert wie zuvor
- [ ] 5.4 Manuell als Spieler ohne Kinder: Pfad unverändert (eigener Member)
