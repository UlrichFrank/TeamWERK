## Context

Chat-Konversationen und -Nachrichten sind bereits serverseitig in SQLite gespeichert. Zwei Lücken existieren:

1. **Cross-Device-Sync**: `SendMessage`, `EditMessage` und `DeleteMessage` senden SSE-Events nur an *andere* Mitglieder (`activeMembers(…, claims.UserID)` schließt den Sender aus). Öffnet der gleiche User die App auf Gerät B, bekommt er keine Echtzeit-Updates für eigene Aktionen.
2. **Leave-Notification**: `LeaveConversation` setzt `left_at` und antwortet mit 204, sendet aber kein SSE-Event und legt keine System-Nachricht an. Verbleibende Mitglieder erfahren vom Austritt erst beim nächsten manuellen Reload.

## Goals / Non-Goals

**Goals:**
- Eigene SSE-Sessions des Senders erhalten bei Nachricht senden/bearbeiten/löschen dasselbe Event wie andere Mitglieder.
- Beim Verlassen einer Gruppe erscheint im Chatverlauf eine System-Nachricht ("X hat die Gruppe verlassen"), und alle verbleibenden Mitglieder bekommen ein SSE-Event.

**Non-Goals:**
- Keine Offline-Persistenz / Service-Worker-Cache für Chat-Nachrichten.
- Keine Paginieriung für ältere Nachrichten (bleibt bei 100 Nachrichten).
- Kein Multi-Device-Read-Status-Sync (unreadCount ist bereits serverseitig korrekt).

## Decisions

### 1 — SSE-Event auch an Sender senden

`activeMembers(r, convID, excludeUserID)` ausschließen und stattdessen alle aktiven Mitglieder einbeziehen. Da `BroadcastToUser` alle offenen SSE-Connections eines Users anspricht (inkl. des aktuellen Tabs), empfängt auch der Sender das Event auf allen Geräten.

**Alternative betrachtet:** Separater Event-Typ `chat:own-message:{convId}` nur für den Sender. Verworfen — unnötige Komplexität, Frontend-Code muss ohnehin beide Fälle gleich behandeln.

### 2 — System-Nachricht via neues `is_system`-Feld

Um "X hat die Gruppe verlassen" im Chatverlauf sauber darzustellen, wird ein `is_system BOOLEAN NOT NULL DEFAULT 0` über `ALTER TABLE messages ADD COLUMN` hinzugefügt. Der Sender der System-Nachricht ist der austretende User (gültiger FK). Das Frontend rendert `is_system = true`-Nachrichten als zentriertes graues Label statt als Sprechblase.

**Alternative betrachtet:** Body-Konvention `__system__:…` ohne Schema-Änderung. Verworfen — fragil, schwer zu validieren, Body-CHECK würde keine Sonderzeichen erzwingen.

**Alternative betrachtet:** Separates `system_events`-Table. Verworfen — overhead für diesen einfachen Use-Case; `messages` ist der natürliche Ort.

`ALTER TABLE messages ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT 0;` ist in SQLite ohne Tabellen-Neubau möglich (seit SQLite 3.1.3, verfügbar auf dem VPS).

### 3 — SSE-Event-Name für Austritt

`chat:member-left:{convId}` als dediziertes Event. Das Frontend kann so gezielt nur die betroffene Konversation aktualisieren (Teilnehmerliste neu laden, ggf. Nachrichten-Liste für die System-Meldung).

## Risks / Trade-offs

- **Sender bekommt eigenes Event**: Der aktuelle Tab des Senders erhält das SSE-Event und lädt Konversationen neu — bisher wurde das vom Frontend nach dem HTTP-Response selbst erledigt. Doppeltes Reload-Trigger möglich. Mitigation: Das Frontend ignoriert redundante Reloads (React dedupliziert State-Updates bei gleichem Ergebnis).
- **is_system Column**: Bestehende Message-Queries müssen das neue Feld in der SELECT-Liste und im Serialisierungs-Struct ergänzen. Mitigation: Alle Message-Queries sind in `handler.go` zentralisiert; der Compile-Fehler zeigt fehlende Stellen.

## Migration Plan

1. Migration `026_chat_system_messages.up.sql`: `ALTER TABLE messages ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT 0;`
2. Deploy: Binary enthält Migration; `make deploy` führt `migrate up` automatisch aus.
3. Rollback: `026_chat_system_messages.down.sql` mit `-- no down migration` (ALTER ADD COLUMN ist in SQLite nicht rückgängig zu machen; Werte sind alle 0, also kein Datenverlust).
