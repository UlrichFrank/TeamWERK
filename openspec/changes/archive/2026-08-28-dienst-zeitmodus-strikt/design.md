# Design

## 1. Warum der Rückfall weg muss, obwohl er als Sicherheitsnetz gedacht war

Der Rückfall (`resolveSlotHours`: Differenz ≤ 0 → `hours_value`) hat einen echten Fall
abgedeckt: Ein Vorstand vertippt sich beim Versatz, und statt eines fehlenden Zeitnehmers
am Spieltag gibt es einen Zeitnehmer mit einer etwas anderen Dauer. Das Argument stimmt
für **fehlende** Slots — aber nicht für den Preis, den es bezahlt: Die falsche Definition
bleibt dauerhaft unsichtbar. Der Slot sieht in der Dienstbörse korrekt aus, das
Dienstkonto bucht die Rückfall-Stunden, und beim nächsten Regen passiert wieder dasselbe.
Es gibt keinen Zeitpunkt, an dem jemand den Fehler bemerkt.

Die Alternative ist nicht „Dienst fällt aus statt Rückfall", sondern eine Kette:

1. **Verhindern** — was nie funktionieren kann, wird beim Pflegen abgewiesen (§2).
2. **Melden** — was erst gegen einen konkreten Termin scheitert, erscheint sichtbar in
   der Regen-Zusammenfassung (§3).

Erst wenn beide Stufen greifen, ist der Ausfall eines Slots kein stiller Verlust mehr,
sondern eine Meldung mit Datum und Diensttyp.

## 2. Was ist „provably impossible"? (die Validierungsregel)

Mit `A_s`, `A_e` ∈ {Anpfiff, Spielende}, den Versätzen `o_s`, `o_e` und der Spieldauer
`D > 0` gilt:

| Start-Anker | End-Anker | Dauer |
|---|---|---|
| gleich | gleich | `o_e − o_s` |
| Anpfiff | Spielende | `D + o_e − o_s` |
| Spielende | Anpfiff | `o_e − o_s − D` |

Nur die **erste** Zeile ist ohne Kenntnis von `D` entscheidbar: Bei gleichem Anker ist die
Dauer genau die Versatz-Differenz, unabhängig von jedem Termin. `o_e ≤ o_s` heißt dort
also „diese Definition kann an **keinem** Spieltag eine positive Dauer ergeben" — das ist
ein Eingabefehler und wird mit 400 abgewiesen.

Die beiden anderen Zeilen sind **bewusst nicht** validiert. „Start bei Anpfiff, Ende
15 min vor Spielende" (`o_e − o_s = −15`) ist eine völlig legitime Definition — sie ergibt
bei jedem Spiel, das länger als 15 Minuten dauert, eine positive Dauer. Eine Prüfung, die
für *jede denkbare* Spieldauer positiv verlangt, würde diesen Fall verbieten und wäre
damit strenger als die Fachlichkeit. Die Spieldauer steht erst am konkreten Termin fest
(`age_class_game_rules` je Altersklasse, `game_templates.duration_minutes` bei
generischen Terminen) — für heim/auswärts-Vorlagen ist sie zum Pflegezeitpunkt schlicht
unbekannt.

Konsequenz: Die Prüfung ist **notwendig, nicht hinreichend**. Sie fängt den Tippfehler,
nicht die knappe Kombination. Deshalb braucht es §3.

## 3. Der Restfall beim Regen: kein Slot, aber eine Meldung

Ergibt die Auflösung gegen einen konkreten Termin trotzdem ≤ 0, erzeugt das Item dort
keinen Slot. Der Eintrag landet in `RegenSummary.InvalidSpan` (`invalid_span` im JSON) —
eine eigene Liste, nicht `Skipped`:

- `Skipped` bedeutet heute **„die Varianten-Logik hat den Dienst absichtlich ausgelassen"**
  (`same_day_behavior='skip'`). Das ist eine gewollte Regel, die Karte zeigt sie neutral.
- `InvalidSpan` bedeutet **„die Definition passt zu diesem Termin nicht"**. Das ist ein
  Fehler, der gepflegt werden muss, und wird in `RegenSummaryCard` rot ausgewiesen — wie
  `conflicts`.

Die beiden in eine Liste zu werfen hieße, den Fehler in der Anzeige als Regel zu tarnen —
genau die Unsichtbarkeit, die dieser Change abschafft.

Für die Benachrichtigungen ändert sich **nichts** und muss sich nichts ändern: Das Item
trägt keinen Eintrag in `outcomeByOriginalType` ein (nacktes `continue`, wie der
Ausrichter-Gate-Miss und der Rotations-Miss). `buildNotificationIntents` behandelt einen
fehlenden Eintrag als `removed` — eine Zusage auf dem eben gelöschten Bestandsslot führt
also zu „Dein Dienst wurde entfernt". Das ist für einen tatsächlich entfallenen Dienst
genau die richtige Meldung.

## 4. Warum die Modus-Werte in der DB bleiben

Die Umbenennung ist eine **Beschriftung**, keine Semantikänderung: `absolut` erzeugt
weiter Start + feste Stundenzahl, `dynamisch` weiter Start + Ende. Eine Migration von
`absolut`→`startzeit_dauer` würde zwei CHECK-Constraints, zwei Handler, ein
Frontend-Mapping und alle Tests anfassen, um exakt dieselbe Bedeutung anders zu
buchstabieren. Die Bezeichner stehen im UI-Text, wo sie hingehören.

`hours_value` bleibt im Modus `dynamisch` **gespeichert**, wird dort aber nicht mehr
gelesen. Der Wert überlebt damit einen Moduswechsel hin und zurück, ohne dass die Maske
ihn in einem Modus zeigen muss, in dem er nichts tut. Die Validierung „Dauer > 0" bleibt
unverändert bestehen — sie schützt den Modus `absolut`.

## 5. Reihenfolge der Felder = Reihenfolge der Rechnung

Beide Masken (`/diensttypen`, `/dienstplan-vorlagen`) zeigen ab jetzt:

```
( ) Startzeit + Dauer      ( ) Startzeit + Endzeit
Start-Anker      Start-Versatz
─────────────────────────────────────────
Dauer                 │  End-Anker    End-Versatz
(Modus 1)             │  (Modus 2)
```

Der Start steht in beiden Modi an derselben Stelle und heißt in beiden gleich; nur die
untere Zeile wechselt. Vorher war die Dauer *oberhalb* der Endfelder und trug im
dynamischen Modus die Beschriftung „Dauer (Rückfall)" — ein Feld, das nur existierte, um
einen Fehlerfall abzufangen, stand damit prominenter als die Felder, die den Normalfall
bestimmen.

## 6. Der Termin-Dialog bleibt bei Start + Dauer

`/kalender` legt über „+ Dienst hinzufügen" und „Bearbeiten" immer `is_custom=1`-Slots an
(`duties.CreateSlot` schreibt 1, `UpdateSlot` setzt es auf 1). Solche Slots fasst der
Regen nie an — ein Modus „folgt dem Spiel" könnte dort nichts nachführen. Die
Vorbelegung aus einem dynamischen Diensttyp bleibt (die Rechnung gegen *diesen* Termin
ist ja bekannt), das Ergebnis ist danach eine feste Zahl.

Neu ist nur, dass der Dialog das sagt. Bisher war die Nebenwirkung von „Bearbeiten" —
der Slot verlässt die automatische Regeneration — nirgends sichtbar; wer die Uhrzeit
korrigierte, nahm den Dienst ungewollt dauerhaft aus dem Regen. Der Hinweis steht in
beiden Dialogen (Anlegen und Bearbeiten), im Bearbeiten-Dialog als Warnung vor der
Nebenwirkung, im Anlege-Dialog als Feststellung.
