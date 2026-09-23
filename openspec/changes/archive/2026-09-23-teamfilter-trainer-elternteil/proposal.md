## Why

`GET /api/teams` füllt den Mannschafts-Filter auf `/termine`, `/kalender` und
`/mitfahrgelegenheiten`. Für einen Nutzer mit der Vereinsfunktion `trainer` liest der
Endpoint ausschließlich `kader_trainers` — der Eltern-Pfad wird dort nicht ergänzt, sondern
**verdeckt**: wer Trainer ist und zugleich Kinder in anderen Mannschaften hat, bekommt nur
sein Trainer-Team.

Gemeldet an einem konkreten Fall (Florian Steinle, Trainer einer Mannschaft, Elternteil von
vier Kindern). Gemessen am Prod-Stand:

| | |
|---|---|
| Zugehörigkeiten (= `GET /api/teams/my`) | gF1, mA1, mA2, mB1, mB2, wC |
| Inhalt von `/termine` (Spiele) | mA1, mA2, mB1, mB2, wC |
| **Filter auf `/termine` und `/kalender`** | **nur mB2** |

Die Folge ist nicht nur eine unvollständige Auswahl. Der Filter gilt als „kein Filter",
solange nichts angehakt ist; hakt der Nutzer die eine vorhandene Mannschaft ab, verschwinden
**alle** Termine — auch die der fünf Mannschaften, für die es gar kein Kästchen gibt. Die
Liste ist dann leer, ohne dass ein Weg zurück sichtbar wäre.

Derselbe Fehler wurde für die Dienstbörse bereits behoben (`?scope=duties` bekam einen
eigenen Zweig mit der Vereinigung). Der scope-lose Zweig blieb stehen — und der damalige
Test hat ihn als „unchanged existing behavior" mit genau einem Team festgeschrieben, statt
ihn zu zeigen.

## What Changes

- **`GET /api/teams` liefert die Vereinigung aller Zugehörigkeiten**, nicht die des
  ranghöchsten Wegs. Der bestehende View `user_accessible_teams` führt genau diese Menge
  (eigener Stammkader, Stammkader der Kinder, eigene Trainer-Kader, eigener erweiterter
  Kader, erweiterter Kader der Kinder) — er wird jetzt auch für Trainer benutzt.
- **Drei Zweige werden zu einem.** Der Trainer-Zweig, der Trainer-`scope=duties`-Zweig und
  der Nicht-Trainer-Zweig beantworteten dieselbe Frage dreimal. Übrig bleibt ein Zweig mit
  einer Fallunterscheidung: `?scope=duties` → Stammkader (Selbst + Kinder) ∪ Trainer-Teams,
  sonst → `user_accessible_teams`.
- **Unverändert:** `admin`/`vorstand` (alle Teams), `sportliche_leitung` (alle Teams),
  `?scope=attendance-stats` / `?scope=diary-stats` (nur Trainer-Teams, damit der Selektor
  nicht auf ein Team zeigt, das der Statistik-Endpoint danach mit 403 abweist).

## Capabilities

### Modified Capabilities

- `api-routes`: die Anforderung „Teams-Endpoint ist rollenabhängig" beschrieb den Fehler
  wörtlich („Trainer: nur eigene Teams") und wird auf die Vereinigungs-Regel korrigiert.

## Impact

- **Code**: `internal/games/handler.go` (`ListTeamsForUser`), netto kürzer.
- **DB**: keine Migration — `user_accessible_teams` existiert und deckt alle fünf Wege ab.
- **Berechtigungen**: keine Ausweitung. Die gelieferten Mannschaften sind eine Teilmenge
  dessen, was `GET /api/teams/my` demselben Nutzer schon heute liefert; die Inhalte
  (`/games/my`, `/training-sessions`, `/duty-board`) bleiben unangetastet — sichtbar wird
  nichts Neues, nur filterbar.
- **Betroffene Oberflächen**: Filter auf `/termine`, `/kalender`,
  `/mitfahrgelegenheiten`, `/admin/trainings`.
- **CHANGELOG**: entsteht aus dem Commit-Betreff (`make build` → `scripts/gen-changelog.py`).
