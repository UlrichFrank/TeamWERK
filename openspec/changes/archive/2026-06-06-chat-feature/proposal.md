## Why

TeamWERK hat bisher keinen direkten Kommunikationskanal zwischen Vereinsmitgliedern. Kommunikation läuft extern über WhatsApp oder Signal, was bedeutet, dass Absprachen zu Trainings, Spielen und Diensten außerhalb der Plattform stattfinden. Ein integrierter Chat erhöht die Relevanz von TeamWERK als einziger Anlaufpunkt für den Vereinsalltag.

## What Changes

- Neue Chat-Funktion mit drei Kommunikationstypen: Direct (1:1), Gruppe (N:N), Broadcast (1:N einweg)
- Neues Package `internal/chat/` mit vollständigem Backend
- 6 neue Datenbanktabellen (eine Migration)
- Erweiterung des bestehenden SSE-Hubs um per-User-Delivery
- Neuer SSE-Endpoint `/api/chat/events` für Echtzeit-Nachrichten
- Neue Frontend-Route `/chat` mit Konversationsliste und Chat-View
- Nav-Badge für ungelesene Nachrichten
- Push Notifications für neue Nachrichten (nutzt bestehende Infrastruktur in `internal/notifications/`)

## Capabilities

### New Capabilities

- `chat-konversationen`: Direkt- und Gruppenchats — erstellen, Nachrichten senden/empfangen, als gelesen markieren, Gruppen verlassen. Echtzeit via SSE. Sichtbarkeit der Gesprächspartner ist rollenbasiert.
- `chat-broadcasts`: Einweg-Mitteilungen von admin/vorstand/trainer an definierte Zielgruppen (alle, Team, Rolle). Empfänger sind anonym, kein Rückkanal. Echtzeit via SSE + Push.
- `chat-push-notifications`: Push Notifications für neue Chat-Nachrichten und Broadcasts. Nutzt bestehende Web-Push-Infrastruktur.

### Modified Capabilities

- `sse-live-updates`: Der SSE-Hub wird um User-aware Delivery erweitert (`SubscribeUser`, `BroadcastToUser`). Bestehende globale Broadcast-API bleibt unverändert.

## Impact

- **Backend**: Neues Package `internal/chat/`, Erweiterung `internal/hub/hub.go`
- **Datenbank**: 1 neue Migration mit 6 Tabellen (`conversations`, `conversation_members`, `messages`, `message_reads`, `broadcasts`, `broadcast_reads`)
- **API**: 10 neue Routen unter `/api/chat/`
- **Frontend**: Neue Seite `web/src/pages/ChatPage.tsx`, Nav-Eintrag in `AppShell.tsx`, Hook `useChatEvents`
- **Push**: Bestehende `internal/notifications/` wird für Chat-Nachrichten genutzt — kein neuer externer Dienst
