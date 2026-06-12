## Context

`POST /api/absences` und `GET /api/absences/preview` arbeiten heute mit einem einzelnen `member_id`. Der Handler-Code (`internal/absences/handler.go`) prüft via `resolveMemberID` die Berechtigung des Aufrufers (Spieler → eigener Member; Elternteil → `family_links`-Eintrag), prüft Überschneidungen über `member_absences`, inserted eine Zeile und macht anschließend Auto-Decline der Trainings-/Spiel-Responses im Zeitraum.

Frontend (`KalenderPage.tsx`, Wizard Step 2 für `eventType === 'abwesenheit'`) zeigt für `user?.isParent` ein `<select>` mit den Kindern aus `r.data.children` (geladen einmal beim Öffnen). Der Submit ruft erst `GET /api/absences/preview`, zeigt — falls Events betroffen — ein Bestätigungs-Modal, und dann `POST /api/absences`.

## Goals / Non-Goals

**Goals:**
- Familien-Urlaub für N Kinder in einem Submit eintragen
- Eine UI-Komponente weniger zu bedienen, wenn nur 1 Kind verlinkt ist
- All-or-nothing: keine Halbzustände bei Konflikt

**Non-Goals:**
- Keine Aggregations-Entity „Familien-Abwesenheit" im Datenmodell — Einträge bleiben strikt pro Member
- Keine per-Kind-Felder (unterschiedlicher Typ/Notiz/Datum) — Komfort-Gewinn würde verpuffen
- Kein Eltern-übergreifender Multi-Select (Eltern können nur ihre eigenen Kinder eintragen — unverändert)
- Kein neuer `/api/absences/batch`-Endpoint — die bestehende Route bekommt `member_ids[]` als Erweiterung, sauberer als ein zweiter Endpoint mit fast identischer Semantik

## Decisions

**Erweiterung der bestehenden Route statt neuer `/batch`-Pfad.**  
Argument für eine eigene `/batch`-Route wäre die klare Trennung. Argument dagegen: Die Semantik ist exakt dieselbe — Validieren, Konflikt-Prüfung, Insert, Auto-Decline — nur eben über N Members statt 1. Eine zweite Route würde dieselbe Logik verdoppeln. Stattdessen: Request akzeptiert `member_ids: []int` ODER `member_id: int`; intern wird daraus eine `[]int` normalisiert.

```go
type req struct {
    MemberID   int    `json:"member_id"`
    MemberIDs  []int  `json:"member_ids"`
    Type       string `json:"type"`
    StartDate  string `json:"start_date"`
    EndDate    string `json:"end_date"`
    Note       string `json:"note"`
}

ids := req.MemberIDs
if len(ids) == 0 && req.MemberID > 0 {
    ids = []int{req.MemberID}
}
if len(ids) == 0 {
    // Spieler-Fallback wie heute via resolveMemberID(claims, 0)
}
```

**All-or-nothing über zwei Phasen:**  
Phase 1 — Berechtigung + Konflikt-Prüfung außerhalb der Transaktion:

```go
for _, mid := range ids {
    resolved, errMsg := resolveMemberID(ctx, claims, mid)
    if errMsg != "" { 403 oder 400 }
    // Konflikt-Check pro Member
    if overlap > 0 { conflicts = append(conflicts, {member_id, name}) }
}
if len(conflicts) > 0 {
    409 Conflict + {"conflicts": [...]}
    return
}
```

Phase 2 — Insert + Auto-Decline in einer Transaktion:

```go
tx, _ := db.BeginTx(...)
defer tx.Rollback()
for _, mid := range resolvedIDs {
    INSERT INTO member_absences ...
    Auto-Decline-SQL für training_responses + game_responses
}
tx.Commit()
```

Die Konflikt-Prüfung läuft *vor* der Transaktion, damit der 409-Pfad ohne aufgeräumten Insert-Versuch funktioniert. Das ist marginal anfällig für eine Race-Condition (ein anderer Tab könnte zwischen Check und Insert eine konkurrierende Abwesenheit anlegen), aber der Worst Case ist ein Constraint-Fail beim Insert → Rollback → 500. Pragmatisch akzeptabel im Vereinskontext.

**Response-Format bei Erfolg:**  
HTTP 201, Body `{ "absence_ids": [123, 124, 125] }` — in derselben Reihenfolge wie die übergebenen `member_ids`. Bei legacy `member_id`-Aufruf: weiterhin leerer Body wie heute (kein Breaking-Change).

**Response-Format bei Konflikt:**

```json
HTTP 409
{
  "error": "overlap",
  "conflicts": [
    { "member_id": 42, "member_name": "Ben Frank" },
    { "member_id": 47, "member_name": "Clara Frank" }
  ]
}
```

Das alte `{ "error": "overlap" }`-Format bleibt im Single-Member-Pfad gültig — Frontend-Logik kann am Vorhandensein von `conflicts` unterscheiden.

**Preview-Erweiterung:**  
`GET /api/absences/preview?member_ids=1,2,3&from=…&to=…`. Comma-separated als Query-Param ist konsistent mit Chi-Konventionen und einfacher zu cachen. Wenn `member_ids` fehlt, fällt es auf `member_id` zurück. Server-seitige Aggregation:

```sql
SELECT DISTINCT event_id, type, name, date FROM (
    -- Training-Events für Kind 1
    UNION
    -- Training-Events für Kind 2
    UNION
    -- Spiel-Events für Kind 1
    ...
)
ORDER BY date
```

Performance-mäßig ist N klein (typisch 2–3), kein Problem.

**UI-Verhalten bei 1 Kind:**  
Heute zeigt der Wizard auch bei 1 Kind das Select mit „Bitte wählen…" als Default und blockt Submit bis ausgewählt. Mit der Änderung wird die Auswahl bei `children.length === 1` ganz weggelassen — `member_ids` wird beim Mounten direkt mit dem einen Kind initialisiert. Konsistenz mit der Spieler-Variante, die heute schon keinerlei Auswahl zeigt.

**Checkbox-Liste statt Multi-Select:**  
Beim N=2-3-Fall wäre ein `<select multiple>` zwar kompakter, aber auf Mobile schlecht zu bedienen (Touch-Target-Größe, plattformabhängige Darstellung). Checkbox-Liste mit `py-2.5`-Targets ist robuster. „Alle wählen"-Shortcut bei N≥3 wäre nett, sparen wir uns aber für einen späteren Convenience-Change auf.

## Risks / Trade-offs

**API-Erweiterung statt neuer Endpoint.** Ein zukünftiger Reviewer muss verstehen, dass `member_id` UND `member_ids` koexistieren. Ich dokumentiere das im Handler-Kommentar. Alternative wäre eine v2-Route, die hier overengineered wäre.

**All-or-nothing kann frustrieren.** Wenn Eltern einen Konflikt bei *einem* Kind haben (z.B. der ältere hat schon eine Verletzungspause eingetragen, die mit dem geplanten Urlaub überlappt), muss er erst den Konflikt klären und dann erneut alle Kinder auswählen. Das war die explizite Anwender-Präferenz — Alternative wäre best-effort mit Per-Kind-Status, die du verworfen hast. Wir bleiben bei deinem Wunsch.

**Race-Condition Konflikt-Prüfung ↔ Insert.** Theoretisch möglich, in der Praxis irrelevant. Worst Case: 500-Fehler durch Constraint-Verletzung, Eltern probiert nochmal. Nicht wert, Pessimistic-Locking einzuführen.

**Backwards-Compat des 409-Bodys.** Wenn jemand `member_id` (single) sendet und es einen Konflikt gibt, behalten wir das alte `{"error":"overlap"}` ohne `conflicts`-Liste. Das Frontend, das wir jetzt ändern, geht stets über `member_ids` — also irrelevant für den Hauptpfad, aber alte Clients (z.B. eine Test-Curl) sehen kein neues Format.

**Preview-Query-Param als CSV.** `?member_ids=1,2,3` ist HTTP-konventionell und schon im Carpooling/Members-API so verwendet. Alternative `?member_id=1&member_id=2&member_id=3` (repeating) ist auch valid, aber Go's `r.URL.Query().Get("member_id")` liefert nur den ersten — wir müssten `r.URL.Query()["member_id"]` verwenden. CSV-Splitting ist hier sauberer und symmetrisch zum POST-Body.
