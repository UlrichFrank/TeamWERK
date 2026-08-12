## Context

Der Auto-Regen ist bereits tagesweise organisiert: `regenSingleDay(date, seasonID)` lädt die Spiele des Tages (`loadDayGames`), berechnet mit `buildRotationPlan` eine **tagesweite** Vorausberechnung (Kuchenbedarf, Team-Warteschlange) und läuft danach die Pro-Spiel-Schleife `regenGameItems`. Der Ausrichter hat exakt dieselbe Granularität — ein Wert je Tag, der beeinflusst, welche Vorlagen-Zeilen überhaupt Slots erzeugen. Er fügt sich damit als zweite tagesweite Vorausberechnung neben `buildRotationPlan` ein, ohne dass die Engine eine neue Ebene lernen muss.

Die vorhandenen Filter auf `game_template_items` filtern entlang **Mannschaften**: `team_ids` (Change `duty-template-team-scope`, Migration `044`) schränkt ein, welche der am Spiel beteiligten Teams einen Slot bekommen; `rotation_enabled` (Change `bewirtung-cap-global`, Migration `046`) schaltet die tagesweite Kuchenrotation frei. Beide sagen nichts darüber, **ob der Dienst an diesem Tag überhaupt anfällt** — das ist die Lücke, die der Ausrichter schließt.

`internal/settings` ist im Architektur-Test (`internal/arch/arch_test.go`) als **Foundation** klassifiziert und wird bereits von `internal/games/regen.go` importiert (`GetBewirtungVerhaeltnis`). `internal/stammvereine` ist dagegen **Domain** — ein Domain-Package darf ein anderes nicht importieren. Das entscheidet, wo die Ausrichter-Auflösung liegen muss (Decision 3).

Die Preview/Apply-Mechanik aus `duty-bulk-regen` (`internal/games/bulkregen_handler.go`) führt beide Wege durch denselben Code und unterscheidet sich nur im Abschluss (`tx.Rollback()` statt `Commit`). Das ist die vorhandene Antwort auf „zeig mir, was das kaputtmacht, bevor du es tust" und wird hier wiederverwendet statt nachgebaut.

## Goals / Non-Goals

**Goals:**
- Ein Ausrichter je Spieltag, dessen Auflösung **total** ist — es gibt keinen Zustand „kein Ausrichter".
- Vorlagen-Zeilen optional an einen Ausrichter binden, ohne bestehende Vorlagen zu verändern (rein additiv).
- Das Gate an genau den zwei Stellen im Regen, an denen es wirken muss, ohne eine dritte Regen-Implementierung.
- Jede ausrichter-bedingte Löschung von Diensten läuft über eine Bilanz-Vorschau.

**Non-Goals:**
- Keine Zurechnung von Diensten an Mitglieder des Ausrichters (der Ausrichter **filtert**, er **weist nicht zu**).
- Kein Ausrichter je Halle oder je Spiel.
- Keine Heuristik und kein zusätzlicher Schritt im H4A-Import.
- Keine Vorschau beim Wechsel des Default-Ausrichters.
- Kein saisonübergreifender Zustand und keine Rotation der Ausrichter (anders als bei der Kuchen-Warteschlange gibt es hier keine Reihenfolge).

## Decisions

### Decision 1: Der Ausrichter hängt am Tag, nicht am Spiel

`spieltag_ausrichter (date, season_id, ausrichter_id)` mit `PRIMARY KEY (date, season_id)`.

Alternative wäre `games.ausrichter_id` gewesen. Verworfen, weil die Kuchen-Rotation tagesweise rechnet: bei zwei Spielen desselben Tages mit unterschiedlichem Ausrichter wäre der Bedarf eines Tages nicht mehr wohldefiniert, und `buildRotationPlan` müsste in Untergruppen zerfallen. Fachlich richtet ohnehin ein Verein die Halle für den ganzen Tag aus.

**Folge:** Zwei Heimspiele desselben Tages in verschiedenen Hallen mit verschiedenen Ausrichtern sind kein modellierter Fall. Tritt er auf, gilt für beide derselbe Ausrichter und der Vorstand korrigiert die betroffenen Slots als `is_custom=1`-Einträge von Hand — derselbe Ausweg wie heute.

`season_id` gehört in den Schlüssel, weil auch `games` saisongebunden ist und `regenSingleDay` immer `(date, seasonID)` führt. Ohne die Saison würde ein Datum aus einer archivierten Saison mit demselben Tag der aktiven kollidieren.

### Decision 2: Die Auflösung ist total — Default statt Leerzustand

```
ausrichter(tag) = spieltag_ausrichter[(date, season)] ?? ausrichter.is_default
```

Genau ein Eintrag der Liste trägt `is_default=1`, mechanisch erzwungen durch einen Partial-Unique-Index:

```sql
CREATE UNIQUE INDEX idx_ausrichter_default ON ausrichter(is_default) WHERE is_default = 1;
```

Das ist die zentrale Vereinfachung dieses Designs. Weil nie ein Tag ohne Ausrichter existiert, entfallen alle Sonderfälle, die eine „fehlt noch"-Modellierung nach sich zöge:

- Der H4A-Import braucht keinen zusätzlichen Schritt und keine Heuristik — importierte Spiele fallen auf den geltenden Tageswert bzw. den Default.
- Die Bestandsmigration muss nichts reparieren.
- Der Kalender braucht kein „Ausrichter fehlt"-Warnsignal.
- Der Regen kann nie in einen fail-open/fail-closed-Zweig laufen.

**Preis:** Der Default kann still falsch sein. Das ist akzeptiert, weil in der Praxis überwiegend derselbe Verein ausrichtet — die Abweichung ist der Sonderfall, den man einträgt, nicht die Regel.

**Ein `NULL` in `spieltag_ausrichter.ausrichter_id` verhält sich identisch zu „keine Zeile"** und fällt ebenfalls auf den Default. Das hält den Zustand nach dem Löschen eines Ausrichters (Decision 6) trivial korrekt, ohne aufräumen zu müssen.

### Decision 3: Auflösung liegt in `internal/settings` (Foundation), Preview/Apply in `internal/games`

Die Zuständigkeit wird vom Architektur-Test diktiert, nicht von der Fachlichkeit:

| Was | Wo | Warum |
|---|---|---|
| Liste, CRUD, `ResolveAusrichterForDay` | `internal/settings/ausrichter.go` | `settings` ist Foundation und wird von `internal/games` bereits importiert. Ein eigenes Domain-Package `internal/ausrichter` wäre ein Domain→Domain-Import und würde `TestArchitecture_*` brechen. |
| Routen der Liste | `internal/settings/handler.go` | Gehört zum Einstellungen-Tab, keine Regen-Abhängigkeit. |
| `POST /api/game-days/host/{preview,apply}` | neu `internal/games/ausrichter_handler.go` | Braucht den **unexportierten** `runAutoRegen`. Exakt die Begründung, aus der `h4aimport_handler.go` in `games` liegt statt in `h4aimport`. |

`ResolveAusrichterForDay` nimmt — wie `GetBewirtungVerhaeltnis` — das schmale `RowQuerier`-Interface statt `*sql.DB`, damit die Regen-Engine innerhalb ihrer laufenden `tx` liest. Bewusst **ohne** `Store`/Cache (kein Hot-Path, ein `SELECT` je Regen-Tag).

### Decision 4: Das Gate sitzt an zwei Stellen, und die Reihenfolge ist wesentlich

`regenSingleDay` löst den Tages-Ausrichter **einmal** auf und reicht ihn durch:

```
regenSingleDay(date, seasonID)
  ├─ loadSameDayContextTx        (unverändert)
  ├─ loadDayGames                (unverändert)
  ├─ ResolveAusrichterForDay ──▶ dayAusrichterID          ◀── NEU, einmal je Tag
  ├─ buildRotationPlan(…, dayAusrichterID)                ◀── Gate #1
  │     filtert rotations-aktive Items VOR der Bedarfsrechnung
  └─ for game in dayGames:
        regenGameItems(…, dayAusrichterID)                ◀── Gate #2
              itemApplies = !item.AusrichterID.Valid
                          || item.AusrichterID.Int64 == dayAusrichterID
```

Gate #1 ist der nicht offensichtliche Teil. Ließe man es weg und filterte erst in `regenGameItems`, würde `buildRotationPlan` den Kuchenbedarf über Heimspiele rechnen, deren Slots danach verworfen werden — die Team-Warteschlange verbrauchte Positionen für nie entstehende Slots, und der ausgewiesene Bedarf im `RegenSummary` wäre schlicht falsch. Der Filter gehört deshalb in denselben Schleifendurchlauf, der die rotations-aktiven Items ohnehin sammelt.

**Das Gate wirkt nur bei `event_type='heim'`.** Auswärts- und generische Termine ignorieren `ausrichter_id` vollständig — Vorlagen mit `template_type != 'heim'` dürfen das Feld gar nicht erst setzen (Decision 5), der Zweig ist also nur eine Sicherung.

### Decision 5: `ausrichter_id` nur auf Heim-Vorlagen — serverseitig erzwungen

`POST`/`PUT` auf Vorlagen-Items lehnt ein gesetztes `ausrichter_id` mit `400 ausrichter_requires_heim_template` ab, wenn die Vorlage nicht `template_type='heim'` ist. Gleiches Muster wie `400 rotation_requires_normal_behavior` bei der Kuchenrotation: eine Kombination, die fachlich nichts bedeutet, wird nicht gespeichert, statt sie in der Engine still zu ignorieren.

Im Frontend erscheint die Auswahl entsprechend nur bei Heim-Vorlagen — die Server-Prüfung bleibt trotzdem, weil sie die eigentliche Garantie ist.

### Decision 6: Löschen eines Ausrichters — Spieltage entkoppeln, Vorlagen-Zeilen mitlöschen

Zwei Referenzen, zwei unterschiedliche Antworten:

| Referenz | Verhalten beim Löschen | Warum |
|---|---|---|
| `spieltag_ausrichter.ausrichter_id` | `SET NULL` → Tag fällt auf den Default | Entspricht Decision 2: `NULL` ist ein wohldefinierter Zustand. Der Tag verliert nur seine Abweichung. |
| `game_template_items.ausrichter_id` | Die betroffenen **Items werden mitgelöscht** | `SET NULL` hieße hier „gilt ab jetzt immer" — die Zeile würde nach dem Löschen **mehr** Dienste erzeugen als vorher. |

Das ist die einzige Stelle im Change, an der eine Löschung den Umfang erweitern könnte, und deshalb explizit anders gelöst. Ein Dienst, der nur existiert, weil Verein X ausrichtet, hat ohne Verein X keine Bedeutung.

Mechanisch: `game_template_items.ausrichter_id` bekommt `ON DELETE RESTRICT`, damit es keinen stillen `SET NULL`-Pfad gibt. Der Handler führt die Kaskade explizit in einer Transaktion aus, nachdem `GET /api/ausrichter/{id}/usage` die betroffenen Spieltage und Vorlagen-Zeilen benannt und der Vorstand bestätigt hat.

**Der Default-Ausrichter ist nicht löschbar** (`409 default_ausrichter_undeletable`) — sonst wäre die Auflösung aus Decision 2 nicht mehr total. Erst einen anderen Eintrag zum Default machen, dann löschen.

### Decision 7: Vorschau über den bestehenden Dry-Run, nicht über einen neuen

`POST /api/game-days/host/preview` schreibt `spieltag_ausrichter`, ruft `runAutoRegen` für den Tag und macht am Ende `tx.Rollback()` statt `Commit` — identisch zum Muster aus `duty-bulk-regen`. Kein Client-Nachbau der Regen-Logik, damit Vorschau und Ergebnis nicht auseinanderlaufen können.

**Der geerbte Trade-off gilt hier ebenso:** der Dry-Run ist eine echte Schreibtransaktion und serialisiert kurzzeitig gegen andere Schreiber. Bei einem einzelnen Tag ist das unkritisch.

Die Bilanz kommt aus dem vorhandenen `RegenSummary` plus einer Vor-/Nach-Zählung von `duty_slots` und `duty_assignments` — dieselben Felder, die `bulkRegenRow` schon ausweist (`created`, `deleted_auto`, `assignments_kept`, `assignments_lost`).

### Decision 8: Kein Vorschau-Zwang beim Default-Wechsel

Ein Default-Wechsel ändert die Auflösung **aller** Spieltage ohne expliziten Wert, potenziell die halbe Restsaison. Trotzdem bewusst ohne Vorschau: der Default ist der Verein, der typischerweise ausrichtet, und wird einmalig eingerichtet — nicht laufend umgestellt. Ein Dry-Run über die gesamte Restsaison für einen Fall, der praktisch nicht eintritt, wäre unverhältnismäßig (er wäre zudem die teuerste Schreibtransaktion im System).

**Bewusst in Kauf genommen:** Wer den Default doch umstellt, erfährt die Wirkung erst beim nächsten Regen der betroffenen Tage. Sollte sich das in der Praxis als Problem zeigen, ist die Nachrüstung ein eigener Change — der Dry-Run-Pfad existiert dann bereits.

### Decision 9: Das Wizard-Feld ist ein Tages-Feld und muss als solches lesbar sein

Im Termin-Wizard steht der Ausrichter zwischen lauter Termin-Feldern, ändert aber den ganzen Tag — inklusive bereits bestehender Termine mit laufenden Zusagen. Das ist eine echte UX-Falle.

Auflösung: Das Feld ist explizit tagesbezogen beschriftet („Ausrichter am 14.09. — gilt für alle Termine dieses Tages") und vorbelegt mit dem aktuell **geltenden** Wert (explizit oder Default). Weicht der gewählte Wert davon ab, läuft das Speichern über dieselbe Vorschau wie im Kalender.

Alternative wäre gewesen, das Feld im Wizard nur anzuzeigen und Änderungen ausschließlich in der Tagesansicht zuzulassen (ein einziger Schreibpfad, Falle existiert nicht). Verworfen, weil der häufigste Fall — „ich lege den Spieltag an und weiß dabei, wer ausrichtet" — dann zwei getrennte Arbeitsschritte bräuchte.

## Risks / Trade-offs

- **Der Default kann still falsch sein.** Ein Tag ohne expliziten Eintrag erzeugt Dienste nach dem Default, ohne dass irgendwo „ungeprüft" steht. Mitigation: die Kalender-Tagesansicht zeigt sichtbar, ob der Wert explizit gesetzt oder geerbt ist (`is_explicit`).
- **Vier Schreibpfade auf denselben Wert** (Kalender, Wizard, Massenlauf, indirekt das Löschen eines Ausrichters). Mitigation: alle vier gehen durch dieselbe Preview/Apply-Funktion in `internal/games/ausrichter_handler.go`; keiner schreibt `spieltag_ausrichter` direkt.
- **Zwei Hallen an einem Tag sind nicht modellierbar** (Decision 1). Bewusst akzeptiert; Ausweg sind `is_custom=1`-Slots.
- **Das Gate ist unsichtbar, wenn niemand es nutzt.** Solange keine Vorlagen-Zeile ein `ausrichter_id` trägt, ändert der ganze Change nichts — was gut für den Deploy ist, aber heißt, dass ein Konfigurationsfehler in der Vorlage erst am erzeugten Dienstplan auffällt. Mitigation: Der Vorlagen-Editor zeigt je Zeile im Klartext, wann sie greift.
- **Migrationsnummer.** `046_bewirtung_cap_global` liegt zum Zeitpunkt dieses Designs uncommittet im Working Tree. Die Nummer ist vor dem Schreiben erneut zu prüfen (`ls internal/db/migrations/ | sort -V | tail -1`) — eine Nummer ≤ aktueller DB-Version wird von golang-migrate lautlos übersprungen.

## Migration Plan

1. Migration `047_*` anlegen: `ausrichter`, `spieltag_ausrichter`, `game_template_items.ausrichter_id`, Partial-Unique-Index.
2. **Seed:** genau eine Ausrichter-Zeile mit `is_default=1` (Name aus der Vereins-Stammdatenzeile bzw. fachlicher Platzhalter, in der UI umbenennbar). Ohne diesen Seed wäre die Auflösung nicht total.
3. Deploy ist verhaltensneutral: kein Item trägt `ausrichter_id`, das Gate ist für alle Zeilen offen.
4. Der Vorstand pflegt die Liste, setzt den Default und bindet danach einzelne Vorlagen-Zeilen. **Erst dieser Schritt** ändert erzeugte Dienste — und läuft über die Vorschau.

**Rollback:** `047_*.down.sql` entfernt Spalte und Tabellen. Weil das Gate additiv ist, verhält sich der Regen danach exakt wie vorher; verlorene Ausrichter-Zuordnungen sind Konfiguration, keine Nutzerdaten.

## Open Questions

- Wo genau in der Kalender-Tagesansicht der Picker sitzt (Tages-Header vs. Event-Info-Modal) — reine UI-Frage, in der Umsetzung zu entscheiden.
- Ob die Ausrichter-Liste ein SSE-Event `settings-changed` mitnutzt (wie die Bewirtungs-Felder) oder ein eigenes bekommt. Vorschlag: mitnutzen, solange nur der Einstellungen-Tab darauf hört; der Kalender lädt den Tageswert ohnehin mit dem Tag.
