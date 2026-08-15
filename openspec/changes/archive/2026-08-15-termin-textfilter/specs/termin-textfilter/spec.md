## ADDED Requirements

### Requirement: Textfilter auf Termin-Ansichten

Die Seiten `/dienste`, `/termine` und `/kalender` SHALL je ein Textfeld anbieten, dessen
Inhalt (`q`) als **zusätzlicher Filter** auf die bereits geladene Termin-Menge wirkt. Der
Filter SHALL sich mit den bestehenden Filtern (Team, Event-Typ, `past`, `mine`, `audience`)
konjunktiv verbinden: ein Element ist genau dann sichtbar, wenn es **alle** aktiven Filter
erfüllt.

Der Filter SHALL **keine** Daten nachladen und **keinen** serverseitigen Query-Parameter
erzeugen. Sein Geltungsbereich ist ausschließlich der Bestand, den die Seite ohne ihn
anzeigen würde.

Ein leerer, ausschließlich aus Whitespace bestehender oder fehlender `q`-Wert SHALL die
sichtbare Menge unverändert lassen.

#### Scenario: Filter verengt die Liste
- **WHEN** ein Nutzer auf `/termine` `q = "ludwigsburg"` setzt
- **THEN** bleiben genau die Termine sichtbar, bei denen mindestens ein durchsuchtes Feld den Wert enthält

#### Scenario: Leerer Filter ist ein No-Op
- **WHEN** `q` fehlt, leer ist oder nur Leerzeichen enthält
- **THEN** ist die sichtbare Menge identisch zu der ohne Textfilter

#### Scenario: Filter komponiert mit dem Team-Filter
- **WHEN** ein Nutzer nach Team B filtert und zusätzlich `q = "ludwigsburg"` setzt
- **THEN** bleibt ein Termin gegen Ludwigsburg, der Team A zugeordnet ist, ausgeblendet

#### Scenario: Kein serverseitiger Nachladevorgang
- **WHEN** ein Nutzer `q` setzt oder ändert
- **THEN** wird kein zusätzlicher API-Request ausgelöst und kein Datumsfenster erweitert

### Requirement: Auswertungsmodell Token-AND / Interpretation-OR

Der Filterausdruck SHALL an Whitespace in Tokens zerlegt werden. Ein Element ist genau dann
ein Treffer, wenn **jedes** Token matcht (konjunktiv).

Ein einzelnes Token SHALL matchen, wenn **mindestens eine** der folgenden Interpretationen
zutrifft (disjunktiv) — die Interpretationen SHALL NOT exklusiv ausgewertet werden:

1. **Freitext:** das normalisierte Token ist Teilstring eines der normalisierten
   durchsuchbaren Felder des Elements.
2. **Datum:** das Token hat die Form `TT.MM.` oder `TT.MM.JJJJ` und passt auf das Datum des
   Elements.
3. **Monat:** das Token ist ein Präfix von mindestens drei Zeichen eines der zwölf
   deutschen Monatsnamen und passt auf den Monat des Elements.

Normalisierung SHALL lowercase plus Entfernen von Unicode-Combining-Marks (NFD) sein.
Es SHALL keine deutsche Transliteration (`ö` → `oe`) stattfinden und keine
Tippfehler-Toleranz (Fuzzy-Matching, Stemming).

#### Scenario: Tokens sind konjunktiv
- **WHEN** `q = "sept ludwigsburg"` gesetzt ist
- **THEN** bleiben nur Termine sichtbar, die im September liegen **und** Ludwigsburg im Freitext tragen
- **AND** nicht alle September-Termine und nicht alle Ludwigsburg-Termine

#### Scenario: Ein Token verliert keinen Treffer
- **WHEN** `q = "mar"` gesetzt ist, ein Termin im März liegt und ein zweiter Termin in einer Halle namens „Markthalle" stattfindet
- **THEN** sind **beide** Termine sichtbar

#### Scenario: Diakritika werden gestrippt
- **WHEN** `q = "goppingen"` gesetzt ist und ein Termin den Ort „Göppingen" trägt
- **THEN** ist der Termin sichtbar

#### Scenario: Keine Transliteration
- **WHEN** `q = "goeppingen"` gesetzt ist und ein Termin den Ort „Göppingen" trägt
- **THEN** ist der Termin **nicht** sichtbar

### Requirement: Datums- und Monatsauswertung

Ein Datums-Token **ohne** Jahresangabe (`TT.MM.`) SHALL jahresunabhängig matchen: es trifft
jedes Element, dessen Datum denselben Tag und Monat trägt, unabhängig vom Jahr. Ein
Datums-Token **mit** Jahresangabe (`TT.MM.JJJJ`) SHALL exakt matchen.

Ein Monats-Token SHALL ebenfalls jahresunabhängig matchen. Ein Präfix von weniger als drei
Zeichen SHALL NOT als Monat ausgewertet werden.

Der Datumsvergleich SHALL gegen `date.slice(0, 10)` erfolgen, nie gegen den rohen
API-Wert — die API liefert Datumsfelder als ISO-Timestamp (`"2026-09-14T00:00:00Z"`).

#### Scenario: Jahresloses Datum trifft mehrere Jahrgänge
- **WHEN** `q = "14.09."` gesetzt ist und der geladene Bestand Termine am 14.09.2026 und am 14.09.2027 enthält
- **THEN** sind beide sichtbar

#### Scenario: Datum mit Jahr trifft exakt
- **WHEN** `q = "14.09.2026"` gesetzt ist und der geladene Bestand Termine am 14.09.2026 und am 14.09.2027 enthält
- **THEN** ist nur der Termin von 2026 sichtbar

#### Scenario: Monatsname trifft jahresübergreifend
- **WHEN** `q = "september"` gesetzt ist und der Bestand September-Termine aus zwei Saisonjahren enthält
- **THEN** sind alle sichtbar

#### Scenario: Monatspräfix ab drei Zeichen
- **WHEN** `q = "jun"` gesetzt ist
- **THEN** matchen Juni-Termine über die Monats-Interpretation und Juli-Termine nicht

#### Scenario: Zwei Zeichen sind kein Monat
- **WHEN** `q = "ju"` gesetzt ist
- **THEN** wird das Token ausschließlich als Freitext ausgewertet und zieht weder Juni- noch Juli-Termine über die Monatsregel

### Requirement: Durchsuchte Felder je Seite

Die Menge der durchsuchten Felder SHALL sich pro Seite nach den dort verfügbaren Daten
richten. Es SHALL kein seitenübergreifendes Einheitsmodell erzwungen werden.

| Seite | Durchsuchte Felder |
|---|---|
| `/dienste` | `opponent`, `venue`, `team_names[]`, `label`, je Slot `duty_type`, `role_desc` und `assignees[].name` |
| `/termine` | `opponent`, `venue.name`, `venue.city`, Team-Anzeigenamen, Trainingstitel, `note` |
| `/kalender` | wie `/termine`, zusätzlich je Abwesenheit `member_name`, `note` und das Typ-Label |

Das Eingabefeld SHALL einen Placeholder tragen, der die auf **dieser** Seite tatsächlich
durchsuchten Felder benennt.

#### Scenario: Diensttyp ist auf /dienste filterbar
- **WHEN** ein Nutzer auf `/dienste` `q = "kuchen"` setzt
- **THEN** bleiben die Gruppen sichtbar, die einen Slot mit dem Diensttyp „Kuchendienst" enthalten

#### Scenario: Zugewiesene Person ist auf /dienste filterbar
- **WHEN** ein Nutzer auf `/dienste` `q = "müller"` setzt
- **THEN** bleiben die Gruppen sichtbar, in denen eine Person dieses Namens einem Slot zugewiesen ist

#### Scenario: Abwesenheiten werden im Kalender mitgefiltert
- **WHEN** ein Nutzer auf `/kalender` ein `q` setzt, das auf keine Abwesenheit passt
- **THEN** enthält keine Tageszelle einen Abwesenheits-Balken

### Requirement: Ausgeblendete Treffer werden ausgewiesen

Ergibt ein nicht-leeres `q` **null** sichtbare Elemente, während mindestens ein weiterer
Filter aktiv ist, SHALL die Seite die Anzahl der Elemente ausweisen, die `q` erfüllen und
ausschließlich durch die übrigen Filter ausgeblendet werden, und eine Aktion zum
Zurücksetzen dieser Filter anbieten.

Ist **kein** weiterer Filter aktiv, SHALL dieser Hinweis nicht erscheinen.

#### Scenario: Nulltreffer durch Team-Filter wird erklärt
- **WHEN** ein Nutzer nach Team B filtert und `q = "ludwigsburg"` setzt, während drei passende Termine anderen Teams zugeordnet sind
- **THEN** meldet die Seite „keine Treffer" zusammen mit der Anzahl 3 und einer Aktion „Filter zurücksetzen"

#### Scenario: Echter Nulltreffer bleibt schlicht
- **WHEN** kein weiterer Filter aktiv ist und `q` auf kein Element passt
- **THEN** erscheint die normale Leermeldung ohne Ausgeblendet-Hinweis

### Requirement: Persistenz in URL-Search-Params

Die Seiten SHALL `q` in den URL-Search-Params ablegen, sodass der Filterzustand teilbar und
per Browser-Back/Forward navigierbar ist. Beim Mount SHALL `q` aus der URL gelesen und die
Liste unmittelbar gefiltert dargestellt werden.

Der Filter SHALL bei jedem Tastenanschlag sofort wirken; die Aktualisierung der URL SHALL
verzögert (~250 ms) und via Replace — nicht Push — erfolgen, damit keine History-Einträge
pro Zeichen entstehen.

#### Scenario: Geteilter Link öffnet gefiltert
- **WHEN** ein Nutzer `/termine?q=ludwigsburg` aufruft
- **THEN** ist das Eingabefeld mit `ludwigsburg` vorbelegt und die Liste entsprechend gefiltert

#### Scenario: Tippen erzeugt keine History-Einträge
- **WHEN** ein Nutzer acht Zeichen in das Feld tippt
- **THEN** ist ein einzelner Druck auf „Zurück" ausreichend, um die Seite zu verlassen

### Requirement: Fokussierte Elemente überleben den Textfilter

Ein über den `focus`-Parameter markiertes Element (Wiederherstellung der Scroll-Position
nach „Zurück") SHALL sichtbar bleiben, auch wenn das gesetzte `q` auf keines seiner Felder
passt. Der Textfilter SHALL in der Prädikatenkette **nach** den bestehenden Filtern
ausgewertet werden.

#### Scenario: Fokus schlägt den Textfilter
- **WHEN** auf `/dienste` `focus=slot-42` und ein `q` gesetzt sind, das auf die zugehörige Gruppe nicht passt
- **THEN** bleibt die Gruppe sichtbar
