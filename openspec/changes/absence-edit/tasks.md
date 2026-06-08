## 1. Backend — PUT /api/absences/{id}

- [ ] 1.1 `Update`-Handler in `internal/absences/handler.go` anlegen: Berechtigungsprüfung (creator oder admin), Overlap-Check mit eigener ID ausgenommen, Felder aktualisieren
- [ ] 1.2 Auto-Decline-Restore: bestehende Responses mit dieser `absence_id` zurücksetzen (`status='confirmed', absence_id=NULL`)
- [ ] 1.3 Auto-Decline neu anlegen für neuen Zeitraum (gleiche Logik wie in `Create`)
- [ ] 1.4 Route `PUT /api/absences/{id}` in `cmd/teamwerk/main.go` eintragen (Authenticated-Gruppe)
- [ ] 1.5 `hub.Broadcast("absences")`, `hub.Broadcast("trainings")`, `hub.Broadcast("games")` nach erfolgreichem Update

## 2. Frontend — infoItem-State und Klick-Handler

- [ ] 2.1 `infoItem`-Typ in `KalenderPage.tsx` um `type: 'absence'` und `absence?: Absence` erweitern
- [ ] 2.2 Klick-Handler auf Abwesenheitsbalken: `onPointerDown={e => e.stopPropagation()}` + `onClick={() => setInfoItem({ type: 'absence', absence })}`
- [ ] 2.3 `EventInfoModal`-Aufruf: `absence`-Prop übergeben, `onEdit`/`onDelete` für Absence-Fall verdrahten

## 3. Frontend — EventInfoModal Absence-Zweig

- [ ] 3.1 Props um `absence?: Absence`, `onDeleteAbsence?: () => void` erweitern; `type` um `'absence'` ergänzen
- [ ] 3.2 Absence-Anzeige-Zweig: Typ-Label, Mitgliedsname, Zeitraum (von–bis formatiert), Notiz
- [ ] 3.3 Bearbeiten- und Löschen-Button nur wenn `onEdit` bzw. `onDeleteAbsence` übergeben
- [ ] 3.4 Inline-Edit-Modus (`editMode`-State): Formular mit Typ-Select, Start-/Enddatum-Inputs, Notiz-Textarea
- [ ] 3.5 Speichern-Handler: `PUT /api/absences/{id}` aufrufen, bei 409 Fehlermeldung „Überschneidung mit bestehender Abwesenheit", bei Erfolg Edit-Modus schließen und `onSaved`-Callback

## 4. Frontend — Löschen aus Modal

- [ ] 4.1 `onDeleteAbsence` in `KalenderPage`: `DELETE /api/absences/{id}` aufrufen, Modal schließen, `loadAbsences()` aufrufen
