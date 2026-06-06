## Why

Die Chat-Funktion hat drei konkrete Nutzungslücken: Broadcast-Mitteilungen sind auf Mobile nicht lesbar (fehlender Detail-Aufruf), es gibt keine Möglichkeit Gespräche oder Mitteilungen aufzuräumen, und Gruppen-Ersteller können nachträglich niemanden mehr einladen. Diese Lücken entstammen direktem Nutzerfeedback.

## What Changes

- **Bug-Fix**: Broadcast-Detailansicht öffnet sich auf Mobile korrekt beim Antippen einer Mitteilung
- **Neu**: Gespräche (Direct + Gruppe) können für sich selbst gelöscht/ausgeblendet werden; löscht der letzte Teilnehmer, werden alle Daten bereinigt
- **Neu**: Broadcast-Mitteilungen können für sich selbst ausgeblendet werden; löscht der letzte Empfänger, wird die Mitteilung bereinigt
- **Neu**: Ersteller eines Gruppen-Chats kann jederzeit weitere Personen hinzufügen (auch nach Verlassen re-adden)
- **Migration**: `broadcast_reads` erhält Spalte `hidden_at DATETIME`

## Capabilities

### New Capabilities

- `chat-loeschen`: Gespräche und Broadcasts für sich selbst ausblenden, mit automatischer Bereinigung wenn alle gelöscht haben
- `chat-mitglieder-hinzufuegen`: Gruppen-Ersteller kann Mitglieder nachträglich zu einer Gruppe hinzufügen

### Modified Capabilities

- `chat-konversationen`: Bestehende Konversations-Logik wird um Lösch-Semantik erweitert (kein Verhaltensbruch, Ergänzung)
- `chat-broadcasts`: Bestehende Broadcast-Logik wird um Lösch-Semantik erweitert

## Impact

- `internal/chat/handler.go`: 3 neue Endpoints, 2 bestehende Queries angepasst
- `internal/db/migrations/`: 1 neue Migration (hidden_at auf broadcast_reads)
- `web/src/pages/ChatPage.tsx`: openBroadcast() Bug-Fix, Lösch-UI (ActionMenu oder Trash2-Icon), AddMember-Modal
- Keine neuen Abhängigkeiten, keine Breaking Changes an bestehenden API-Routen
