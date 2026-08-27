## Why

`duty_accounts.ist` — die geleisteten Dienststunden eines Nutzers — wird im gesamten
Code an genau **zwei** Stellen geschrieben:

1. `INSERT OR IGNORE INTO duty_accounts (user_id, season_id, soll, ist) VALUES (…, 0)`
   beim Ziehen eines Dienstes (`internal/duties/handler.go:1136`)
2. eine vollständige Neuberechnung im Cascade-Delete eines Termins
   (`internal/games/handler.go:1532`)

**`Fulfill` (`POST /api/duty-assignments/{id}/fulfill`, `duties/handler.go:1172`) erhöht
`ist` nicht.** Der Handler setzt `duty_assignments.status='fulfilled'` und
`fulfilled_at`, rührt das Konto aber nicht an. `CashSubstitute` ebenso wenig.

Folge: Das Konto steht für jeden Nutzer auf `0`, bis zufällig ein Termin gelöscht wird, an
dem er eine erledigte Zuweisung hatte — dann springt es auf den korrekten Gesamtstand und
verharrt dort. `GET /api/duty-accounts` und der CSV-Export liefern damit eine Zahl, die
weder 0 noch die Wahrheit ist, sondern vom Löschverhalten des Vorstands abhängt. `soll -
ist` (die im Frontend angezeigte Bilanz) ist entsprechend falsch.

Aufgefallen ist das bei der Umsetzung von `dienst-dauer`, das die **Quelle** dieser
Aggregation von `duty_types.hours_value` auf `duty_slots.hours_value` umgestellt hat
(dort design.md, Decision 4). Der Quellenwechsel ist korrekt und getestet
(`TestDeleteGame_IstNutztSlotDauer`) — er repariert aber nur den einen Pfad, der
überhaupt rechnet.

## What Changes

_(Noch nicht ausgearbeitet — dieser Change ist bewusst als Stub festgehalten, damit der
Befund nicht in einem Design-Absatz verschwindet.)_

Zu klären, bevor Tasks geschrieben werden:

- **Inkrementell oder neu berechnen?** `Fulfill` könnte `ist = ist + slot.hours_value`
  buchen (billig, aber driftanfällig — jede vergessene Gegenbuchung bleibt für immer) oder
  dieselbe vollständige Neuberechnung fahren wie `DeleteGame` (teurer, aber
  selbstheilend und bereits erprobt). Die Neuberechnung ist ein Query über die Saison
  eines Nutzers; bei der Größenordnung dieses Vereins spricht wenig gegen sie.
- **Welche Statuswechsel buchen?** Nicht nur `Fulfill`, sondern auch `CashSubstitute`
  (zählt eine Ersatzzahlung als geleistet?), das Zurücknehmen einer Erledigung, `Unclaim`
  einer bereits erledigten Zuweisung und das Löschen eines einzelnen Slots
  (`DELETE /api/duty-slots/{id}`) — Letzteres kaskadiert heute Assignments weg, ohne `ist`
  nachzuziehen, hat also dasselbe Loch wie `Fulfill`, nur in die andere Richtung.
- **Bestandskorrektur.** Die heutigen `ist`-Werte sind unbrauchbar. Ein einmaliger
  Neuberechnungslauf über alle `(user_id, season_id)` gehört dazu — als Migration oder als
  Subcommand.
- **Verhältnis zu `dienstkonto-dynamische-soll-formel`.** Jene Capability beschreibt die
  `soll`-Seite; die Bilanz stimmt erst, wenn beide Seiten stimmen.

## Capabilities

### Modified Capabilities

- `duties` (voraussichtlich): `duty_accounts.ist` folgt dem Status der Zuweisungen statt
  nur dem Löschen von Terminen.

## Impact

- `internal/duties/handler.go` — `Fulfill`, `CashSubstitute`, `Unclaim`, `DeleteSlot`
- `internal/games/handler.go` — die bestehende Neuberechnung wird vermutlich zu einem
  geteilten Helfer
- Einmalige Bestandskorrektur über alle Konten
