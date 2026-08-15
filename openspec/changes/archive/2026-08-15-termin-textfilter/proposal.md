## Why

`/dienste`, `/termine` und `/kalender` haben je einen Team-Dropdown, Typ-Chips und einen
`past`-Toggle — und darüber hinaus keine Möglichkeit, einen Termin zu finden, von dem man
nur weiß, gegen **wen** oder **wo** er stattfindet. Wer das Auswärtsspiel in Göppingen
sucht, scrollt.

Die Felder, nach denen man tatsächlich sucht, haben alle **kein** Bedienelement:

| Feld | hat heute ein Control? |
|---|---|
| Team | ✓ Dropdown |
| Event-Typ (heim/auswärts/…) | ✓ Chips |
| Zeitraum | ✓ `past`-Toggle |
| **Gegner** | ✗ |
| **Ort / Halle** | ✗ |
| **Notiz** | ✗ |
| **Diensttyp** („Kuchen", „Kasse") | ✗ |
| **Zugewiesene Person** | ✗ |
| **konkretes Datum** | ✗ |

Ein Textfeld ist damit nicht „Suche über alles", sondern der **Rest-Filter für genau diese
Spalte** — die Felder, die aus den vorhandenen Filtern herausfallen.

Auf `/dienste` fehlt der Ort dafür sogar in der API: `boardGroup`
(`internal/duties/handler.go:707`) trägt `opponent`, `event_type`, `team_names`, `label` —
aber kein Venue. Der Handler joint die Spiele bereits (`handler.go:773`), der Ort hängt nur
einen LEFT JOIN weiter.

## What Changes

- **Ein Textfilter `q` auf `/dienste`, `/termine` und `/kalender.`** Er ist ein **Filter,
  keine Suche**: er verengt, was die Seite ohnehin lädt, und verspricht nicht, außerhalb
  des geladenen Zeitraums zu finden. Er UND-verknüpft sich mit den bestehenden Filtern und
  reiht sich an der jeweils vorhandenen Prädikatenkette ein (`DutyPage.tsx:174`,
  `TerminePage.tsx:292`, `KalenderPage.tsx:490`/`:514`).

- **Auswertungsmodell: Token-AND, Interpretation-OR.** `q` wird an Whitespace zerlegt;
  **jedes** Token muss matchen (AND), ein Token matcht, wenn **irgendeine** Interpretation
  greift (OR): Freitext gegen die Felder der Seite, oder `TT.MM.` / `TT.MM.JJJJ`, oder ein
  Monatsname. Ein Token verliert damit nie einen Treffer — `mar` matcht die Markt-Halle
  *und* März. Begründung: `design.md` §2.

- **Jahreslose Datumsangaben sind jahresunabhängig.** `14.09.` matcht (Tag 14, Monat 9) in
  jedem Jahr des geladenen Bestands, `september` matcht Monat 9 in jedem Jahr.
  `14.09.2026` ist die exakte Variante. Auf „das aktuelle Jahr" zu raten würde die halbe
  Saison stillschweigend verwerfen — die Terminliste läuft über den Jahreswechsel
  (`design.md` §3).

- **Monatsnamen als Präfix ab 3 Zeichen**, gegen die zwölf deutschen Monatsnamen.
  Normalisierung ist lowercase + Diakritika strippen (`Göppingen` findet man als
  `goppingen`). `ju` bleibt reiner Freitext statt `juni` und `juli` zu ziehen.

- **`GET /api/duty-board` liefert `venue` pro Gruppe.** Ein zusätzlicher
  `LEFT JOIN venues v ON v.id = g.venue_id`, eine Spalte, ein JSON-Feld. Gruppen ohne
  `game_id` (teamgebundene/generische Handslots) tragen einen leeren Ort — der bleibt dort
  nicht durchsuchbar. Keine Migration; GET-Route, also **kein** Broadcast-Gate betroffen.

- **Die Feldmenge ist pro Seite verschieden**, und das bleibt so. `/dienste` filtert über
  Diensttyp und zugewiesene Person, was es auf den anderen beiden Seiten nicht gibt;
  `/termine` und `/kalender` filtern über Notizen und Trainingstitel, die `/dienste` nicht
  kennt. Der Placeholder benennt pro Seite die tatsächlich durchsuchten Felder statt ein
  gemeinsames Minimum zu suggerieren (`design.md` §6).

- **Der Ausgeblendet-Zähler.** Ergibt `q` null sichtbare Treffer, während ein anderer
  Filter aktiv ist, prüft ein zweiter Durchlauf mit **nur** dem `q`-Prädikat den
  ungefilterten Bestand und meldet: „Keine Treffer. 3 Termine passen, werden aber von
  aktiven Filtern ausgeblendet. [Filter zurücksetzen]". Ohne das ist der Filter still
  unehrlich — leere Seite bei aktivem Team-Chip ist korrektes Verhalten und sieht trotzdem
  aus wie „gibt's nicht" (`design.md` §7).

- **Abwesenheiten im Kalender werden mitgefiltert** (über `member_name`, `note` und das
  Typ-Label). Nicht optional: `absencesForDay` (`KalenderPage.tsx:533`) filtert heute
  clientseitig **gar nicht**, die Abwesenheits-Balken blieben bei aktivem `q` in Zellen
  stehen, deren Termine weggefiltert sind.

- **`q` steht in den URL-Search-Params**, auf allen drei Seiten. Alle nutzen
  `useSearchParams` bereits (`KalenderPage.tsx:146` für `date`). Der Filter selbst wirkt
  sofort aus lokalem State; die URL zieht ~250 ms verzögert mit `replace: true` nach — die
  bestehende Konvention (`DutyPage.tsx:120`) — sonst schreibt jeder Tastenanschlag in die
  History-API.

- **`focus` schlägt `q`.** Beide Listenseiten lassen das fokussierte Element heute
  unconditional durch (`DutyPage.tsx:181`, `TerminePage.tsx:293`). Das bleibt: sonst
  zerreißt der Zurück-Sprung, sobald ein `q` in der URL steht.

## Nicht Teil dieses Changes

- **Keine Präfix-Syntax** (`gegner:Ludwigsburg`, `ort:…`). Gegner, Ort, Diensttyp und
  Person kollidieren praktisch nie, und in den Fällen, wo sie es tun, will man beide
  Treffer sehen. Das OR-Modell macht Präfixe zu Zucker, der sich später additiv nachrüsten
  lässt, ohne bestehende Eingaben zu brechen (`design.md` §5).
- **Kein serverseitiges `?q=`.** Filter-Semantik verspricht nur, den geladenen Bestand zu
  verengen. Damit entfällt der gesamte Komplex „Ladefenster aufziehen, `limit=500`
  reparieren, drei Endpoints um einen Suchparameter erweitern" (`design.md` §1).
- **Kein eigener Ansichtsmodus im Kalender.** Das Monatsgitter bleibt; bei aktivem `q`
  leeren sich die Zellen, genau wie beim Team-Filter heute. Ein Treffer im November ist im
  September-Gitter nicht sichtbar — dasselbe gilt für den Team-Filter und wird dort
  seit jeher nicht als Problem empfunden.
- **Keine Fuzzy-/Tippfehler-Toleranz.** Substring-Match auf normalisierten Strings, sonst
  nichts. Kein Levenshtein, kein Stemming.
- **`ü` → `ue` und Verwandte.** `Göppingen` findet man als `goppingen` (Diakritika
  gestrippt), **nicht** als `goeppingen`. Das wäre eine zweite, unabhängige
  Transliterations-Regel.
- **Ort auf game-losen Dienst-Gruppen.** Handslots ohne `game_id` haben keinen Ort in der
  Datenbank; hier wird keiner erfunden.

## Capabilities

### Added Capabilities

- **`termin-textfilter`** — das Auswertungsmodell (Token-AND/Interpretation-OR), die
  Datums- und Monatsregeln, die Feldmengen je Seite, die Komposition mit den bestehenden
  Filtern, der Ausgeblendet-Zähler und die URL-Persistenz.

### Modified Capabilities

- **`duties`** — die Spec zählt die Felder der `GET /api/duty-board`-Gruppe abschließend
  auf (`specs/duties/spec.md`, „Jede Gruppe SHALL folgende Felder enthalten"). `venue`
  kommt hinzu.
- **`termine-unified-view`** — die Spec listet die unterstützten URL-Query-Parameter
  abschließend (`team`, `types`, `past`) und schreibt fest, dass unbekannte Werte ignoriert
  werden. `q` kommt hinzu.

## Test-Anforderungen

**Routen:**

| Route | Fall | Erwartung |
|---|---|---|
| `GET /api/duty-board` | Gruppe mit `game_id`, dessen Spiel ein `venue_id` trägt | 200, Gruppe enthält `venue` mit dem `venues.name` |
| | Gruppe mit `game_id`, Spiel ohne `venue_id` | 200, `venue` fehlt bzw. ist leer — **kein** 500 |
| | game-lose Gruppe (Handslot) | 200, `venue` leer |
| | Spieler ohne privilegierte Funktion | 200, `venue` genauso vorhanden (kein neues Sichtbarkeits-Gate) |
| | ohne Token | 401 (unverändert) |

**Garantierte Invarianten** (je ein eigener Test):

1. **Der Ort öffnet keine Sichtbarkeitslücke.** Ein Nutzer, der eine Board-Gruppe heute
   nicht sieht, sieht sie mit dem neuen Feld weiterhin nicht — die Team-/Audience-/
   Saison-Klauseln bleiben Zeile für Zeile unverändert. Test: identische Gruppen-ID-Mengen
   vor/nach für Spieler, Eltern, Trainer, Vorstand.
2. **Ein Token verliert nie einen Treffer** (Vitest, `eventFilter.ts`). Für einen Termin
   mit einem Ort namens „Markthalle" **und** einen im März gilt: `mar` liefert beide. Der
   Test bricht, sobald jemand die Interpretationen exklusiv statt ODER-verknüpft macht.
3. **Jahreslose Datumsangaben sind jahresunabhängig.** Ein Bestand mit Terminen am
   14.09.2026 **und** 14.09.2027 liefert bei `14.09.` beide; bei `14.09.2026` genau einen.
   Analog `september` über zwei Jahre.
4. **Tokens sind UND-verknüpft, nicht ODER.** `sept ludwigsburg` liefert **nicht** alle
   September-Termine und **nicht** alle Ludwigsburg-Termine, sondern nur den Schnitt.
5. **Monatspräfix erst ab drei Zeichen.** `ju` matcht keinen Monat (nur Freitext), `jun`
   matcht Juni und nicht Juli, `juli` matcht Juli und nicht Juni.
6. **Diakritika werden gestrippt, nicht transliteriert.** `goppingen` findet „Göppingen",
   `goeppingen` findet es nicht (bewusst dokumentierte Grenze, siehe „Nicht Teil dieses
   Changes").
7. **`q` komponiert, es ersetzt nicht.** Ein Termin, der auf `q` passt, aber vom
   Team-Filter ausgeschlossen ist, bleibt unsichtbar — und die Seite meldet ihn im
   Ausgeblendet-Zähler mit exakter Anzahl.
8. **`focus` überlebt `q`.** Ein fokussierter Termin bleibt sichtbar, auch wenn das
   gesetzte `q` auf keines seiner Felder passt (je ein Test auf `/dienste` und `/termine`).
9. **Der leere Filter ist ein No-Op.** `q=""`, `q="   "` und ein fehlender Parameter
   liefern exakt dieselbe Menge wie heute — für alle drei Seiten. Verhindert, dass der
   Filter bei leerem Feld still etwas wegschneidet.
10. **Im Kalender bleiben keine verwaisten Abwesenheiten stehen.** Bei aktivem `q`, das auf
    keine Abwesenheit passt, enthält keine Tageszelle einen Abwesenheits-Balken.
