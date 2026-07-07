# Design — payload-measurement-harness

## Warum ein Go-Integrationstest (Weg A), nicht Access-Logs oder Lighthouse

| Weg | Reproduzierbar | CI-tauglich | Reale Zahlen | Aufwand |
|---|---|---|---|---|
| **A: Go-Test + testutil.NewServer** | ja (fixer Seed) | ja | synthetisch | niedrig |
| B: `bytes_out` im Access-Log | nein (Traffic-abhängig) | nur mit Prod-Daten | ja | mittel |
| C: DevTools/Lighthouse | manuell | nein | ja (Einzelmessung) | niedrig, aber nicht wiederholbar |

Weg A gewinnt für den Zweck „belastbarer Vorher/Nachher-Vergleich der Optimierungen", weil deterministisch und im selben Lauf wie die Tests. B bleibt der ergänzende Weg für echte Produktions-Kennzahlen (separater Change, an `production-monitoring` andockend).

## Deterministisches Seeding (verbindlich)

Die Größen sind **festgezurrt** (eine zentrale Konstante `measureSeed`), nicht beispielhaft — sonst driften Baseline und Nachher-Läufe. Änderungen an diesen Zahlen invalidieren die Baseline und sind ein bewusster, eigener Commit.

```
Teams:            T1, T2, T3, T4                         (4)
Seasons:          3   (S_curr aktiv, S_prev, S_next)
Members:          200 gesamt, feste Verteilung:
                    - je Team 45 spieler                 (180)
                    - je Team 1 trainer                  (4)
                    - 2 sportliche_leitung (T1, T2)      (2)
                    - 3 vorstand, 1 vorstand_beisitzer   (4)
                    - 2 kassierer                        (2)
                    - 8 elternteil (je Kind in T1..T4)   (8)
Games:            100  (S_curr; 60 vergangen, 40 künftig; je Spiel 1–2 game_teams)
DutyTypes:        20   (10 davon instruction_md = fixer 3 072-Byte-Lorem-Block)
DutySlots:        500  (verteilt über die 100 Games, je Slot 0–3 Assignees mit photo_url)
TrainingSessions: 100  (60 seriengebunden, 40 standalone → deckt exclude_series ab)
ChatMessages:     100  in 1 Konversation (80 kurz ~40 B, 15 lang ~2 KB, 5 gelöscht)
```

- **Kein** Zufall, **kein** `time.Now()` im Datensatz: alle DATE-Felder relativ zu einer festen Referenzzeit-Konstante `measureRefTime` (z. B. Mitte von `S_curr`). „Vergangen/künftig" sind relativ zu `measureRefTime`, nicht zur Wanduhr → zwei Läufe byte-identisch.
- Der `instruction_md`-Block ist ein **fixer** 3 072-Byte-String (Konstante), damit der `duty-types`-Payload-Delta exakt bezifferbar ist (`#1`: 10 × 3 072 B fallen aus der Liste).
- Report-Kopfzeile darf einen Zeitstempel + `git rev-parse HEAD` tragen (Kosmetik, außerhalb der Assertions).

## Was gemessen wird

**1. Payload pro Route** — GET via `httptest`, `len(body)` + Status:
```
/api/kader, /api/duty-slots, /api/games, /api/games/{id}/participants,
/api/training-sessions, /api/duty-board, /api/duty-types,
/api/chat/conversations/{id}/messages,
/api/teams, /api/seasons, /api/venues, /api/encryption-pubkey,
/api/push/vapid-public-key, /api/age-class-rules
```

**2. 304/Cache** — jede Referenzroute zweimal; zweiter Call mit `If-None-Match` des ersten `ETag`. Erfasst Status + Bytes des zweiten Calls. Auf `main`: 200 + volle Bytes (kein ETag). Nach `reference-data-caching`: 304 + ~0 Bytes.

**3. SSE-Fan-out pro Mutation** — Kern für `scoped-live-updates`. Der Empfängerkreis muss **exakt** determiniert sein, sonst ist das Nachher unbezifferbar. Deshalb ein **fester Satz von M = 8 benannten Clients**, jeder ein konkretes geseedetes Member mit bekannter Funktion/Team:

| Client | Funktion | Team | `members`-Audience | `games(T1)`-Audience | `settings` |
|---|---|---|---|---|---|
| C1 | admin | — | ✔ | ✔ | ✔ |
| C2 | vorstand | — | ✔ | ✔ | ✔ |
| C3 | kassierer | — | ✔ | ✘ | ✔ |
| C4 | trainer | T1 | ✘ | ✔ | ✔ |
| C5 | spieler | T1 | ✘ | ✔ | ✔ |
| C6 | spieler | T2 | ✘ | ✘ | ✔ |
| C7 | spieler | T3 | ✘ | ✘ | ✔ |
| C8 | elternteil (Kind in T1) | T1 | ✘ | ✔ | ✔ |
| **Σ berechtigt** | | | **3** | **5** | **8** |

```
Ablauf je Mutation:
  8 httptest-Clients (C1..C8) abonnieren /api/events
  1 Mutation auslösen; Zustellfenster (z. B. 500 ms) abwarten
  je Client gelieferte Events/Bytes zählen → Σ + Verteilung
```

- **Gemessene Mutationen (fest):**
  - `members`: `PUT /api/members/{C5.member_id}` (Statuswechsel) → Topic `members`.
  - `games(T1)`: `PUT /api/games/{ein T1-Spiel}` → Topic `games`.
  - `settings` (Kontrolle): `PUT /api/club` → Topic `settings`.
- **Auf `main` (globaler `Broadcast`):** jede der drei Mutationen stellt an **alle 8** Clients zu → Baseline `8 / 8 / 8`.
- **Nach `scoped-live-updates`:** `members → 3`, `games(T1) → 5`, `settings → 8` (bleibt global). Genau diese Ganzzahlen sind der Wirkungsnachweis; `kassierer` (C3) im `games`-Fall und die teamfremden Spieler (C6/C7) sind die aussagekräftigen „sollte-nicht-mehr-empfangen"-Fälle.
- **`kassierer` ≠ Team:** C3 ist bei `members` drin (Finance liest Mitglieder), bei `games(T1)` aber draußen — dieser Kontrast prüft, dass Rollen- und Team-Scoping nicht verwechselt werden.
- **Nicht hier gemessen:** `live-update-coalescing` (Frontend) ändert nicht den Server-Fan-out, sondern die **Client-Reload-Zahl** bei Bursts — das ist ein Frontend-Test in `efficient-data-loading-quickwins`, nicht Teil dieses Server-Werkzeugs.

## Report-Format (`metrics/PAYLOAD.md`)

```
# Payload-Messung (baseline: <git-sha>)

## Payload pro Route
| Route | Status | Bytes |
|---|---|---|
| GET /api/duty-board | 200 | 187341 |
| …

## Referenzdaten-Revalidierung
| Route | 1. Call | 2. Call (If-None-Match) |
|---|---|---|
| GET /api/seasons | 200 / 4123 B | 200 / 4123 B   ← main: kein 304

## SSE-Fan-out pro Mutation (M=5 Clients)
| Mutation | zugestellte Events | Σ Bytes |
|---|---|---|
| POST /api/members/… | 5 | …   ← main: global
```

`metrics/payload-baseline.md` = committete Kopie dieses Reports auf `main`. `.gitignore` bekommt `metrics/PAYLOAD.md` (der Lauf-Output), die Baseline bleibt versioniert.

## Architektur-Test

`internal/measure` importiert `internal/app` (Router) + `internal/testutil`. Das ist ein Composition-/Test-Support-Package — in `arch_test.go` entsprechend klassifizieren, damit die „Domain importiert nicht Domain"-Regel nicht getriggert wird (es ist kein Domain-Package).

## Nicht-Ziele

- Kein Micro-Benchmark der Latenz/CPU (das deckt ggf. `go test -bench` separat ab).
- Keine Produktions-Traffic-Messung (Weg B, eigener Change).
- Kein Einhängen in `pre-push` — die Fan-out-Messung mit Timing-Fenster ist potenziell flaky; nur freiwilliges `make measure`/`measure-gate`.
