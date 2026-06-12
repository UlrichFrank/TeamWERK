## 1. Backend: Preview-Endpoint erweitern

- [x] 1.1 In `internal/absences/handler.go` das `previewEvent`-Struct um ein Feld `Pending bool json:"pending"` erweitern
- [x] 1.2 Im `Preview`-Handler einen zweiten Query-Block hinzufügen: Training-Sessions ohne bisherige Response, bei denen der Member Kader-Mitglied ist (analog zur Auto-Decline-Query in `Create`)
- [x] 1.3 Sicherstellen, dass Duplikate vermieden werden (Sessions, die bereits als `confirmed` zurückgegeben wurden, nicht nochmals als `pending` anzeigen)

## 2. Frontend: KalenderPage nach Absence-Save aktualisieren

- [x] 2.1 In `doSaveAbsence()` nach dem POST `loadTrainings()` parallel zu `loadAbsences()` aufrufen
- [x] 2.2 In `useLiveUpdates` Handler für `"trainings"`-Event ergänzen: `if (event === 'trainings') loadTrainings()`

## 3. Frontend: Preview-Anzeige anpassen

- [x] 3.1 Preview-Liste im Wizard um `pending: true`-Sessions ergänzen (andere Darstellung, z.B. gedimmte Farbe oder Label „Offen" statt „Bestätigt")
- [x] 3.2 Überschrift der Preview-Liste anpassen: statt „Bestehende Zusagen werden zurückgezogen" allgemeiner formulieren, z.B. „Folgende Trainings werden automatisch abgesagt"
