# Design

## 1. Häkchen an Modus 2, kein dritter Dauer-Modus

Der Wunsch kam als „dritte Option neben Start+Dauer und Start+Ende" — die Rechnung ist
aber keine dritte Art, ein Ende zu definieren, sondern eine **Kappung der zweiten**:

```
Ende = MIN( Ablösung , gepflegtes Ende )
```

Ein dritter `duration_mode`-Wert müsste End-Anker und End-Versatz trotzdem tragen, weil
der Deckel genau daraus besteht. Modus 2 und Modus 3 hätten in der Maske exakt dieselben
Felder — ein starkes Signal, dass sie kein eigener Modus sind. Also:

| | gewählt |
|---|---|
| Speicherung | `duration_mode` bleibt `('absolut','dynamisch')`; neue Spalte `end_at_next_duty` (0/1) |
| Maske | Häkchen unter End-Anker/-Versatz: „Endet spätestens bei Ablösung durch den nächsten gleichartigen Dienst am selben Tag" |
| Code | `resolveSlotHours` bleibt zweiwertig; die Kappung ist ein eigener, nachgelagerter Schritt |

Der Preis: Der Vorstand sieht zwei Radio-Buttons plus ein Häkchen statt drei
Radio-Buttons. Vertretbar, weil das Häkchen genau dort steht, wo es wirkt — direkt unter
dem Ende, das es deckelt.

## 2. „Gleichartig" = derselbe Diensttyp, nicht das nächste Spiel

Die Ablösung dockt am nächsten **Dienst** an, nicht am nächsten **Spiel**. Der Unterschied
wird sichtbar, sobald die Rotation nicht jedem Heimspiel einen Slot gibt:

```
  Spiel1 (Slot A)   Spiel2 (Slot B)   Spiel3 (kein Slot)   Spiel4 (kein Slot)

  nächster DIENST   B endet an seinem Deckel (Spiel2-Ende + Versatz)
  (gewählt)         └─ die Kette bricht ab, wo die Rotation aufhört

  nächstes SPIEL    B endet bei Spiel3-Start − Versatz
  (verworfen)       └─ B erbt zwei weitere Spiele, obwohl der Bedarf
                       nur zwei Kuchen ausgewiesen hat
```

„Nächstes Spiel" würde der letzten eingeteilten Mannschaft stillschweigend Arbeit
zuschieben, für die niemand eingeteilt wurde — und ihr die Stunden dafür gutschreiben.
Dass Spiel 3 und 4 dann unbewirtet bleiben, ist eine Eigenschaft von
`bewirtung_verhaeltnis`, nicht der Dauer (siehe Non-Goals).

Maßgeblich ist der **resultierende** Diensttyp des Slots, also `duty_slots.duty_type_id`
nach einer eventuellen Varianten-Reduktion — nicht der Diensttyp der Vorlagen-Zeile. Das
ist derselbe Diensttyp, den ein Helfer in der Dienstbörse liest, und damit das, was
„gleichartig" umgangssprachlich meint.

## 3. Die Kappung liest reale Slots, keine vorhergesagten

Die naheliegende Bauform wäre ein Prepass vor der Pro-Spiel-Schleife — so löst
`buildRotationPlan` seine tagesweite Abhängigkeit. Für die Ablösung ist das die falsche
Stelle, weil ein Prepass die Startzeiten der Nachfolger **vorhersagen** müsste. Zwischen
„Vorlagen-Zeile existiert" und „Slot entsteht" liegen fünf Gates:

```
   Item aus der Vorlage
        ├─ Ausrichter-Gate ──────────▶ kein Slot
        ├─ Rotationsplan: kein Anker ▶ kein Slot
        ├─ Team-Allowlist verfehlt ──▶ kein Slot
        ├─ Varianten-Logik ──────────▶ ANDERER duty_type_id
        ├─ Konflikt mit is_custom=1 ─▶ kein Slot
        └─ Spieldauer unbekannt ─────▶ ganzes Spiel übersprungen
```

Rät ein Prepass hier falsch, bekommt der Vorgänger einen Nachfolger, den es nicht gibt —
und trägt eine zu kurze Dauer, die völlig plausibel aussieht. Das ist exakt die
Fehlerklasse, die `dienst-zeitmodus-strikt` gerade beseitigt hat.

**Gewählt: ein Nachlauf am Ende von `regenSingleDay`**, nach der Pro-Spiel-Schleife und
damit nach allen `INSERT`s des Tages:

```
  Phase 1  Pro-Spiel-Schleife, unverändert.
           Jeder Slot entsteht mit der Dauer aus Modus 2 (dem Deckel).
           Der invalid_span-Gate greift wie heute.
           Nebenbei gesammelt: die IDs der Slots mit end_at_next_duty=1.
                        │
                        ▼
  Phase 2  EIN SELECT über alle duty_slots des Tages (event_date + season_id).
           Je Kandidat: früheste event_time desselben duty_type_id, die nach
           der eigenen Startzeit liegt → hours_value kürzen, falls das früher
           ist als der Deckel.
```

Ein einziges `SELECT` je Tag. Alle sechs Gates sind dadurch automatisch berücksichtigt,
weil die Kette ausschließlich über Zeilen läuft, die tatsächlich in `duty_slots` stehen.

Drei Eigenschaften fallen dabei kostenlos ab:

- **`is_custom=1`-Slots lösen ab.** Ein von Hand angelegter Bewirtungsdienst ist ein
  realer gleichartiger Dienst; dass er nicht aus einer Vorlage stammt, ändert daran
  nichts. Gekappt wird er nie — manuell angelegte Dienste tragen laut Bestandsregel eine
  absolute Dauer und sind nie Kandidat.
- **Ausnahme ≠ Kontext gilt weiter.** Slots an Terminen aus `excluded_game_ids` wurden
  nicht gelöscht, stehen also in der Tabelle und lösen ab — dieselbe Regel, der
  `loadSameDayContextTx` schon folgt, hier ohne eigenes Zutun.
- **`purge`/`none` im Massenlauf wirken korrekt.** Dort gelöschte Slots stehen nicht mehr
  in der Tabelle und lösen folgerichtig nicht mehr ab.

## 4. Nur Nachfolger nach dem eigenen Start — und was daraus folgt

Ein „Nachfolger", der **vor** dem eigenen Slot startet, ist keine Ablösung. So etwas
entsteht, sobald zwei Termine desselben Tages Vorlagen mit unterschiedlichen
Start-Versätzen tragen. Buchstäbliches `min()` lieferte dort eine negative Spanne und
löschte den Dienst — ein vertippter Versatz würde den Kuchendienst entfernen. Deshalb:
**es zählen nur Slots mit `event_time > eigene Startzeit`**, alles andere fällt sauber auf
den Deckel zurück.

Daraus folgt die tragende Invariante des ganzen Changes:

```
  Ablösung  >  Startzeit          (per Auswahlregel)
  Deckel    >  Startzeit          (sonst wäre der Slot gar nicht entstanden,
                                   dienst-zeitmodus-strikt)
  ────────────────────────────────────────────────────────────
  MIN(Ablösung, Deckel) > Startzeit   →  die Dauer bleibt IMMER positiv
```

Die Kappung kann also **niemals** einen Slot entfallen lassen. Konsequenzen:

- Phase 2 ist ein reines `UPDATE duty_slots SET hours_value=?` — kein `DELETE`.
- Keine neuen `invalid_span`-Einträge.
- Keine Berührung mit `restoreAssignments` (matcht über
  `(duty_type_id, event_time)`, die Dauer geht nicht in den Schlüssel ein) und keine mit
  `buildNotificationIntents` (kein Slot verschwindet, also gibt es nichts zu melden).
- **Keine neue Eingabe-Validierung.** Ein gesetztes Kennzeichen kann eine Definition nicht
  unmöglich machen; `dynamicSpanImpossible` bleibt unverändert.

Das ist der Grund, warum der Change trotz tagesweiter Abhängigkeit klein bleibt.

## 5. Bei Varianten-Reduktion gilt der Diensttyp der Variante

`dienst-dauer` Decision 3 hat festgelegt: greift die Varianten-Logik, entsteht der Slot
unter einem anderen Diensttyp, und dann gilt dessen Dauer — „eine Variante ist kein
Rabatt, sondern eine andere Arbeit". `dienst-zeitmodus-strikt` hat das auf Modus,
End-Anker und End-Versatz fortgeschrieben. `end_at_next_duty` folgt derselben Regel: Ob
sich eine Arbeit ablösen lässt, gehört zur Arbeit.

## 6. Fairness: die Kappung verkürzt nur

`duty_accounts.ist` summiert `duty_slots.hours_value`. Weil die Kappung eine Dauer nur
verkürzen kann, verschiebt sie Stunden systematisch **weg von den frühen** Positionen der
Rotations-Warteschlange: Wer abgelöst wird, bekommt weniger gutgeschrieben; wer zuletzt
dran ist, behält den vollen Deckel.

Bewusst akzeptiert. Die frühen Mannschaften arbeiten tatsächlich kürzer — die heutige
Gutschrift enthält Zeit, in der bereits die nächste Mannschaft am Stand steht. Der Change
korrigiert damit eine Überzahlung, er erzeugt keine Unterzahlung.

Erwähnenswert nur, weil die Zahlen in bestehenden Konten **nicht** rückwirkend korrigiert
werden: Bestandsslots behalten ihre Dauer bis zum nächsten Regen-Lauf.

## 7. Die Einzeltermin-Vorschau zeigt den Deckel

`GET /api/duty-templates/{id}/preview-slots` (Termin-Dialog, Anlege-Wizard) rechnet gegen
**einen** Termin und kann die Kette strukturell nicht kennen. Sie zeigt deshalb weiterhin
den Deckel — also möglicherweise eine längere Dauer, als der Slot am Ende trägt.

Bewusst nicht nachgebaut: Ein Client-seitiger Nachbau der Kette wäre eine zweite Quelle
derselben Regel und würde driften. Die Vorschau bleibt damit eine **obere Schranke**, was
in ihrer Rolle (grobe Kontrolle vor dem Speichern) vertretbar ist.

Der Massenlauf-Dry-Run ist davon nicht betroffen: `preview` ist derselbe Codepfad wie
`apply`, nur mit `Rollback` statt `Commit` — er durchläuft Phase 2 mit und zeigt die
gekappten Dauern exakt.

## Risks

| Risiko | Bewertung |
|---|---|
| Zusätzliches `SELECT` je Tag im Regen | Ein Query über einen Tag; der Regen macht ohnehin pro Spiel mehrere. Vernachlässigbar. |
| Sehr kurze Restdauern (Nachfolger startet 5 min später) | Positiv, also zulässig. Sichtbar in der Dienstbörse als kurze Spanne — das ist dann eine korrekte Aussage über eine schlecht gepflegte Vorlage, keine falsche Zahl. |
| Zwei Slots desselben Typs zur exakt gleichen Zeit | Lösen einander nicht ab (`>`, nicht `>=`); beide behalten ihren Deckel. Gewollt. |
| Kennzeichen an einem nicht-rotierenden Diensttyp (z. B. Zeitnehmer) | Zulässig — jeder Diensttyp darf es tragen. Eine sinnlose Kombination ist möglich, aber offensichtlich, und eine Sonderregel müsste auf zwei Tabellen mit unterschiedlichen Flags gepflegt werden. |
| Bestandsslots tragen bis zum nächsten Regen die alte Dauer | Wie bei jeder Dauer-Änderung. Kein Backfill. |
