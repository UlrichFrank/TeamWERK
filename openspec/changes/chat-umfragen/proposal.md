## Why

Terminabsprachen, Trikotfarben, „wer fährt am Samstag?", „Pizza oder Nudeln beim
Saisonabschluss?" — in den Gruppenchats (vor allem den Team-Gruppen Trainer/Spieler/Eltern)
werden solche Fragen heute als Freitext gestellt und mit 20 Einzelantworten beantwortet, die
anschließend jemand von Hand auszählt. Signal löst das mit **Umfragen** direkt im
Gruppenchat; genau dieses Muster fehlt in `/chat`. Die Chat-Infrastruktur (Nachricht als
Träger, Reaktionen als Muster für „Stimmen pro Nutzer", eigener SSE-Kanal) trägt das mit
überschaubarem Aufwand.

## What Changes

- **Umfrage als Nachrichtentyp in Gruppenkonversationen**: aktive Gruppenmitglieder können
  eine Umfrage anlegen — Frage, 2–10 Antwortoptionen, Schalter „Mehrfachantworten erlauben".
  Die Umfrage erscheint als eigene Karte im Verlauf, zählt wie eine Nachricht für
  Ungelesen-Zähler, Konversationsliste und Push („Umfrage: <Frage>").
- **Abstimmen, ändern, zurückziehen**: jedes aktive Mitglied wählt eine (bzw. bei
  Mehrfachauswahl beliebig viele) Option(en), kann die Wahl jederzeit ändern oder ganz
  zurückziehen, solange die Umfrage offen ist.
- **Nicht anonym** (wie Signal): alle Mitglieder sehen je Option die Stimmenzahl, einen
  Balken und auf Tipp die Namen der Abstimmenden.
- **Beenden**: nur die Erstellerin/der Ersteller beendet eine Umfrage; danach sind keine
  Stimmen mehr möglich, das Ergebnis bleibt sichtbar.
- **Nicht bearbeitbar**: eine versendete Umfrage kann nicht mehr geändert werden (Frage und
  Optionen sind nach der ersten Stimme nicht mehr neutral änderbar). Löschen funktioniert
  wie bei jeder Nachricht (Soft-Delete, Placeholder).
- **Live**: Stimmen und Beenden aktualisieren offene Clients sofort über den Chat-SSE-Kanal;
  Stimmen lösen **keine** Push-Benachrichtigung aus.
- **Nicht im Scope**: Umfragen in Direktchats und in Mitteilungen (Broadcasts), anonyme
  Umfragen, Ablaufdatum/automatisches Beenden, Push an den Ersteller pro Stimme.

## Capabilities

### New Capabilities
- `chat-umfragen`: Anlegen, Abstimmen (Einfach-/Mehrfachauswahl, ändern, zurückziehen),
  nicht-anonyme Ergebnisanzeige, Beenden, Live-Aktualisierung und Darstellung von Umfragen in
  Gruppenkonversationen inkl. Konversationsliste/Push-Text.

### Modified Capabilities
- `chat-message-edit`: Umfrage-Nachrichten sind vom Bearbeiten ausgenommen (Menüeintrag fehlt,
  `PUT /api/chat/messages/{id}` lehnt ab).

## Impact

- **DB**: neue Migration `059_chat_polls` — Tabellen `chat_polls`, `chat_poll_options`,
  `chat_poll_votes`, alle per `ON DELETE CASCADE` an `messages` gehängt. Keine Änderung an
  `messages` selbst (die Frage steht im bestehenden `body`).
- **Backend** (`internal/chat`): neue Routen
  `POST /api/chat/conversations/{id}/polls`, `PUT /api/chat/messages/{id}/poll/vote`,
  `POST /api/chat/messages/{id}/poll/close`, `GET /api/chat/messages/{id}/poll`;
  `ListMessages` liefert pro Umfrage-Nachricht ein `poll`-Objekt; `ListConversations`/
  `getConversation` markieren eine Umfrage als letzte Nachricht; `EditMessage` lehnt Umfragen
  ab. Neues SSE-Event `chat:poll-updated:<convId>:<messageId>`.
- **Frontend** (`web/src/pages/ChatPage.tsx` + neue Komponenten `ChatPollCard`,
  `ChatPollCreateModal`, `ChatPollVotesModal`): Umfrage-Button im Composer (nur Gruppen),
  Karte im Verlauf, Kontextmenü „Umfrage beenden".
- **Gates**: Broadcast-Gate bleibt ohne Allowlist-Eintrag (alle neuen Mutationen fächern über
  `BroadcastToUser` auf); Push-Fan-out-Gate unberührt (neue Umfrage nutzt den bestehenden
  Chat-Pfad `push.SendToUserWithBadge`).
- **RAM/Performance**: vernachlässigbar; Stimmen werden je Nachrichtenseite in einer
  Batch-Query nachgeladen (Muster der Reaktionen).

## Test-Anforderungen

| Route | Test | Erwartet |
|---|---|---|
| `POST /api/chat/conversations/{id}/polls` | `TestCreatePoll_Group_OK` | 201, Nachricht + Umfrage + Optionen in Request-Reihenfolge |
| | `TestCreatePoll_DirectConversation_400` | 400, keine Nachricht angelegt |
| | `TestCreatePoll_OptionCount_400` (1 bzw. 11 Optionen) | 400 |
| | `TestCreatePoll_DuplicateOptions_400` | 400 |
| | `TestCreatePoll_LeftMember_403` | 403 |
| | `TestCreatePoll_PushAndEvent` | Push „Umfrage: <Frage>" an andere Mitglieder, `chat:new-message` an alle |
| `PUT /api/chat/messages/{id}/poll/vote` | `TestVotePoll_SingleChoice_ReplacesVote` | 204, genau eine Stimme nach Wechsel |
| | `TestVotePoll_MultiChoice_OK` | 204, `voterCount` zählt den Nutzer einmal |
| | `TestVotePoll_EmptyWithdraws` | 204, keine Stimme mehr |
| | `TestVotePoll_TwoOptionsOnSingleChoice_400` | 400, Auswahl unverändert |
| | `TestVotePoll_ForeignOption_400` | 400 |
| | `TestVotePoll_Closed_409` | 409, Auswahl unverändert |
| | `TestVotePoll_DeletedMessage_404` | 404 |
| | `TestVotePoll_NonMember_403` | 403 |
| | `TestVotePoll_NoPushEvent` | kein Push, `chat:poll-updated` statt `chat:new-message` |
| `POST /api/chat/messages/{id}/poll/close` | `TestClosePoll_Creator_OK` | 204, `closedAt` gesetzt, Event `chat:poll-updated` |
| | `TestClosePoll_OtherUserAndAdmin_403` | 403 |
| | `TestClosePoll_Idempotent` | 204, `closed_at` unverändert |
| `GET /api/chat/messages/{id}/poll` | `TestGetPoll_Member_OK` | 200, Zahlen/Namen/`voted` korrekt |
| | `TestGetPoll_NonMember_403` / `TestGetPoll_NotAPoll_404` | 403 / 404 |
| `PUT /api/chat/messages/{id}` | `TestEditMessage_Poll_409` | 409, Body unverändert |
| `GET /api/chat/conversations/{id}/messages` | `TestListMessages_IncludesPoll` / `TestListMessages_DeletedPollHasNoPoll` | `poll` gesetzt bzw. `null` |
| `GET /api/chat/conversations` | `TestListConversations_LastMessageIsPoll` | `lastMessage.isPoll = true` |

**Garantierte Invarianten:** Ein Nutzer hat bei Einfachauswahl höchstens eine Stimme je
Umfrage; nach dem Beenden ändert sich keine Stimme mehr; eine Stimme erzeugt nie eine Push-
Benachrichtigung und ändert keinen Ungelesen-Zähler; Frage und Optionen einer versendeten
Umfrage sind unveränderlich.
