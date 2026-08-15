# Design — Termin-Textfilter

## 1. Filter, nicht Suche — und was dadurch entfällt

Die erste Entscheidung ist die folgenreichste, obwohl sie sich wie eine Wortklauberei
anhört. Ein **Filter** verengt, was auf dem Bildschirm ohnehin steht. Eine **Suche**
verspricht, etwas zu finden — egal wo es liegt.

```
   SUCHE (nicht gebaut)                 FILTER (gebaut)
   ════════════════════                 ═══════════════
   "finde es, egal wo"                  "zeig weniger von dem hier"
          │                                      │
          ▼                                      ▼
   Ladefenster aufziehen                 Fenster bleibt wie es ist
   limit=500 reparieren                  limit=500 bleibt wie es ist
   ?q= an 3 Endpoints                    kein Backend
   Go-Tests, Query-Bau                   ein Prädikat pro Seite
   Kalender braucht einen                Kalender-Gitter bleibt,
   Ergebnis-Modus                        Zellen leeren sich
```

Der Unterschied ist nicht kosmetisch. Alle drei Seiten laden **unterschiedlich weit**:

```
   /dienste    ├──────────────────────────►     ab heute (past=1 → alles)
   /termine    ◄────────┼───────────────────►   Saisonfenster, −365 d
   /kalender
     games     ◄──────────────────────────►     alle, limit=500
     trainings         ├────┤                   nur der sichtbare Monat
     absences          ├────┤                   nur der sichtbare Monat
```

Als **Suche** wäre diese Landschaft ein Bug-Generator: „Ludwigsburg" fände im Kalender das
Spiel im November, das Training im November aber nicht — und nichts würde das sagen. Als
**Filter** ist dieselbe Landschaft unproblematisch, weil der Geltungsbereich das ist, was
die Seite gerade zeigt. `/dienste` filtert die Dienstbörse ab heute, weil die Dienstbörse
ab heute *ist*.

Das ist keine Ausrede, sondern der Grund, warum dieser Change klein bleiben darf. Wer
später doch echte Suche will, baut sie als eigenen Change mit serverseitigem `?q=` — der
hier entstehende Parser ist dann wiederverwendbar, weil er reine Funktion ist.

## 2. Token-AND, Interpretation-OR

```
  q = "sept ludwigsburg"
       │
       ▼  split(/\s+/)
  ┌──────────┐  ┌───────────────┐
  │  "sept"  │  │ "ludwigsburg" │      ── AND ──►  jedes Token muss matchen
  └────┬─────┘  └───────┬───────┘
       │                │
       ▼                ▼
  ┌─────────────────────────────────────────┐
  │  ein Token matcht, wenn IRGENDEINE      │   ── OR ──►  über die Interpretationen
  │  Interpretation greift:                 │
  │    · Freitext gegen die Seitenfelder    │
  │    · Datum   14.09. | 14.09.2026        │
  │    · Monat   sept | september           │
  └─────────────────────────────────────────┘
```

Das OR ist der eigentliche Trick. Die naheliegende Alternative wäre, ein Token *zuerst* als
Datum zu parsen und, falls das gelingt, **nur** als Datum auszuwerten. Das ist verlockend
(klarer, deterministischer) und falsch:

- `mar` ist ein Präfix von „März" **und** ein Präfix von „Markthalle".
- `mai` ist ein Monat **und** ein plausibler Nachname.
- `03.` sieht nach Tag aus und ist womöglich eine Hallennummer.

Bei exklusiver Auswertung verschwinden Treffer, ohne dass der Nutzer eine Chance hat, das
zu bemerken — die Klasse Fehler, die nie gemeldet wird, weil eine leere Liste plausibel
aussieht. Beim OR-Modell ist der Fehler in die andere Richtung gedreht: man bekommt
gelegentlich einen Treffer zu viel, und das nächste getippte Token schneidet ihn weg. Zu
viel ist sichtbar und selbstkorrigierend, zu wenig ist unsichtbar.

Invariante 2 der Test-Anforderungen friert genau das ein.

## 3. Jahreslose Datumsangaben matchen jahresunabhängig

`14.09.` ohne Jahr ist mehrdeutig. Drei Auflösungen standen zur Wahl:

| Regel | Verhalten bei Saison 08/2026–06/2027 |
|---|---|
| aktuelles Kalenderjahr annehmen | im Januar 2027 findet `14.09.` **nichts** |
| nächstes Vorkommen ab heute | genau ein Treffer, der Rest fällt weg |
| **jahresunabhängig matchen** | alle 14.09. im geladenen Bestand |

Die Terminliste läuft strukturell über den Jahreswechsel — eine Saison beginnt im August
und endet im Juni. Beide jahresratenden Varianten verwerfen damit systematisch die Hälfte
des Bestands, und zwar still. Die dritte Variante kann nur einen Treffer *zu viel* liefern
(zwei Jahrgänge desselben Datums), und dafür gibt es die explizite Form `14.09.2026`.

Dieselbe Regel gilt für Monatsnamen: `september` matcht Monat 9, unabhängig vom Jahr.

**Umsetzungshinweis:** Der Vergleich läuft gegen `date.slice(0, 10)`, nie gegen den rohen
API-Wert. Die API liefert `"2026-09-14T00:00:00Z"` (Gotcha „SQLite DATE-Felder" in
`docs/agent/06-gotchas.md`); ein Substring-Match von `14.09.` gegen einen ISO-Timestamp
scheitert ohnehin, aber der Monats-/Tagesvergleich muss die Zeitzone strukturell meiden.

## 4. Monatspräfix ab drei Zeichen

Die zwölf deutschen Monatsnamen, normalisiert, per Präfix. Drei Zeichen ist die kleinste
Länge, bei der die Menge trennscharf wird:

```
   ju   → juni, juli        zwei Treffer  → verworfen, bleibt Freitext
   jun  → juni              eindeutig
   jul  → juli              eindeutig
   mai  → mai               drei Zeichen ist bereits der volle Name
   mar  → marz (März)       eindeutig nach Diakritika-Strip
```

Zwei Zeichen zuzulassen hieße, dass `ju` alle Juni- **und** Juli-Termine zieht — für einen
Nutzer, der gerade erst anfängt zu tippen, ein Sprung in einen völlig anderen Ergebnisraum
als beim dritten Zeichen. Ab drei Zeichen wird die Liste beim Weitertippen monoton kleiner,
was die Erwartung an ein Filterfeld ist.

**Normalisierung** ist lowercase + Unicode-NFD + Combining-Marks strippen. `März` → `marz`,
`Göppingen` → `goppingen`. Bewusst **keine** deutsche Transliteration (`ö` → `oe`): das
wäre eine zweite Regel mit eigener Fehlerklasse (`Goethe` → `gooethe`), und der Gewinn
— `goeppingen` findet „Göppingen" — ist gering, weil auf Mobile die Umlaut-Tastatur direkt
verfügbar ist.

## 5. Warum keine `gegner:`-Präfixe

Die strukturierte Variante (`gegner:Ludwigsburg ort:Scharnhausen`) wurde erwogen und
verworfen, aus drei Gründen:

1. **Die Kollisionen, die sie löst, gibt es kaum.** Gegnervereine, Hallennamen, Diensttypen
   und Personennamen überschneiden sich in der Praxis nicht. Und wo sie es tun — eine
   Halle, die nach dem Gegnerverein heißt — will man beide Treffer sehen, nicht einen
   davon wegdisambiguieren.
2. **Ohne Autocomplete ist sie auf Mobile unbenutzbar**, und Autocomplete ist mehr Arbeit
   als der gesamte Rest dieses Changes.
3. **Sie ist additiv nachrüstbar.** Da heute jedes Token als „Freitext ODER Datum" gelesen
   wird, kann ein späterer Change `feld:wert` als *weitere* Interpretation einführen, ohne
   dass eine bestehende Eingabe ihre Bedeutung ändert. Die Entscheidung ist also
   umkehrbar — was der beste Grund ist, sie jetzt nicht zu treffen.

## 6. Drei Feldmengen statt eines gemeinsamen Modells

Die Versuchung ist, ein `SearchableEvent`-Interface zu definieren, auf das alle drei Seiten
ihre Objekte abbilden. Das trägt nicht — die Seiten sind sich weniger ähnlich, als der
gemeinsame Begriff „Termin" nahelegt:

| Feld | /dienste | /termine | /kalender |
|---|---|---|---|
| Gegner | ✓ | ✓ (Game) | ✓ (Game) |
| Ort | ✓ *(neu)* | ✓ | ✓ |
| Team | ✓ | ✓ | ✓ |
| Notiz | – | ✓ | ✓ |
| Trainingstitel | – | ✓ | ✓ |
| **Diensttyp** | ✓ | – | – |
| **Zugewiesene Person** | ✓ | – | – |
| **Abwesende Person** | – | – | ✓ |

Ein gemeinsames Modell müsste entweder auf den Schnitt eindampfen (dann verliert `/dienste`
seine beiden nützlichsten Filterfelder) oder die Vereinigung mit lauter leeren Feldern
tragen (dann ist es kein Modell mehr, sondern eine Sammelstelle). Stattdessen:

```
   eventFilter.ts        parseQuery(q) → Token[]
   (pure, seitenblind)   matchesQuery(tokens, haystack: string[], dates: string[]) → boolean
          ▲
          │  je Seite ein Adapter, ~10 Zeilen
          │
   DutyPage    → [opponent, venue, ...team_names, label,
                  ...slots.map(duty_type), ...slots.flatMap(assignees.name)]
   TerminePage → [opponent, venue.name, venue.city, team_names, title, note]
   KalenderPage→ dito, plus Absence: [member_name, note, typLabel]
```

Der Parser sieht nur `string[]` und `string[]` (Daten). Er weiß nichts von Spielen,
Diensten oder Abwesenheiten — und ist damit ohne DOM, ohne Fixtures und ohne Mocks
testbar.

**Konsequenz für die UI:** der Placeholder benennt pro Seite die tatsächlich durchsuchten
Felder (`Gegner, Ort, Dienst, Person…` auf `/dienste`; `Gegner, Ort, Notiz…` auf
`/termine`). Ein einheitlicher Text würde Felder versprechen, die es dort nicht gibt.

## 7. Der Ausgeblendet-Zähler

Das ist der Preis der Filter-Semantik, und die einzige Stelle, an der dieser Change echte
UI-Arbeit hat.

```
   Nutzer tippt "ludwigsburg"      →  0 Treffer
   Team-Chip steht auf "mJC"       →  das Spiel ist mJB
   Nutzer sieht: leere Seite.
   Nutzer schließt: "gibt's nicht."
```

Bei einer Suche wäre das ein Bug. Bei einem Filter ist es **korrektes Verhalten** — und
genau deshalb gefährlich: es fühlt sich nie wie ein Fehler an und wird deshalb nie
gemeldet.

```
┌──────────────────────────────────────────────────────┐
│ [🔍 ludwigsburg                                  ×]  │
├──────────────────────────────────────────────────────┤
│  Keine Treffer.                                      │
│  3 Termine passen, werden aber von aktiven           │
│  Filtern ausgeblendet.        [Filter zurücksetzen]  │
└──────────────────────────────────────────────────────┘
```

Die Umsetzung ist ein zweiter `filter()`-Durchlauf über das **ungefilterte** Array mit
ausschließlich dem `q`-Prädikat. Bei ≤ 500 Elementen ist das rechnerisch nichts, und er
läuft nur, wenn die sichtbare Menge leer und `q` nicht leer ist.

Der Zähler erscheint **nur**, wenn mindestens ein anderer Filter aktiv ist. Ohne aktiven
Nebenfilter ist „keine Treffer" vollständig erklärt und ein Zusatz wäre Lärm.

## 8. Reihenfolge in der Prädikatenkette

`focus` bleibt oben, unverändert:

```
  ┌─ groupMatchesFocus / focus-Match ──────────► DURCH, unbedingt
  │
  ├─ past
  ├─ types.has
  ├─ team
  │
  └─ q          ← neu, ganz unten
```

`focus` markiert das Element, zu dem der Zurück-Sprung scrollen soll
(`zurueck-navigation-restore`). Stünde `q` darüber, würde ein in der URL stehendes `q` das
Sprungziel wegfiltern und die Wiederherstellung der Scroll-Position stillschweigend
brechen — ausgerechnet in dem Fall, in dem der Nutzer über einen geteilten Link mit Filter
gekommen ist.

`q` steht ganz unten, weil es das teuerste Prädikat ist (String-Normalisierung über mehrere
Felder). Die billigen Set-Lookups schneiden vorher weg.

## 9. URL-Sync verzögert, Filterung sofort

```
   Tastenanschlag ──► lokaler State ──► Filterung   (sofort, jeder Anschlag)
                            │
                            └──► ~250 ms ──► setSearchParams(replace: true)
```

Zwei Gründe für die Entkopplung: die History-API verträgt kein Schreiben pro
Tastenanschlag, und `useSearchParams` löst einen Router-Re-Render aus, den die Liste bei
jedem Zeichen nicht braucht. `replace: true` ist die bestehende Konvention
(`DutyPage.tsx:120`) — ohne sie würde jedes getippte Zeichen einen History-Eintrag legen
und den Zurück-Button unbrauchbar machen.

Beim Mount läuft es andersherum: `q` aus der URL initialisiert den lokalen State, damit ein
geteilter Link sofort gefiltert öffnet.

## 10. Der Ort auf `/duty-board`

```sql
 FROM duty_slots ds
 JOIN      duty_types dt        ON dt.id = ds.duty_type_id
 LEFT JOIN duty_assignments da  ON da.duty_slot_id = ds.id AND da.user_id = ?
 LEFT JOIN games g              ON g.id = ds.game_id        ← existiert bereits
 LEFT JOIN teams t              ON t.id = ds.team_id
+LEFT JOIN venues v             ON v.id = g.venue_id        ← neu
```

Plus `COALESCE(v.name, '')` in der Projektion und ein `Venue string \`json:"venue,omitempty"\``
auf `boardGroup`. `games.venue_id → venues(id) ON DELETE SET NULL` existiert seit
`001_initial.up.sql:206`.

Drei Eigenschaften, die diesen Eingriff harmlos machen:

- **Kein neues Sichtbarkeits-Gate.** Der Ort hängt am Spiel, das die Gruppe ohnehin
  benennt (Gegner, Datum, Uhrzeit sind bereits drin). Wer die Gruppe sieht, darf den Ort
  sehen. Der `whereParts`-Block bleibt Zeile für Zeile unangetastet — Invariante 1 friert
  das ein.
- **Kein neuer Nullwert-Pfad.** Beide LEFT JOINs können leer laufen (Slot ohne Spiel, Spiel
  ohne Venue); `COALESCE` fängt beides auf denselben leeren String, den `opponent` und
  `event_type` schon heute liefern.
- **`omitempty`** hält die Payload für game-lose Gruppen unverändert groß — relevant auf
  1 GB VPS und für die `payload-measurement`-Capability.

Nur der **Name** der Halle wandert mit, nicht Straße/Stadt/PLZ. Auf `/termine` und
`/kalender` ist `venue.city` mit durchsuchbar (die Objekte tragen es ohnehin); auf
`/dienste` wäre es eine Payload-Vergrößerung für einen selten getippten Suchbegriff. Die
Asymmetrie ist bewusst und im Placeholder nicht sichtbar — beide sagen „Ort".
