## Why

Beim Klick auf einen Kalendereintrag in `/kalender` öffnet sich das `EventInfoModal`. Das Modal integriert Dienste heute unvollständig:

1. Der **„Dienste"-Button oben rechts** (ClipboardList, `onDienste`) öffnet zwar den `SpieltagDetailModal`, in dem Slots angelegt, bearbeitet und gelöscht werden **könnten** — aber die Slot-Edit-UI erscheint dort **nie**, weil das Frontend `game.can_edit` liest, das Backend seit dem Policy-Refactor aber `game.can.edit` liefert (`can_edit` existiert im Response nicht). Für Admin, Vorstand, Trainer und sportliche Leitung ist der Slot-CRUD damit unsichtbar.
2. Es gibt **keine Navigation von einem Kalendereintrag zur Dienstbörse** (`/dienste`), obwohl das für Nicht-Bearbeiter (Spieler, Eltern) der einzige sinnvolle Sprung wäre.
3. Der ClipboardList-Button oben rechts wird **auch Nicht-Bearbeitern angezeigt**, obwohl er ausschließlich Bearbeitungs-UI öffnet — für sie also nutzlos.

## What Changes

- **Bugfix (Frontend):** `SpieltagDetailModal.tsx` liest den Bearbeiten-Flag aus `game.can.edit` (statt `game.can_edit`). Damit erscheint der Slot-CRUD (Hinzufügen/Bearbeiten/Löschen) für Admin, Vorstand, Trainer und sportliche Leitung.
- **Sichtbarkeit oben rechts:** Der ClipboardList-Button in `EventInfoModal` wird nur noch angezeigt, wenn der Nutzer bearbeiten darf — d. h. `KalenderPage` übergibt `onDienste` nur unter der gleichen `canEdit`-Bedingung wie `onEdit`.
- **Neuer Footer-Button „In Diensten öffnen":** Im `EventInfoModal` erscheint für Spiele ein dritter Footer-Button, der zur Dienstbörse `/dienste` navigiert. Sichtbar für **alle** eingeloggten Rollen; **disabled**, wenn das Spiel keine Slots hat (`slot_count === 0`).

Nicht Teil dieses Changes:
- Kein neues Slot-CRUD-Modal — der bestehende `SpieltagDetailModal` erfüllt den Bedarf nach dem Bugfix.
- Kein Deep-Link (`/dienste?focus=…`) — bewusst einfach gehalten, weil die Dienstbörse pro Spiel gruppiert ist und der Nutzer den Eintrag optisch schnell findet.

## Capabilities

### Modified Capabilities

- `event-info-modal`: Ergänzt um den Footer-Button „In Diensten öffnen" (für alle Rollen, bei Spielen ohne Slots disabled) und um die Sichtbarkeits-Regel des ClipboardList-Buttons (nur bei Bearbeitungsrechten).

## Impact

- **Frontend:**
  - `web/src/components/EventInfoModal.tsx` — neuer Footer-Button + zugehörige Prop.
  - `web/src/components/SpieltagDetailModal.tsx` — `canEdit`-Feld auf `game.can.edit` umstellen; `GameDetail`-Typ um `can`-Feld erweitern (`can_edit`-Feld entfernen, es existierte nie im Response).
  - `web/src/pages/KalenderPage.tsx` — `onDienste`-Prop an `EventInfoModal` nur setzen, wenn `canEdit`.
- **Backend:** unverändert. `GET /api/games/{id}` liefert `can.edit` bereits seit dem Policy-Refactor.
- **Test-Coverage:** Handler-Test `TestGetGame_CanEdit_ByRole` wird ergänzt (Happy + Negativ für Admin/Vorstand/Trainer/sL vs. Spieler).
