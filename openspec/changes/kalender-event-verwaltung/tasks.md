## 1. Wizard-Erweiterung: Training und Trainingsserie

- [x] 1.1 `KalenderPage.tsx` — Schritt 1: Optionen `'training'` und `'serie'` zum `eventType`-Union und zur Auswahl-Liste hinzufügen; nur sichtbar wenn `hasFunction(user, 'manage-trainings')` oder `role in ['trainer', 'admin']`; Icons: `Dumbbell` (Training), `RefreshCw` (Serie)
- [x] 1.2 `KalenderPage.tsx` — neue States: `trainingStartTime`, `trainingEndTime`, `trainingLocation` (jeweils `useState('')`) und `seriesWeekday` (`useState(1)`), `seriesValidFrom`, `seriesValidUntil`
- [x] 1.3 `KalenderPage.tsx` — Schritt 2 für `eventType === 'training'`: Felder Datum (vorbelegt mit `selectedDate`), Start-/Endzeit, Ort, Mannschaft (Single-Select auf aktive Teams); Submit ruft `POST /api/training-sessions` auf
- [x] 1.4 `KalenderPage.tsx` — Schritt 2 für `eventType === 'serie'`: Felder Wochentag (`<select>` Mo–So), Start-/Endzeit, Ort, Mannschaft, Gültig-von / Gültig-bis; Submit ruft `POST /api/training-series` auf
- [x] 1.5 `KalenderPage.tsx` — Reset-Logik in `closeDialog` um neue States erweitern; nach erfolgreichem Submit `loadTrainings()` aufrufen

## 2. Inline-Edit für Trainingstermine

- [x] 2.1 Neue Komponente `web/src/components/TrainingEditModal.tsx`: Props `session` (Training-Interface), `onClose`, `onSaved`; zeigt Formular mit Feldern aus `PUT /api/training-sessions/{id}` (Datum, Start-/Endzeit, Ort, Status, Cancel-Reason); wenn `session.series_id` vorhanden: Radio-Gruppe „Nur dieser Termin / Dieser und folgende / Alle der Serie"
- [x] 2.2 `TrainingEditModal.tsx` — Scope-Logik: „Nur dieser Termin" → `PUT /api/training-sessions/{id}`; „Dieser und folgende" → `PUT /api/training-series/{series_id}?scope=this_and_following&from_date=...`; „Alle der Serie" → `PUT /api/training-series/{series_id}?scope=all`
- [x] 2.3 `KalenderPage.tsx` — Training-Klick-Handler: wenn `role in ['trainer', 'admin']` → `setEditingTraining(t)` (neuer State); sonst → `navigate('/trainings/' + t.id)` (unverändert)
- [x] 2.4 `KalenderPage.tsx` — `TrainingEditModal` einbinden: `{editingTraining && <TrainingEditModal session={editingTraining} onClose={() => setEditingTraining(null)} onSaved={() => { loadTrainings(); setEditingTraining(null) }} />}`
