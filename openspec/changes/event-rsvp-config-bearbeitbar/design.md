## Context

Die Spalten `rsvp_opt_out` und `rsvp_require_reason` existieren seit Migration 015 auf den drei
betroffenen Tabellen (`games`, `training_series`, `training_sessions`) — DB-seitig ist nichts
zu tun. Der Defekt liegt durchgängig auf Application-Ebene:

| Layer | `games` | `training_sessions` | `training_series` |
|---|---|---|---|
| Backend CREATE | ✓ akzeptiert | ✓ akzeptiert | ✓ akzeptiert |
| Backend UPDATE | ✗ nicht im Request-Struct (`internal/games/handler.go:731-740`) | ✗ nicht in `UpdateSession` (`internal/trainings/handler.go:617-619`) | ✓ akzeptiert (`UpdateSeries` 379-380) |
| Frontend Create | ✗ keine UI im `GameEditModal` | ✓ in `AdminTrainingsPage:517,527`, aber via `disabled={!isNewSeries}` nur bei NEUER Serie | ✓ siehe Sessions |
| Frontend Edit | ✗ nicht vorhanden | ✗ disabled | ✗ disabled |
| Detail-Badge | ✗ keine Anzeige | ✗ keine Anzeige | n/a |

Folge im Betrieb: Wer beim Anlegen die Standardwerte (`0` / `1`) nimmt, kann sie nie wieder
ändern — und sieht im UI auch nicht, welcher Modus aktiv ist. Genau dieses Verhalten hat die
aktuelle Frage „bei Spiel 39 sehe ich nicht, ob Opt-Out aktiv ist" ausgelöst.

Die bestehenden Live-Update-Hubs (`games`, `trainings`) lassen sich ohne Anpassung wiederverwenden,
da die UPDATE-Handler ohnehin schon broadcasten.

## Goals / Non-Goals

**Goals:**
- `UpdateGame` und `UpdateSession` akzeptieren die beiden Felder zusätzlich; `UpdateSeries` bleibt unverändert (funktioniert schon).
- `GameEditModal` zeigt beide Felder als Checkboxen (Create + Edit, ein UI-Pfad).
- `AdminTrainingsPage` öffnet die bisher gesperrten Checkboxen auch im Edit-Modus für Session und Series.
- Aktueller Wert ist sichtbar in der Detailansicht — Badge im Termin-Header.
- Partial-Update-Semantik: fehlt das Feld im Request, bleibt der DB-Wert unverändert (kein impliziter Reset auf `0`).
- Permission: identisch zu den anderen Edit-Operationen am selben Endpoint — keine neue Rollen-Matrix.

**Non-Goals:**
- Der Counter-Bug zwischen `ListMyGames` (Kalender, opt-out-aware) und `ListGames` / `GetParticipants` (nicht opt-out-aware) wird hier NICHT behoben. Folge-Change.
- Keine Migration und kein Datendurchlauf — Defaults bleiben für bestehende Termine.
- Kein Audit-Log für Konfigurationsänderungen (kann später ergänzt werden).
- Keine Anpassung an `CreateGame` / `CreateSession` — die Felder werden dort bereits akzeptiert.

## Decisions

### Entscheidung 1 — Partial-Update statt vollständige Ersetzung

Die UPDATE-Endpoints behandeln `rsvp_opt_out` und `rsvp_require_reason` als optionale Felder im
Request-JSON. Pattern: `*int`-Pointer im Request-Struct, Nil-Check vor dem UPDATE.

```go
type updateGameReq struct {
    // ...
    RsvpOptOut        *int `json:"rsvp_opt_out,omitempty"`
    RsvpRequireReason *int `json:"rsvp_require_reason,omitempty"`
}
```

Im SQL wird COALESCE / dynamisches SET genutzt — entweder mit einem dynamisch zusammengebauten
UPDATE-Statement (so wie `UpdateMember` es heute schon macht) oder via `CASE WHEN ? IS NULL THEN
... ELSE ?`. Bevorzugt: **dynamisches Statement**, weil das schon Convention im Projekt ist.

**Alternative verworfen:** „Pflichtfeld im Request, Frontend muss immer beide Werte schicken".
Macht die API für Tooling brüchig (jeder externe Caller, der nur Datum ändern will, müsste die
RSVP-Werte erst auslesen und mitsenden). Partial-Update ist die Convention im Rest des Codes.

### Entscheidung 2 — UI-Position der zwei Checkboxen

Im `GameEditModal` werden die Checkboxen am Ende des Formulars (unter Datum/Zeit/Gegner/Type/Ort)
in einer eigenen Sektion „RSVP-Einstellungen" eingebettet — ohne Collapse, ohne eigenes Modal,
nur ein dünner Trenner darüber. Das spiegelt 1:1, wie `AdminTrainingsPage:514-536` es heute hat,
und vermeidet eine UI-Innovation.

**Alternative verworfen:** Eigener Tab im Modal. Zu viel Aufwand für zwei Boolean-Felder, und
ein einseitiges Modal liest sich besser als Tabbing zwischen zwei kleinen Sektionen.

### Entscheidung 3 — Default-Vorbelegung `rsvp_require_reason` für generische Events

Beim Anlegen eines Spiels mit `event_type='generisch'` wird `rsvp_require_reason=false`
vorbelegt — Carry-over aus bestehender Anforderung `rsvp-config-creation-ui`. Beim Bearbeiten
greift dieser Default nicht — der vorhandene DB-Wert wird gezeigt.

### Entscheidung 4 — Badge-Anzeige in der Detailansicht

In `TermineDetailPage.tsx` werden zwei kleine Status-Pills neben dem Termin-Header gerendert
(`bg-brand-info/10`, `bg-brand-yellow/20` o.ä.). Sichtbarkeit:
- `rsvp_opt_out=1` → Pill „Opt-Out aktiv"
- `rsvp_require_reason=1` → Pill „Begründung Pflicht"
- Beide `0` → keine Pill.

So bleibt der Header bei der Standardkonfiguration aufgeräumt und nur ungewöhnliche
Konfigurationen werden hervorgehoben.

**Alternative verworfen:** Immer beide Pills mit „aktiv/inaktiv". Wirkt redundant für den
Default-Fall und überfrachtet den Header.

### Entscheidung 5 — Permission ohne neue Middleware

Die drei betroffenen Endpoints sind heute schon hinter `auth.RequireRole("admin")` bzw.
`RequireClubFunction("trainer", "sportliche_leitung", "vorstand")` gegated. Keine Änderung am
Auth-Setup nötig. Test deckt nur ab: Spieler-Token bekommt 403, der Edit-Pfad funktioniert
unverändert.

## Risks / Trade-offs

**[Risiko]** Ein Trainer schaltet `rsvp_opt_out` mitten in der RSVP-Phase eines Spiels von 1 auf 0.
Alle implizit zugesagten Spieler gelten danach plötzlich als „noch nicht geantwortet". Die
Live-Update-Mechanik aktualisiert das im Browser, aber die Spieler haben evtl. schon Push-
Benachrichtigungen mit „du bist dabei" gesehen.
**→ Mitigation:** Keine technische — fachliche Entscheidung des Trainers. Falls erwünscht, könnte
das Edit-Modal beim Umschalten eine Warnung zeigen („X Spieler sind aktuell implizit zugesagt").
Im ersten Wurf bewusst NICHT umgesetzt, um Scope klein zu halten — als offene Frage notiert.

**[Risiko]** Die Detail-Badges machen sichtbar, dass Spiel 39 mit `rsvp_opt_out=0` läuft —
gleichzeitig zeigt der Kalender 19 Zusagen. Das ist genau die Inkonsistenz, die wir mit dem
Folge-Change beheben wollen, aber sie wird ab Deploy dieses Changes für den Vorstand auffälliger.
**→ Mitigation:** Folge-Change zeitnah nachschieben. Bewusst keine Vermischung von Edit-Feature
und Counter-Fix in einem Change — Diagnose-Schritt zuerst.

**[Trade-off]** Partial-Update auf Pointer-Basis macht das Request-Struct hässlicher als die
bestehenden „alle Felder Pflicht"-Patterns in `UpdateGame`. Akzeptiert, weil bestehende Clients
sonst beim Migrieren brechen würden.

**[Trade-off]** `AdminTrainingsPage` heute editiert Series und Session über teils geteilten
Code-Pfad. Das Aufheben der `disabled`-Sperre muss in beiden Fällen funktionieren — Risiko von
zwei Stellen statt einer.

## Migration Plan

1. **Backend** (`UpdateGame`, `UpdateSession`): Request-Struct + UPDATE-SQL erweitern. Tests
   ergänzen — Happy-Path, Partial-Update (Felder fehlen → DB unverändert), Permission (Spieler-
   Token → 403).
2. **Frontend** `GameEditModal`: Checkboxen + State + PUT-Payload erweitern. Default-Logik für
   `event_type='generisch'` nur bei Create.
3. **Frontend** `AdminTrainingsPage`: `disabled={!isNewSeries}` an beiden Checkboxen entfernen;
   beim PUT für Session und Series Werte mitsenden.
4. **Frontend** `TermineDetailPage`: Badge-Komponente für Spiel- und Trainings-Pfad einbauen.
5. **Lokaltest:** Spiel anlegen mit Default, im Modal nachträglich `rsvp_opt_out=1` setzen,
   Detail-Badge prüfen, Counter-Werte vergleichen (zur Folgechange-Vorbereitung dokumentieren).
6. **Deploy:** `make deploy` — keine zusätzlichen Schritte, keine Migration.

**Rollback:** Reverter-Commit reicht. Keine DB-Migration zum Rückrollen. Bereits via UI gesetzte
RSVP-Konfigurationen bleiben in der DB stehen (Default-Spalten waren schon da) — kein Schaden.

## Open Questions

1. **Warnung beim Modus-Wechsel?** Soll das Edit-Modal beim Umschalten von `rsvp_opt_out` einen
   Hinweis zeigen, dass das alle bisher impliziten Zusagen umkippt? Im ersten Wurf nein, evtl.
   später als kleines Follow-up. Nur fachlich relevant, kein Bug.
2. **Wer sieht das Detail-Badge?** Alle eingeloggten Nutzer oder nur Trainer/Vorstand? Aktueller
   Vorschlag: alle. Niemand wird durch die Information geschädigt, und so erkennt auch ein
   Spieler, dass er nicht aktiv zusagen muss. Bestätigung bei Umsetzung.
