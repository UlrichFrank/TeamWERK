## Context

Spiele und Trainings im Kalender sind anklickbar: Klick → `setInfoItem` → `EventInfoModal` (Anzeige) → `onEdit` → `GameEditModal` / `TrainingEditModal`. Abwesenheiten haben bisher keinen Klick-Handler. Das Backend hat `POST` und `DELETE` für Abwesenheiten, aber kein `PUT`.

`EventInfoModal` nimmt `type: 'game' | 'training'` als Diskriminator. Der `infoItem`-State in `KalenderPage` ist entsprechend typisiert. Der Auth-Context liefert `user.id` und `user.role` für Berechtigungsprüfungen im Frontend.

## Goals / Non-Goals

**Goals:**
- Konsistentes Klick-Muster: Abwesenheitsbalken verhält sich wie Event-Pills
- Info-Anzeige und Edit-Formular in einem Modal (kein separates EditModal nötig, da wenige Felder)
- `PUT /api/absences/{id}` mit Überlappungscheck (eigene ID ausgenommen)
- Berechtigungsprüfung: nur Ersteller oder Admin darf bearbeiten/löschen

**Non-Goals:**
- Separate `AbsenceEditModal`-Komponente
- Admin-seitige Übersicht aller Abwesenheiten (anderes Feature)
- Änderung des Auto-Decline-Verhaltens bei Edit

## Decisions

### Entscheidung: Absence-Zweig in EventInfoModal statt eigener Komponente

**Gewählt:** `EventInfoModal` bekommt `type: 'game' | 'training' | 'absence'` und rendert einen dritten Zweig. Inline-Edit-State (`editMode: boolean`) direkt in der Komponente.

**Warum:** Abwesenheiten haben nur 3 editierbare Felder (Typ, Datum-Range, Notiz). Ein separates Modal wäre Overengineering. Das Info- und Edit-Formular im selben Modal (toggle `editMode`) hält die UX schlank — analog zu anderen kleinen Inline-Edit-Patterns im Projekt.

**Alternative verworfen:** Eigene `AbsenceInfoModal`-Komponente. Würde Duplizierung der Modal-Shell (Overlay, Header, Close-Button, Escape-Key) bedeuten.

### Entscheidung: infoItem-State um Absence erweitern

**Gewählt:** `infoItem` Typ wird zu `{ type: 'game' | 'training' | 'absence'; game?: Game; training?: Training; absence?: Absence } | null`. Klick auf Balken: `setInfoItem({ type: 'absence', absence })`.

**Warum:** Minimale Änderung am bestehenden State-Modell. Kein zweiter State nötig.

### Entscheidung: PUT-Endpoint mit Overlap-Check (eigene ID ausgenommen)

**Gewählt:** `PUT /api/absences/{id}` prüft Überlappung wie `POST`, schließt aber die eigene ID aus:
```sql
SELECT COUNT(*) FROM member_absences
WHERE member_id = ? AND type = ?
  AND start_date <= ? AND end_date >= ?
  AND id != ?
```
Berechtigungsprüfung: `created_by = claims.UserID` oder `claims.Role = 'admin'`.

**Warum:** Ohne Ausschluss der eigenen ID würde jedes Speichern ohne Datumsänderung fälschlicherweise mit sich selbst kollidieren.

### Entscheidung: Auto-Decline bei Edit neu auslösen

**Gewählt:** Beim `PUT` werden Auto-Decline-Responses wie beim `POST` neu gesetzt: zuerst alle bestehenden Responses mit dieser `absence_id` auf `confirmed` zurücksetzen (Restore), dann für den neuen Zeitraum neu declinen.

**Warum:** Wenn der Zeitraum verändert wird, müssen Events außerhalb des neuen Zeitraums wieder freigegeben werden und neue Events im neuen Zeitraum gedeclint werden.

## Risks / Trade-offs

- [Restore-Logic beim Edit komplex] → Implementierung: erst `UPDATE training_responses SET status='confirmed', absence_id=NULL WHERE absence_id=?`, dann Auto-Decline neu ausführen. Bewusst einfach gehalten — kein diff-Ansatz.
- [EditModal zeigt veraltete Daten wenn anderer Tab ändert] → vernachlässigbar, SSE-Reload nach Broadcast aktualisiert `absences`-State
