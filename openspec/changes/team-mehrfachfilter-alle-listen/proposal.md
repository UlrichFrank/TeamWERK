## Why

`termine-team-mehrfachfilter` hat den Mannschafts-Filter auf `/termine` zur
Mehrfachauswahl gemacht (Dropdown mit Checkboxen, `team` als ID-Liste, leere
Auswahl = kein Filter). Drei weitere Listen stellen dieselbe Frage und
beantworten sie noch jede anders:

- **`/kalender`** — `<select>` mit einer Mannschaft, im lokalen State, auf Mobile
  gar nicht sichtbar (`hidden sm:block`).
- **`/dienste`** — `<select>` mit einer Mannschaft, `team` als einzelne ID in der
  URL, auf Mobile ausgeblendet. Zusätzlich derselbe Fokus-Fehler, den
  `termine-team-mehrfachfilter` auf `/termine` behoben hat: `?focus=game-<id>`
  (aus dem Kalender-Modal, `EventInfoModal.tsx`) überlebt eine Filteränderung
  und zeigt danach eine Gruppe, die der Filter ausschließt.
- **`/mitfahrten`** — `<select>` mit einer Mannschaft, die pro Filterklick eine
  neue Anfrage auslöst (`?team_id=`).

Ein Elternteil mit Kindern in zwei Mannschaften — der Normalfall dieser vier
Seiten — kann also auf keiner davon „mA2 und mC2, aber nicht den Rest" sehen.
Der Zustand ist außerdem teuer geworden: vier Kopien derselben Filterlogik, die
bereits vierfach verschieden ist (mal URL, mal State, mal Server, mal Client).

## What Changes

- **Alle vier Listen nutzen denselben `TeamFilter`** (Dropdown mit Checkboxen,
  Icon + Zähler im Compact-Modus) und dieselbe Semantik: `team` ist eine
  ID-Liste, die leere Auswahl heißt „kein Filter" und zeigt alle Kästchen
  angehakt, das Abwählen der letzten Mannschaft fällt dorthin zurück.
- **Die Semantik lebt an einer Stelle** — `web/src/lib/teamFilter.ts` (Parsen,
  Serialisieren, Umschalten, Filterprädikat, Optionen). `/termine` wird
  verhaltensgleich darauf umgestellt, die drei anderen Seiten kommen dazu.
- **Der Filter ist überall auf Mobile bedienbar.** Als Icon-Button mit Zähler
  braucht er den Platz nicht mehr, für den das `<select>` ausgeblendet war.
- **`/dienste` beendet den Fokus bei aktiver Team-/Typ-Filteränderung** — dieselbe
  Regel wie auf `/termine`.
- **`/mitfahrten` filtert clientseitig**, statt pro Filterklick neu zu laden: die
  Antwort trägt `teamIds` je Spiel, die ungefilterte Menge ist ohnehin die
  geladene. Nebenbei behoben: der serverseitige Filter (`gt.team_id = ?`) verkürzte
  über `GROUP_CONCAT` auch die **angezeigte** Mannschaftsliste eines
  Mehr-Team-Spiels auf die gefilterte. `GET /api/mitfahrgelegenheiten?team_id=`
  bleibt unverändert bestehen (Bestandslinks, andere Aufrufer).
- **`GET /api/absences/calendar` nimmt `team_id` als ID-Liste** (`team_id=3,7`).
  Die Abwesenheiten sind die einzige Datenquelle des Kalenders, die serverseitig
  gefiltert wird — ohne diese Erweiterung wäre der Kalender-Filter für sie nicht
  ausdrückbar. Die Einzel-ID bleibt gültig, die beiden bisherigen Query-Varianten
  (mit/ohne Filter) werden zu einer mit optionaler Zusatzbedingung.

## Capabilities

### Modified Capabilities

- `duty-board-team-filter`: Der Filter nimmt mehrere Mannschaften; der
  Deep-Link-Fokus endet bei aktiver Filteränderung.
- `mitfahrgelegenheiten-team-filter`: Mehrfachauswahl per Checkboxen, clientseitig
  statt per `?team_id=`.
- `team-absences-calendar`: `team_id` ist eine ID-Liste.

`termine-unified-view` bleibt unberührt — `/termine` ändert sein Verhalten nicht,
nur seine Fundstelle (Umzug auf `lib/teamFilter.ts`).

## Impact

- `web/src/lib/teamFilter.ts` (neu) — die gemeinsame Semantik
- `web/src/components/TeamFilter.tsx` — `TeamFilterOption` zieht in die lib
- `web/src/pages/TerminePage.tsx` — verhaltensgleich auf die lib umgestellt
- `web/src/pages/KalenderPage.tsx`, `DutyPage.tsx`, `MitfahrgelegenheitenPage.tsx`
- `internal/absences/handler.go` — `team_id` als Liste (`parseTeamIDList`,
  `placeholders`), zwei Query-Varianten zu einer zusammengeführt
- Keine Migration, keine neue Route.

## Test-Anforderungen

| Fläche | Test | Erwartung |
|---|---|---|
| `lib/teamFilter` | `parseTeamIds` / `serializeTeamIds` / `matchesTeamFilter` | Einzel-ID und Liste, leere = vollständige Auswahl = kein Filter, Überschneidung genügt |
| `/dienste?team=1,2` | `team=1,2 zeigt beide Mannschaften` | dritte Mannschaft bleibt aus |
| `/dienste` | `Abwählen aller Mannschaften ist kein Filter` | `team` verschwindet aus der URL, alles sichtbar |
| `/dienste?focus=game-2` | `Team-Filteränderung beendet den Fokus` | `focus` fällt, die Gruppe verschwindet |
| `/mitfahrten` | `Abwählen einer Mannschaft filtert ohne erneutes Laden` | keine zusätzliche `/mitfahrgelegenheiten`-Anfrage |
| `/mitfahrten` | `die Liste wird ohne team_id-Parameter geladen` | die Seite nutzt den Server-Filter nicht mehr |
| `/kalender` | `Abwählen einer Mannschaft blendet nur deren Termine aus` | Gitter folgt der Mehrfachauswahl |
| `/kalender` | `Abwesenheiten werden mit der ID-Liste nachgeladen` | `team_id=1,2` in der Anfrage |
| `GET /api/absences/calendar` | `TestCalendar_TeamIDListe` | beide gewählten Mannschaften, die dritte nicht |
| `GET /api/absences/calendar` | `TestCalendar_TeamIDEinzeln` | Bestandsverhalten bleibt |
| `GET /api/absences/calendar` | `TestCalendar_TeamIDUnbrauchbar` | wirkt wie kein Filter, kein Fehler |

**Garantierte Invariante:** Die vier Listen beantworten „welche Mannschaften sehe
ich?" identisch — dieselbe Menge, dieselbe Deselektions-Regel, dieselbe
Fundstelle. Und: ein Termin, den der Nutzer nach einer selbst vorgenommenen
Filteränderung sieht, erfüllt den Filter.
