## Why

Eltern mit mehreren Kindern im Verein (typisch: Geschwisterpaare in zwei Mannschaften) tragen Familienurlaub heute einzeln pro Kind ein. Drei mal denselben Zeitraum tippen, drei mal Preview bestätigen, drei mal speichern — und wenn dazwischen ein Vorschlag stört oder die Konflikt-Liste verwirrt, doppelt so frustrierend. Die Einträge selbst sind und bleiben pro Kind (`member_absences` ist nicht aggregierbar — jedes Kind hat einen eigenen Kader, eigene Auto-Decline-Wirkungen), aber das *Eintragen* darf einmal genügen.

## What Changes

- **Backend: `member_ids[]` ergänzen** an `POST /api/absences` und `GET /api/absences/preview`. Falls vorhanden, ersetzt `member_ids` das alte `member_id`; ist beides nicht gesetzt, gilt die heutige Fallback-Logik (eigene Spieler-Abwesenheit). `member_id` bleibt akzeptiert (backwards-compatible für den Spieler-Pfad und alte Clients).
- **All-or-nothing-Semantik**: Wenn auch nur eines der ausgewählten Kinder im angegebenen Zeitraum mit Typ `vacation`/`injury` überschneidet, wird *nichts* eingetragen. Response: `409 Conflict` mit `{ conflicts: [{member_id, member_name}] }`, damit das Frontend gezielt sagen kann, bei welchem Kind der Konflikt liegt.
- **Transactional Insert**: Beim Sammel-Insert läuft alles in einer SQL-Transaktion, damit ein Mid-Loop-Fehler (DB-Down o.ä.) keine Teilstände hinterlässt. Auto-Decline der Training-/Spielzusagen läuft pro Kind im selben Block.
- **Preview-Union**: `GET /api/absences/preview?member_ids=1,2,3&from=&to=` liefert die Vereinigung der betroffenen Events über alle ausgewählten Kinder (jedes Event nur einmal, sortiert nach Datum). Heute wird derselbe Endpoint mit `member_id=N` betrieben und liefert nur die Treffer eines Kindes.
- **Frontend Wizard Step 2** auf `KalenderPage.tsx`:
  - Bei `children.length === 0` (kein Kind verlinkt — z.B. Spieler-Account): keine Auswahl, das Form arbeitet wie heute auf dem eigenen Member.
  - Bei `children.length === 1`: Auswahl-UI weggelassen, das eine Kind wird automatisch verwendet (Vereinfachung gegenüber heute, wo „Bitte wählen…" als Default angezeigt wird).
  - Bei `children.length > 1`: Checkbox-Liste mit den verfügbaren Kindern, mindestens eines muss ausgewählt sein.
- Form-State `member_id: number` wird zu `member_ids: number[]`. Typ, Start-/Enddatum und Notiz bleiben pro Anlegevorgang einheitlich (keine per-Kind-Felder).
- Konflikt-Fehler-Copy im Wizard: „Eintragung abgebrochen — {Kindname} hat in diesem Zeitraum bereits eine Abwesenheit." Bei mehreren Konflikten alle aufgezählt.

## Capabilities

### Modified Capabilities

- `member-absences` — `POST /api/absences` und `GET /api/absences/preview` akzeptieren `member_ids[]`; Frontend zeigt Multi-Select für Eltern mit mehreren Kindern.

## Impact

- `internal/absences/handler.go`: Request-Struct um `MemberIDs []int` erweitern; `resolveMemberID` zu `resolveMemberIDs` ergänzen (oder Schleife mit bestehender Funktion); Konflikt-Prüfung über alle `member_ids` vor dem Insert; Insert + Auto-Decline pro Kind in einer Transaktion.
- `internal/absences/handler.go`, `Preview`: zusätzlich `member_ids`-Query-Param parsen, UNION der Events pro Kind aggregieren.
- `web/src/pages/KalenderPage.tsx`: Form-State, Step-2-Rendering, `handleAbsencePreview` und `doSaveAbsence` auf Liste umstellen; Konflikt-Response-Handling (409 mit `conflicts[]`).
- `internal/absences/handler_test.go` (falls vorhanden) bzw. neuer Test: Multi-Child-Insert ohne Konflikt, Multi-Child-Insert mit Konflikt → all-or-nothing.
- **Keine DB-Migration**, kein Schema-Change.
- **Backwards-compatible**: alte Clients mit `member_id` funktionieren unverändert weiter; die alten Test-Scenarien für „Elternteil legt für ein Kind an" bleiben gültig.
