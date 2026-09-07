## Why

Die Push „Neuer Dienst verfügbar" (`POST /api/duty-slots`, `duties.eligibleDutyUsers`)
trifft heute deutlich mehr Leute als der Dienst betrifft. Sie kennt nur den **Team-Scope**
des Slots — die beiden anderen Filter, die die Dienstbörse beim Lesen anlegt, fehlen:

1. **Keine Zielgruppe.** Ein Slot mit `audiences=["eltern"]` (bzw. der Vorbelegung aus
   `duty_types.audiences`) geht trotzdem an alle Spieler und Trainer des Teams. Die
   Empfänger bekommen eine Push für einen Dienst, den ihnen die Dienstbörse anschließend
   gar nicht anzeigt — der Klick auf die Meldung führt ins Leere.
2. **Keine aktive Saison.** `player_memberships` wird ohne `seasons.is_active = 1`
   gejoint. Wer vor drei Jahren im Kader der A-Jugend stand, wird bis heute für jeden
   Dienst dieses Teams benachrichtigt. Das ist der stille Dauer-Fehlversand — er wächst
   mit jeder abgeschlossenen Saison.
3. **Trainer fehlen.** Der Team-Zweig sucht ausschließlich in `player_memberships`. Ein
   Trainer steht aber in `kader_trainers` (`trainer_memberships`) und wird deshalb nur
   erreicht, wenn er zufällig auch als Spieler im selben Kader geführt wird — obwohl die
   Dienstbörse ihm den Slot zeigt und ein Slot mit `audiences=["trainer"]` genau ihn
   meint.

Der Maßstab steht schon im Code: `Board` (`GET /api/duty-board`) löst Sichtbarkeit über
Team-Quelle (Spieler / Trainer / Eltern, jeweils **aktive Saison**) **und** Audience-Match
auf. Die Benachrichtigung soll dieselbe Menge treffen — nicht mehr, nicht weniger.

## What Changes

- `duties.eligibleDutyUsers` wird zu `eligibleDutyRecipients(ctx, teamIDs, audiences)` und
  spiegelt die Sichtbarkeitsregel der Dienstbörse:
  - **Team-Quelle** (unverändert aus `slotTeamScope` abgeleitet): Spieler im Kader,
    Trainer des Kaders (`trainer_memberships`), Eltern eines Spielers im Kader — jeweils
    **nur in der aktiven Saison**.
  - **Zielgruppe**: `COALESCE(duty_slots.audiences, duty_types.audiences)`. NULL/leer =
    keine Einschränkung; sonst Treffer über eine Vereinsfunktion des Empfängers oder —
    für `eltern` — über ein Kind im Team-Scope (dieselbe Team-Bindung wie im Board).
- `duties.CreateSlot` löst die wirksame Zielgruppe (Slot-Wert, sonst Vorbelegung des
  Diensttyps) auf und übergibt sie an die Empfängerbestimmung.
- Die **Lese-Privilegien** von `admin`/`vorstand` werden bewusst **nicht** übernommen: wer
  in der Dienstbörse alles sehen darf, wird deshalb nicht über jeden neuen Dienst
  benachrichtigt. „Darf sehen" ist keine Betroffenheit.

Nicht Teil dieses Changes: die Dienst-Erinnerung T-2 (`internal/scheduler`) — sie zielt
bereits über `duty_types.target_role` + aktive Saison und hat eine eigene Empfängerregel.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `push-duties`: Empfängermenge bei `POST /api/duty-slots` folgt zusätzlich der Zielgruppe
  und der aktiven Saison; Trainer des Kaders werden erreicht.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `POST /api/duty-slots` | `TestCreateSlot_ZielgruppeEltern_BenachrichtigtNurEltern` | 201; Elternteil eines Spielers im Team im `user_events`-Set, Spieler und Trainer desselben Teams **nicht** |
| `POST /api/duty-slots` | `TestCreateSlot_ZielgruppeTrainer_BenachrichtigtTrainerDesKaders` | 201; Trainer aus `kader_trainers` im Set, Spieler des Teams nicht |
| `POST /api/duty-slots` | `TestCreateSlot_ZielgruppeAusDiensttyp_WirdAngewendet` | 201; Slot ohne eigene `audiences` erbt die Zielgruppe des Diensttyps |
| `POST /api/duty-slots` | `TestCreateSlot_OhneZielgruppe_BenachrichtigtDenGanzenKader` | 201; Spieler, Trainer und Eltern des Teams im Set (Regression zur Bestandszusage) |
| `POST /api/duty-slots` | `TestCreateSlot_AltePlayerMembership_WirdNichtBenachrichtigt` | 201; Spieler, der nur in einer **inaktiven** Saison im Kader stand, ist **nicht** im Set |

Garantierte Invariante: Die Empfängermenge der „Neuer Dienst verfügbar"-Meldung ist eine
Teilmenge derer, denen `GET /api/duty-board` den Slot ohne `?audience=all` anzeigt.
