# Design

## Warum Bugfix statt neuer Komponente

Der bestehende `SpieltagDetailModal` (`web/src/components/SpieltagDetailModal.tsx`) enthält bereits das komplette Slot-Management:

- `showAddSlot`-State + Formular (Duty-Type, Zeit, Slots, Audience) → `POST /api/games/{id}/duty-slots`
- `editSlot`-State + Formular → `PUT /api/duty-slots/{id}`
- `deleteSlotId`-State + Confirm → `DELETE /api/duty-slots/{id}`

Alles ist an `canEdit` gebunden. `canEdit` wird berechnet aus `game?.can_edit === true` — aber diese Property gibt es im API-Response nicht. `GET /api/games/{id}` liefert (seit dem Policy-Refactor auf `internal/policy/annotate.go`):

```json
{
  "game": {
    ...
    "can": { "edit": true, "delete": true, "manage_lineup": true }
  }
}
```

`can_edit` ist **kein Alias**. Das Frontend liest ins Leere → `canEdit === false` für **jeden**. Der Fix ist einzeilig:

```diff
- const canEdit = game?.can_edit === true
+ const canEdit = game?.can?.edit === true
```

Zusätzlich muss die `GameDetail`-Typdefinition (`SpieltagDetailModal.tsx:11–24`) das `can`-Feld statt `can_edit` deklarieren, damit TypeScript den Bug künftig fängt.

Ein zweites, kompakteres Modal (Option 3b aus der Exploration) einzuführen wäre reine Duplikation — der `SpieltagDetailModal` ist genau dafür gebaut, wird bereits von `KalenderPage` und `TerminePage` konsumiert, und dort geändert würde die Behebung auch dort greifen.

## „In Diensten öffnen" — bewusst ohne Focus-Parameter

Die Termine-Seite kennt `?focus=game-<id>` und scrollt zum Eintrag. `/dienste` (`DutyPage`) hat kein solches Pattern. Zwei Optionen:

| Variante | Aufwand | UX |
|---|---|---|
| a) `navigate('/dienste')` ohne Focus | trivial | Nutzer landet auf der gruppierten Liste, muss ggf. scrollen |
| b) `?focus=game-<id>` + Scroll/Highlight in `DutyPage` einführen | Frontend-Änderung an DutyPage + neuer Query-Param | einheitlich mit Termine |

Wir gehen mit (a): Die Dienstbörse gruppiert bereits pro Spiel; der Sprung zur Seite ist der eigentliche Wert, das Scrollen ist trivial. Ein Deep-Link kann später sauber als eigener Change nachgezogen werden, ohne (a) zu invalidieren.

## Sichtbarkeit der Header-Icons vs. Footer-Buttons

- **Header-Icons oben rechts** (Pencil, ClipboardList) öffnen **Bearbeitungs-UI**. Sie erscheinen nur, wenn der Nutzer bearbeiten darf. Bereits so gehandhabt für `onEdit`; wir ziehen `onDienste` in dieselbe Bedingung.
- **Footer-Buttons unten** (In Terminen öffnen, In Diensten öffnen) sind **Navigation**. Sie erscheinen für alle eingeloggten Nutzer. „In Diensten öffnen" ist zusätzlich `disabled`, wenn das Spiel keine Slots hat — sonst würde die Navigation ins Leere führen (der Nutzer sieht auf `/dienste` keinen Eintrag zu diesem Spiel).

Diese Trennung ist auch semantisch sauber: Header = In-Place-Aktion, Footer = Sprung.

## Slot-Count-Signal

Das `EventInfoModal` bekommt `game.slot_count` bereits über das existierende `Game`-Interface (`slot_count?: number`). Für die neue „In Diensten öffnen"-Bedingung reicht `(game.slot_count ?? 0) > 0`. Kein neues API-Feld nötig.

## Nicht adressiert

- **Ausrufezeichen an der „N offene Dienst-Slots"-Zeile im Modal:** nicht Teil dieses Changes. Das Warn-Signal bleibt an den Notizen (`EventNoteIndicator` inline) verankert. Die Kachel im Kalender behält ihr kombiniertes Warn-Icon (Notiz ODER offene Slots) unverändert.
- **Trainings:** Trainings haben kein Dienst-Konzept. Der neue Button erscheint nicht für `type === 'training'`.
- **Absences:** Der neue Button erscheint nicht für `type === 'absence'` (kein „In Diensten öffnen" möglich).
