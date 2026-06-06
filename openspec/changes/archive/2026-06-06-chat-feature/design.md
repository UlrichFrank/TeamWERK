## Context

TeamWERK hat bisher nur einen globalen SSE-Hub (`internal/hub/`) der Mutations-Events an alle verbundenen Clients broadcastet. Für Chat wird ein Delivery-Mechanismus benötigt der Nachrichten gezielt an einzelne User zugestellt — ohne externe Message-Broker (kein Redis, kein NATS). Der VPS hat 1 GB RAM und eine überschaubare Nutzerzahl (~50 aktive User).

## Goals / Non-Goals

**Goals:**
- Direkt- und Gruppenchats mit Echtzeit-Delivery via SSE
- Einweg-Broadcasts mit rollenbasierter Zielgruppensteuerung
- Push Notifications für Offline-User (bestehende Infrastruktur)
- Ungelesen-Badge im Nav
- Rollenbasierte Sichtbarkeit beim User-Picker

**Non-Goals:**
- Typing-Indikatoren, Lese-Bestätigungen pro Nachricht ("Gelesen von X")
- Dateianhänge in Nachrichten
- Nachrichtensuche
- Ende-zu-Ende-Verschlüsselung
- Nachrichtenbearbeitung oder -löschung

## Decisions

### Entscheidung 1: User-aware Hub statt zweitem Hub

**Gewählt:** Den bestehenden `EventHub` um eine `userClients map[int]map[chan string]struct{}` erweitern.

**Warum:** Ein zweiter Hub würde duplizierte Keepalive-Logik und einen zweiten SSE-Endpoint mit gleicher Infrastruktur erfordern. Die Erweiterung des bestehenden Hubs hält die Keepalive-Logik zentral, und der neue `/api/chat/events` Endpoint kann dieselbe Flush/Ping-Struktur nutzen. Der globale Broadcast bleibt unverändert.

**Alternative verworfen:** Separater `ChatHub` struct — mehr Code, gleiche Funktionalität.

```
hub.SubscribeUser(userID int) chan string
hub.UnsubscribeUser(userID int, ch chan string)
hub.BroadcastToUser(userID int, event string)
```

### Entscheidung 2: Zwei Paradigmen, zwei Tabellen-Cluster

**Gewählt:** `conversations`/`conversation_members`/`messages`/`message_reads` für interaktiven Chat, separate `broadcasts`/`broadcast_reads` für Einweg-Mitteilungen.

**Warum:** Broadcasts haben kein Konzept von Mitgliedschaft, Verlassen oder bidirektionaler Kommunikation. Eine unified `conversations`-Tabelle mit `type=broadcast` würde `conversation_members` für Broadcasts sinnlos machen oder komplizierte NULL-Logik erfordern.

**Alternative verworfen:** Unified Conversations — führt zu leeren `conversation_members` bei Broadcasts und ungenutzten JOIN-Feldern.

### Entscheidung 3: Soft-Delete beim Verlassen von Gruppen

**Gewählt:** `conversation_members.left_at DATETIME NULL` statt hartem DELETE.

**Warum:** Nachrichten die vor dem Austritt gesendet wurden, sollen dem ausgetretenen User noch angezeigt werden können (Nachrichtenverlauf bis zum Austritt lesbar). Zudem erlaubt `left_at` späteres Wiedereintreten.

### Entscheidung 4: Direct-Konversation Deduplizierung via SQL

**Gewählt:** Vor dem Erstellen einer Direct-Konversation prüft der Server ob bereits eine existiert (JOIN auf `conversation_members` mit beiden User-IDs).

**Warum:** Verhindert doppelte Direct-Konversationen ohne Unique-Constraint (da SQLite keine Constraints über aggregierte Mengen unterstützt).

### Entscheidung 5: Broadcast-Zielgruppen werden zur Sendzeit aufgelöst

**Gewählt:** `broadcasts.target_type` + `target_id` speichern die Zielgruppe als Referenz. Beim Senden werden die matching User-IDs live aus der DB ermittelt (`broadcast_reads` + Push werden zur Sendzeit befüllt/ausgelöst).

**Warum:** Dynamische Auflösung bedeutet kein Staleness-Problem wenn Team-Mitgliedschaften sich nach dem Senden ändern. Für Read-Tracking werden alle passenden User-IDs zur Sendzeit in `broadcast_reads` eingetragen.

**Alternative verworfen:** Snapshot der Empfänger-IDs in einer Hilfstabelle — unnötige Komplexität für die Nutzerzahl.

### Entscheidung 6: Sichtbarkeits-Filterung im User-Picker Endpoint

**Gewählt:** `GET /api/chat/users?q=...` gibt rollenbasiert gefilterte User zurück. Die Filterlogik liegt im Backend, nicht im Frontend.

```
spieler/elternteil → WHERE team_id IN (eigene Teams)
trainer            → WHERE team_id IN (Teams wo Trainer Mitglied ist)
vorstand/admin     → alle User ohne Filter
```

**Warum:** Sicherheitsrelevant — ein Spieler darf User aus anderen Teams nicht sehen oder anschreiben. Frontend-seitiger Filter wäre umgehbar.

## Risks / Trade-offs

- **SSE-Verbindungen pro User** → Mehrere offene Tabs = mehrere Channels im Hub. Bei 50 Usern à 3 Tabs = 150 Channels, vernachlässigbar für den VPS.
- **SQLite WAL und parallele Writes** → Chat ist write-intensiver als der Rest der App. WAL-Mode ist bereits aktiv und für diesen Nutzungsumfang ausreichend. Bei deutlichem Wachstum wäre PostgreSQL der nächste Schritt.
- **Keine Message-Pagination initial** → Die ersten 100 Nachrichten werden geladen (`LIMIT 100 ORDER BY sent_at DESC`). Ältere Nachrichten sind vorerst nicht abrufbar. Pagination kann bei Bedarf nachgerüstet werden.
- **Push Notifications auf iOS nur bei PWA** → Bekannte Einschränkung der bestehenden Push-Infrastruktur. Keine Änderung nötig.

## Migration Plan

1. Migration `025_chat.up.sql` anlegen — alle 6 Tabellen in einer Migration
2. Kein Daten-Backfill nötig — alle Tabellen sind neu
3. Rollback via `025_chat.down.sql` (DROP TABLE in reverse order)
4. Deploy: `make deploy` führt automatisch `migrate up` aus

## Open Questions

- Sollen ausgetretene Gruppen-Mitglieder die Nachrichten nach ihrem Austritt noch sehen? (Aktueller Stand: nein — `left_at` begrenzt die Sichtbarkeit)
- Maximale Nachrichtenlänge? (Vorschlag: 2000 Zeichen, CHECK-Constraint)
