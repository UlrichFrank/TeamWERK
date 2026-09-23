## Why

Der „Löschen"-Knopf für Chat-Nachrichten (Desktop-Kontextmenü und Mobile-Long-Press-Overlay) hat aktuell keinerlei Rückfrage: ein Klick/Tap löscht sofort und unwiderruflich per Soft-Delete für **alle** Konversationsmitglieder (`internal/chat/handler.go` `DeleteMessage`, Frontend `deleteMsg` in `web/src/pages/ChatPage.tsx`). Ein Fehlklick — gerade auf Mobile, wo der Button in einem dicht gepackten Action-Overlay direkt neben „Antworten"/„Bearbeiten" sitzt — entfernt die Nachricht ersatzlos für die ganze Gruppe. Signal und WhatsApp lösen genau dieses Problem mit einem Bestätigungsdialog, der zusätzlich zwischen „Für mich löschen" (nur die eigene Ansicht) und „Für alle löschen" (nur für den Absender/Admin verfügbar, wirkt bei allen) unterscheidet.

## What Changes

- Klick/Tap auf „Löschen" öffnet einen Bestätigungsdialog statt sofort zu löschen — auf Desktop (Kontextmenü) und Mobile (Long-Press-Overlay) einheitlich, im Modal-Stil des Projekts (kein natives `confirm()`).
- Der Dialog bietet **zwei** Optionen, wenn die Nachricht für alle löschbar ist (eigene Nachricht oder Admin): „Für mich löschen" und „Für alle löschen" — sonst nur „Für mich löschen". Plus „Abbrechen".
- **Neu: „Für mich löschen"** — jedes aktive Konversationsmitglied kann jede sichtbare (nicht bereits gelöschte) Nachricht aus der **eigenen** Ansicht entfernen, ohne dass andere Mitglieder etwas davon merken. Umgesetzt als neue, private Ausblenden-Tabelle (`message_hides`, analog zum bestehenden `broadcast_reads.hidden_at`-Muster), nicht als Soft-Delete.
- **„Für alle löschen"** bleibt fachlich unverändert (Soft-Delete via `deleted_at`, nur Absender/Admin, Platzhalter „Nachricht gelöscht" für alle) — bekommt nur die neue Bestätigung davor.
- Ausgeblendete Nachrichten verschwinden für den ausblendenden Nutzer aus Nachrichtenliste, Suche, Ungelesen-Zähler/Badge und der Konversations-Vorschau (`LastMessage`); für alle anderen Mitglieder bleiben sie unverändert sichtbar.
- Multi-Geräte-Sync: Blendet ein Nutzer eine Nachricht auf einem Gerät aus, verschwindet sie auch in anderen offenen Sessions desselben Nutzers (eigener SSE-Kanal, kein Broadcast an Dritte).

## Capabilities

### New Capabilities
(keine — die Funktionalität erweitert die bestehende Capability unten)

### Modified Capabilities
- `chat-message-delete`: Löschen erfordert jetzt eine Bestätigung; zusätzlich zum bestehenden „Für alle löschen" (Absender/Admin, Soft-Delete) gibt es „Für mich löschen" (jedes aktive Mitglied, private Ausblendung ohne Wirkung für andere).

## Impact

- **Backend:** `internal/chat/handler.go` (`DeleteMessage` unverändert in der Semantik, neuer Handler `HideMessage`), `internal/chat/handler.go` `messageSelect`/`ListMessages`-Varianten, `internal/chat/search.go` (`Search`), `internal/chat/unread.go` (`ComputeUnreadForUser`), Konversations-Vorschau-Query (`LastMessage`). Neue Migration `061_message_hides` (Tabelle + Index). Neue Route `POST /api/chat/messages/{id}/hide` in `internal/app/router.go`.
- **Frontend:** `web/src/pages/ChatPage.tsx` — neuer Bestätigungsdialog (Desktop + Mobile), `deleteMsg` wird durch getrennte `hideMessageForMe`/`deleteMessageForEveryone`-Aktionen ersetzt, `canDelete` wird zu einer reinen Sichtbarkeits-Frage (Dialog immer zeigbar), neue `canDeleteForEveryone`-Prüfung für die zweite Option.
- **Tests:** Happy-Path + Fehlerfall für die neue Route (`internal/chat/*_test.go`), Anpassung bestehender `DeleteMessage`-/`ListMessages`-Tests um die neue Ausblenden-Dimension.
