## Why

`POST /api/training-sessions/{id}/respond` prüft bei der Zusage **für sich selbst** an
keiner Stelle, ob der Antwortende zum Kader des Termins gehört. Jeder eingeloggte Nutzer
mit Mitglieds-Datensatz kann eine beliebige Session-ID ansprechen; die Zeile wird
gespeichert (HTTP 204) und zählt in `confirmed_count` mit — der Trainer sieht eine Zusage
von jemandem, der nicht in seiner Mannschaft ist. Nachgemessen im Zuge von
`uebungsgruppen`, nicht abgeleitet.

Der Kontrast macht die Lücke sichtbar: das Antworten für **andere** (`member_id` gesetzt)
ist sauber abgesichert — Eltern nur für ihre Kinder, sonst nur Staff. Genau dieser
Kommentar steht seit dem damaligen Fix im Code („broken access control"). Die Selbst-RSVP
blieb dabei ungeprüft, weil sie als harmlos galt: wer die Session-ID nicht kennt, kann sie
nicht ansprechen. Das ist Security-by-Obscurity — IDs sind fortlaufend.

Der Change `uebungsgruppen` hat die Lücke **nicht verursacht** (sie trifft Mannschaften
identisch), aber sichtbar gemacht: sein Spec-Szenario „Fremder darf nicht antworten → 403"
war nicht erfüllbar, ohne das Verhalten der Mannschaftsvariante mitzuändern. Es wurde
deshalb dort herausgenommen und hierher verlagert
(`uebungsgruppen/design.md — Bekannter Rest`).

## What Changes

- **`trainings.Respond` prüft die Kaderzugehörigkeit auch für die Selbst-RSVP.** Wer weder
  im Stammkader noch im erweiterten Kader noch unter den Trainern des Termin-Kaders steht,
  bekommt **HTTP 403**. Die Prüfung läuft über `kader_id` und gilt dadurch für
  Mannschaften und Übungsgruppen über denselben Pfad.
- **Die vorhandene Beteiligten-Logik wird wiederverwendet, nicht nachgebaut.**
  `ListSessions` beantwortet dieselbe Frage bereits als `am_i_participant`
  (`kader_members` ∪ `kader_extended_members` ∪ `kader_trainers` gegen `ts.kader_id`).
  Diese drei Zweige wandern in einen Helfer, den Lese- und Schreibpfad teilen — sonst
  entstünde eine zweite Antwort auf „gehört diese Person zum Termin?", die auseinander
  driften kann.
- **Das Antworten für andere bleibt unverändert.** Der Eltern-/Staff-Zweig ist bereits
  korrekt; er bekommt zusätzlich dieselbe Kaderprüfung für das **Ziel**-Mitglied.
- **Bestandstests, die sich auf die Lücke stützen, werden angepasst** — sie dokumentieren
  kein gewolltes Verhalten, sondern haben es nur nicht geprüft.
- **Der Sichtbarkeitsfilter von `ListSessions` verlangt für den Trainer-Zweig nicht mehr
  zusätzlich die Vereinsfunktion `trainer`.** Nachgemessen beim Bau des
  Deckungsgleichheits-Tests, nicht vorher bekannt (`design.md — Entscheidung 4`).

## Impact

**Backend**

- `internal/trainings/handler.go` — `Respond` (Selbst- und Fremd-Zweig), neuer Helfer
  `isKaderParticipant(ctx, kaderID, memberID)`; `ListSessions` nutzt denselben Helfer für
  `am_i_participant`, soweit ohne SQL-Umbau möglich.
- `internal/trainings/handler.go` — `ListSessions`, Sichtbarkeitsfilter: der
  `kader_trainers`-Zweig gilt unabhängig von der Vereinsfunktion `trainer`.

**Tests**

- Bestand: `TestRespond_CreatesRSVP` und `TestRespond_UpdatesExistingRSVP` legen ihr
  Mitglied heute in **keinen** Kader und erwarten 204. Sie bekommen die
  Kader-Zugehörigkeit, die sie fachlich immer gemeint haben — die Assertion bleibt.
- Neu: siehe Test-Anforderungen.

**Kein Schema, keine Migration, kein Frontend.** Die Oberfläche bietet die Zusage ohnehin
nur auf Terminen an, die der Nutzer sieht; ein 403 ist dort nicht erreichbar.

**Verhaltensänderung (bewusst):** eine bisher akzeptierte Anfrage wird abgelehnt. Betroffen
sind ausschließlich Anfragen, die schon heute fachlich unsinnig sind. Bestehende
`training_responses`-Zeilen von Nicht-Kadermitgliedern werden **nicht** bereinigt — das
wäre ein Datenlöschlauf ohne Not; sie verschwinden mit der normalen Saison-Historie.

## Capabilities

### Modified Capabilities

- `trainings`: Die RSVP-Berechtigung wird für **jede** Antwort über `kader_id` aufgelöst —
  auch für die Antwort auf den eigenen Namen.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/training-sessions/{id}/respond` | `TestRespond_FremderAbgelehnt` | 403; keine Zeile in `training_responses` |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_StammkaderErlaubt` | 204, Zeile gespeichert |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_ErweiterterKaderErlaubt` | 204 — der erweiterte Kader ist Beteiligter, nicht Fremder |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_TrainerErlaubt` | 204 |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_UebungsgruppeFremderAbgelehnt` | 403 — der Fall, der `uebungsgruppen` nicht abschließen konnte |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_ElternteilFuerFremdesKindAbgelehnt` | 403, unverändertes Bestandsverhalten |
| `POST /api/training-sessions/{id}/respond` | `TestRespond_StaffFuerNichtKadermitgliedAbgelehnt` | 403 — auch Vorstand/Trainer setzen keine Zusage für jemanden, der nicht zum Termin gehört |
| `GET /api/training-sessions` | bestehende `am_i_participant`-Tests | bleiben grün — der Helfer ändert die Lesesicht nicht |
| `GET /api/training-sessions` | `TestRespond_AnzeigeUndAntwortrechtStimmenUeberein` | für alle drei Beteiligungszweige und den Fremden sagen `am_i_participant` und der Ausgang der Antwort dasselbe |
| `GET /api/training-sessions` | `TestListSessions_KaderTrainerOhneVereinsfunktion` | der Termin ist gelistet mit `am_i_participant: true` — die Eintragung in `kader_trainers` genügt |

**Garantierte Invarianten:**

1. **Eine `training_responses`-Zeile existiert nur für Beteiligte des Termins.** Beteiligt
   ist, wer im Stammkader, im erweiterten Kader oder unter den Trainern des `kader_id`
   steht.
2. **Lese- und Schreibpfad geben dieselbe Antwort.** „Darf ich antworten?" (Respond) und
   „bin ich beteiligt?" (`am_i_participant`) werden von **einer** Funktion beantwortet;
   eine Anzeige ohne Antwortmöglichkeit oder umgekehrt ist damit ausgeschlossen.
3. **Der Eltern-/Staff-Zweig wird nicht gelockert.** Die neue Prüfung kommt additiv hinzu.
