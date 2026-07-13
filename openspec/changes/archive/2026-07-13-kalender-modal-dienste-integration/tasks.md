## 1. Bugfix: Slot-CRUD im Spiel-Detail-Modal sichtbar

- [x] 1.1 `web/src/components/SpieltagDetailModal.tsx` — `GameDetail`-Interface: `can_edit?: boolean` entfernen, dafür `can?: { edit: boolean; delete: boolean; manage_lineup: boolean }` hinzufügen.
- [x] 1.2 Zeile 54: `const canEdit = game?.can?.edit === true` (statt `game?.can_edit === true`).

## 2. Header-Sichtbarkeit: ClipboardList nur bei Bearbeitungsrechten

- [x] 2.1 `web/src/pages/KalenderPage.tsx` — `onDienste`-Prop nur setzen, wenn `canEdit && infoItem.type === 'game' && infoItem.game`. Bislang wurde die `canEdit`-Bedingung nicht geprüft.

## 3. Neuer Footer-Button „In Diensten öffnen"

- [x] 3.1 `web/src/components/EventInfoModal.tsx` — im `Game`-Interface `slot_count?: number` behalten (existiert bereits).
- [x] 3.2 Im Footer für `type === 'game'` einen dritten Button einfügen, links vom „Schließen"-Button (rechts von „In Terminen öffnen"). Beschriftung: **„In Diensten öffnen"**. Klasse: Primary-Button-Style analog „In Terminen öffnen". Klick: `onClose()` + `navigate('/dienste')`.
- [x] 3.3 Button ist `disabled` wenn `(game.slot_count ?? 0) === 0`. Deaktivierte Optik `disabled:opacity-40 disabled:cursor-not-allowed` (Standard-Buttonklasse).
- [x] 3.4 Nur bei `type === 'game'`. Nicht bei Training oder Absence.

## 4. Backend-Test: `can.edit` je Rolle

- [x] 4.1 Neuer Test `internal/games/handler_test.go` → `TestGetGame_CanEdit_ByRole`:
  - Admin: `can.edit === true`
  - Vorstand: `can.edit === true`
  - Trainer eines beteiligten Teams: `can.edit === true`
  - Sportliche Leitung: `can.edit === true`
  - Spieler ohne Vereinsfunktion: `can.edit === false`

## 5. Verifikation

- [x] 5.1 `go build ./...` fehlerfrei.
- [x] 5.2 `pnpm -C web build` und `pnpm -C web lint` ohne Fehler.
- [x] 5.3 `go test ./internal/games/... -run 'TestGetGame'` grün (inkl. neuem Test).
- [x] 5.4 `openspec validate kalender-modal-dienste-integration --strict` grün.

## Test-Anforderungen

| Route/Verhalten | Test | Erwartung |
|---|---|---|
| `GET /api/games/{id}` als Admin | `TestGetGame_CanEdit_ByRole/admin` | `can.edit == true` |
| `GET /api/games/{id}` als Vorstand | `TestGetGame_CanEdit_ByRole/vorstand` | `can.edit == true` |
| `GET /api/games/{id}` als Trainer eines Team-Spielers | `TestGetGame_CanEdit_ByRole/trainer` | `can.edit == true` |
| `GET /api/games/{id}` als sportliche Leitung | `TestGetGame_CanEdit_ByRole/sportliche_leitung` | `can.edit == true` |
| `GET /api/games/{id}` als reiner Spieler | `TestGetGame_CanEdit_ByRole/spieler` | `can.edit == false` |

**Garantierte Invariante:** Das Frontend erkennt aus `GET /api/games/{id}` korrekt, wer den Slot-CRUD im `SpieltagDetailModal` sehen darf. Das Response-Schema (`game.can.edit`) ist Vertrag zwischen Backend und Frontend, kein Zufall.
