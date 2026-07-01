## Context

Heute lassen sich RSVP-Antworten bis zum Termin-Start (und sogar danach via Trainer-Override implizit über `member_id`) ändern. Die einzigen Sperren sind:

1. **Status-Validierung** (`confirmed | declined | maybe`) — 400 bei ungültigen Werten.
2. **Absence-Lock** — wenn `game_responses.absence_id IS NOT NULL` bzw. `training_responses.absence_id IS NOT NULL` gesetzt ist (vom System aus einer Member-Abwesenheit gespiegelt), liefert die API 403.

Es gibt keine zeitliche Begrenzung. Das führt in der Praxis zu Spätabsagen, mit denen Trainer und Mannschaft nicht mehr planen können. Mit diesem Change führen wir einen Cutoff ein — für Trainings 2 Stunden, für Spiele 18 Stunden vor Beginn.

**Berechtigungsmodell als Referenz:** TeamWERK hat zwei orthogonale Dimensionen — `Role` (`admin` / `standard`) und `ClubFunctions` (mehrwertig: `spieler`, `trainer`, `sportliche_leitung`, `vorstand`, `kassierer`, …). „Eltern" sind keine Vereinsfunktion, sondern `claims.IsParent` (aus `family_links`). Der bestehende Helper `claims.IsTrainerLike()` deckt `trainer` + `sportliche_leitung` ab.

**Zeitfelder:**
- `training_sessions`: `date` (ISO `YYYY-MM-DD`), `start_time` (`HH:MM`, ohne Sekunden).
- `games`: `date` (ISO), `time` (`HH:MM`).
- Datums-/Zeitfelder liegen unaufgelöst lokal (Europe/Berlin) in der Datenbank.
- Der Server läuft per Default auf UTC (`datetime('now')`).

## Goals / Non-Goals

**Goals:**

- Spieler und Eltern können ihren RSVP-Status für Trainings nur bis 2 h vor Beginn, für Spiele nur bis 18 h vor Beginn ändern.
- Der Cutoff sperrt **jeden** Statuswechsel (Neuanlage, Wechsel, Reason-Änderung) — nicht nur den Übergang nach `declined`.
- Trainer / sportliche_leitung / Vorstand / Admin können jederzeit pflegen, damit Anwesenheitslisten realistisch bleiben.
- Das Frontend kennt die Sperrzeit vorab (kein clientseitiges Rückwärtsrechnen aus Datum/Zeit), zeigt vor Cutoff einen subtilen Hinweis, danach Buttons disabled mit Klartext.

**Non-Goals:**

- Cutoff für die Vereinsabsage eines kompletten Termins (`status='cancelled'` auf Game/Session).
- Cutoff für die nachträgliche Anwesenheits-Eintragung (`/attendances`-Endpoints).
- Pre-Cutoff-Push („letzte Chance abzusagen") — separater Vorschlag.
- Per-Verein konfigurierbare Cutoff-Werte.
- Änderung an `member_absences`-Lock-Verhalten (bleibt 403, hat Vorrang vor 422).

## Decisions

### 1. Cutoff hart als Konstante im Domain-Package

```go
// internal/trainings/handler.go
const TrainingRSVPCutoff = 2 * time.Hour

// internal/games/handler.go
const GameRSVPCutoff = 18 * time.Hour
```

**Warum nicht konfigurierbar?** YAGNI — wir haben einen Verein. Solange kein zweiter mit abweichendem Bedürfnis kommt, vermeiden wir Konfig-Oberflächen, Migrationen und Test-Aufwand. Wenn er kommt, ziehen wir die Konstante nach `clubs.rsvp_training_cutoff_minutes` etc. um — niedriger Refactoring-Schmerz.

**Alternative verworfen:** Pro `game_templates` / `training_series` konfigurierbar — zu früh, würde UI-Aufwand pro Template-Editor erzeugen.

### 2. Zeitberechnung in Go, nicht in SQL

```go
loc, _ := time.LoadLocation("Europe/Berlin")
start, err := time.ParseInLocation("2006-01-02 15:04", date+" "+startTime, loc)
locksAt := start.Add(-TrainingRSVPCutoff)
if time.Now().After(locksAt) { /* gesperrt */ }
```

**Warum Go?** SQLite `datetime(date || ' ' || time, '-2 hours')` rechnet als UTC und ignoriert DST. Berlin wechselt zwei Mal pro Jahr; wir würden uns systematische Stunden-Fehler in den 2 DST-Wochen einfangen. Go mit `time.LoadLocation("Europe/Berlin")` löst DST korrekt auf, kostet pro Request einen Memory-Lookup (kein I/O — die Zoneinfo-DB ist beim Start geladen).

**Alternative verworfen:** UTC-Zeitfelder in DB ablegen. Wäre sauberer, ist aber ein größerer Schnitt (Migration aller Bestandsdaten, alle Listing-Queries anpassen). Nicht Scope dieser Änderung.

### 3. Override-Check als gemeinsamer Helper

Neuer Helper auf `*Claims`, weil das Muster sonst dreimal hingeschrieben würde (Training, Game, ggf. später):

```go
// internal/auth/tokens.go
func (c *Claims) CanOverrideRSVPCutoff() bool {
    return c.Role == "admin" ||
        c.HasFunction("vorstand") ||
        c.IsTrainerLike() // trainer || sportliche_leitung
}
```

`kassierer` ist bewusst **nicht** drin. `vorstand_beisitzer` ebenfalls nicht (für Mannschaftsplanung nicht zuständig).

**Alternative verworfen:** Team-spezifischer Check „ist Trainer **dieses** Teams". Wir verzichten darauf — wer als Trainer/sL/Vorstand markiert ist, hat ohnehin Schreibzugriff auf RSVPs des Teams via bestehender Routen. Ein Team-Scope würde uns zusätzliche DB-Lookups einbringen ohne neuen Schutz.

### 4. Reihenfolge der Sperren: Absence vor Cutoff

Wenn ein Member sowohl eine Absence über den Termin als auch nach Cutoff antworten will, bleibt die ursprüngliche **403** (Absence-Lock) erhalten, **nicht 422**. Rationale: der Absence-Lock ist eine semantische Sperre („du hast eine Abwesenheit eingetragen — lösche die zuerst"), während 422 zeitabhängig ist. Reihenfolge im Handler: Validierung → Member-Resolution → Absence-Check → **Cutoff-Check** → Upsert.

### 5. `rsvp_locks_at` in API-Responses

Format: RFC3339 in **UTC** (`2026-06-30T16:00:00Z`), berechnet aus lokaler Termin-Zeit (Europe/Berlin) – `cutoff`. Liegt pro Termin/Spiel im JSON, nicht nur in Detail-Responses. Frontend rendert `new Date(rsvp_locks_at)` und kann mit `Date.now()` direkt vergleichen.

**Welche Endpoints?** Alle, die das Frontend für die Buttons braucht:
- `GET /api/training-sessions` (Liste), `GET /api/training-sessions/{id}` (Detail).
- `GET /api/games` (Vorstand-Listing), `GET /api/games/my` (User-Listing), `GET /api/games/{id}` (Detail).

Nicht in Routen, die nur aggregierte Stats liefern (`/attendance-stats`, `/responses`).

### 6. Fehler-Payload

```json
{
  "error": "rsvp_locked",
  "message": "Training kann nur bis 2 Stunden vor Beginn umgesagt werden.",
  "locks_at": "2026-06-30T16:00:00Z"
}
```

HTTP-Status **422 Unprocessable Entity** (semantisch korrekt: Request ist syntaktisch ok, aber fachlich nicht verarbeitbar). Frontend rendert `message` direkt; `locks_at` ist nur Bonus-Info.

**Alternative verworfen:** 403 Forbidden — würde den Absence-Lock-Fehler und den Cutoff-Fehler ununterscheidbar machen.

### 7. Testbarkeit: Clock-Injection

Damit Tests nicht gegen Echtzeit kämpfen müssen, bekommt `*trainings.Handler` und `*games.Handler` ein `now func() time.Time`-Feld:

```go
type Handler struct {
    db  *sql.DB
    hub *hub.EventHub
    now func() time.Time // default: time.Now
}
```

`NewHandler(db, hub)` setzt `now: time.Now`. Tests injizieren `h.now = func() time.Time { return fixedTime }`. Kein neues Interface, keine externe Lib.

**Alternative verworfen:** `clockwork`-Library — overkill für das eine Use-Case.

## Risks / Trade-offs

- **Risk:** Spieler werden 30 Min vor Training krank und können nicht mehr absagen — Trainer plant mit „confirmed", obwohl er nicht kommt.
  → **Mitigation:** Hinweis-Text macht klar, dass Absage telefonisch/per Chat zu erfolgen hat. Trainer kann via UI nachträglich pflegen (siehe Decision 3). Anwesenheitsstatistik (`training_attendances`) wird ohnehin später am realen „war da / war nicht da" gepflegt — nicht aus RSVP abgeleitet.

- **Risk:** DST-Wechsel: am Sonntag der Zeitumstellung könnte ein Training um 03:00 oder 02:00 Uhr starten und der Cutoff-Vergleich kippt.
  → **Mitigation:** `time.ParseInLocation` + `time.LoadLocation("Europe/Berlin")` handhabt DST nativ. Smoke-Test mit einem Datum in der Sommer- und einem in der Winterzeit.

- **Risk:** Übergangsphase nach Deploy — laufende Termine, deren `rsvp_locks_at` bereits in der Vergangenheit liegt, werden plötzlich für Spieler gesperrt.
  → **Mitigation:** akzeptiert. Trainer kann nachpflegen; Hinweis-Text leitet Spieler an. Kein Migrationsschritt nötig.

- **Trade-off:** Hartes Frontend-Lock (Variante B) wirkt rigider als Soft-Lock mit Pflicht-Reason. Wir gehen das Risiko ein, weil die Pflichtreibung gerade der Zweck der Änderung ist.

- **Trade-off:** Kein Pre-Cutoff-Reminder-Push. Spieler, die das Termin-Tab nicht öffnen, bemerken die Sperre erst, wenn sie es versuchen. Akzeptiert; späterer Vorschlag kann das ergänzen.

## Open Questions

_(keine — wenn beim Implementieren etwas auffällt, hierher zurückkehren oder per `/openspec-explore` neue Fragen aufmachen)_
