## Context

`DutyPage` (`/dienste`) zeigt alle Dienst-Slots des Nutzers gruppiert nach Spiel, mit vollständiger Interaktion (Eintragen/Austragen, Zuteilungen, Admin-Aktionen). `SpieltagDetailPage` (`/kalender/{id}`) zeigt Slots desselben Spiels, aber nur als statische ProgressBar-Liste mit Admin-Verwaltung — kein Eintragen für normale Nutzer.

Der `/duty-board`-Endpunkt liefert bereits `claimed_by_me` und `vacancies` pro Slot mit nutzerspezifischer Berechnung. Die Slot-Rendering-Logik existiert zweifach und divergiert zunehmend.

## Goals / Non-Goals

**Goals:**
- Eintragen/Austragen direkt auf `/kalender/{id}` für alle eingeloggten Nutzer
- Gemeinsame `DutySlotList`-Komponente — eine Wahrheit für Slot-Darstellung
- Kein Verhalten von `DutyPage` ändern

**Non-Goals:**
- Neuen API-Endpunkt einführen — der bestehende `/duty-board` reicht aus
- `/kalender/{id}`-Response anreichern (umgeht vorhandene claimed_by_me-Logik)
- Slot-Management-Modals (Add/Edit) in die gemeinsame Komponente ziehen
- Mobile-Anpassung der neuen Darstellung (folgt separatem Change)

## Decisions

### 1. Backend: Filter auf bestehendem Endpunkt statt neuem Endpunkt

`GET /duty-board?game_id=<id>` filtert die bestehende Query via zusätzlichem `AND ds.game_id = ?`.

Alternativen verworfen:
- **Neuer Endpunkt** `/api/duty-slots/board?game_id=`: Dupliziert die gesamte Auth/Team-Filter-Logik — unnötiger Aufwand ohne Mehrwert.
- **`/kalender/{id}` Response erweitern**: Würde `claimed_by_me`-Berechnung ein zweites Mal implementieren und den Spielplan-Handler mit Duty-Logik koppeln.

### 2. Frontend: `DutySlotList` als reine Darstellungskomponente

`DutySlotList` bekommt `slots: BoardSlot[]` als Props und hält intern `expanded` / `assignments` / `cashAmount`. Sie kümmert sich nicht ums Fetching — das bleibt in den jeweiligen Pages.

Alternativen verworfen:
- **`DutyBoardPanel` mit eingebautem Fetch**: Versteckt den `gameId`-Parameter tief in der Komponente, erschwert Reload-Koordination mit Add-Slot-Modals in `SpieltagDetailPage`.
- **Copy-paste mit kleinen Anpassungen**: War der bisherige Zustand — führt zur weiteren Divergenz.

### 3. SpieltagDetailPage: Zwei parallele Fetches

`SpieltagDetailPage` holt weiterhin `GET /kalender/{id}` (Spieldaten + SlotDetail für Add/Edit-Modals) **und** zusätzlich `GET /duty-board?game_id={id}` (BoardSlot für Darstellung).

`SlotDetail` bleibt für das Add-Slot-Formular nötig (duty_type_name, role_description). Nach Slot-Mutation werden beide Datenquellen neu geladen.

### 4. ProgressBar entfällt

Die bisherige ProgressBar (`slots_filled / slots_total`) wird durch die Board-Darstellung ersetzt (`vacancies`, `claimed_by_me`). Die Information ist äquivalent; `slots_filled = slots_total - vacancies` ist jederzeit berechenbar, falls nötig.

## Risks / Trade-offs

- **Zwei Fetches auf SpieltagDetailPage** → minimaler Overhead, beide Requests sind klein und parallel ausführbar.
- **`game_id`-Filter umgeht Team-Sichtbarkeitscheck nicht** — der bestehende WHERE-Clause bleibt aktiv; ein Nutzer ohne Teamzugehörigkeit sieht keine Slots (auch nicht mit `?game_id=`). Admins sehen alles (unverändert).
- **SSE-Reload**: `DutySlotList` triggert via `onReload`-Callback; die Page hält den State. SSE-Subscription bleibt in der Page — kein doppeltes Subscriben.
