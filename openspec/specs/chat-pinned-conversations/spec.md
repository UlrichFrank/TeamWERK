# chat-pinned-conversations Specification

## Purpose
Erlaubt Nutzern, einzelne Chat-Konversationen dauerhaft oben in ihrer eigenen Chat-Liste zu
fixieren (anpinnen) und die Reihenfolge der angepinnten Konversationen selbst per Drag & Drop
zu bestimmen, statt sich bei jeder neuen Nachricht neu durch die aktivitätssortierte Liste
suchen zu müssen.

## Requirements

### Requirement: Konversation anpinnen und lösen

Das System SHALL es einem aktiven Mitglied einer Konversation (`conversation_members.left_at IS
NULL`) erlauben, die Konversation für sich selbst anzupinnen und ein angepinntes Gespräch
wieder zu lösen. Der Pin-Status SHALL ausschließlich für den handelnden User gelten und für
andere Mitglieder derselben Konversation nicht sichtbar sein.

#### Scenario: Mitglied pinnt eine Konversation an
- **WHEN** ein aktives Mitglied `PUT /api/chat/conversations/{id}/pin` aufruft
- **THEN** erscheint die Konversation für diesen User in der gepinnten Gruppe oben in der Liste

#### Scenario: Mitglied löst eine gepinnte Konversation
- **WHEN** ein aktives Mitglied `DELETE /api/chat/conversations/{id}/pin` für eine zuvor
  gepinnte Konversation aufruft
- **THEN** erscheint die Konversation wieder in der ungepinnten, nach letzter Nachricht
  sortierten Gruppe

#### Scenario: Pin ist nicht für andere Mitglieder sichtbar
- **WHEN** User A eine gemeinsame Konversation mit User B anpinnt
- **THEN** sieht User B in seiner eigenen Konversationsliste keine Änderung — die Konversation
  bleibt für User B an ihrer bisherigen, aktivitätsbasierten Position

#### Scenario: Kein Zugriff ohne aktive Mitgliedschaft
- **WHEN** ein User, der kein aktives Mitglied der Konversation ist (nie Mitglied oder bereits
  ausgetreten, `left_at IS NOT NULL`), `PUT /api/chat/conversations/{id}/pin` aufruft
- **THEN** antwortet das System mit 403 oder 404, ohne den Pin-Status zu ändern

### Requirement: Gepinnte Konversationen erscheinen sortiert oben

Das System SHALL bei `GET /api/chat/conversations` alle gepinnten Konversationen des
anfragenden Users vor allen ungepinnten anzeigen. Innerhalb der gepinnten Gruppe SHALL die
manuell festgelegte Reihenfolge gelten, nicht der Zeitpunkt der letzten Nachricht. Die
ungepinnte Gruppe SHALL wie bisher nach dem Zeitpunkt der letzten Nachricht absteigend sortiert
sein.

#### Scenario: Gepinnte Konversation bleibt trotz neuer Nachricht in ungepinnten Chats oben
- **WHEN** eine Konversation gepinnt ist und in einer anderen, ungepinnten Konversation eine
  neue Nachricht eintrifft
- **THEN** bleibt die gepinnte Konversation weiterhin über allen ungepinnten Konversationen,
  unabhängig davon, wie aktuell deren letzte Nachricht ist

#### Scenario: Neue Nachricht in einer gepinnten Konversation ändert ihre Position innerhalb der Pins nicht
- **WHEN** in einer von mehreren gepinnten Konversationen eine neue Nachricht eintrifft
- **THEN** bleibt die manuell festgelegte Reihenfolge der gepinnten Gruppe unverändert (keine
  automatische Neusortierung nach Aktivität innerhalb der Pins)

### Requirement: Manuelles Umsortieren gepinnter Konversationen

Das System SHALL es einem User erlauben, die Reihenfolge seiner eigenen gepinnten
Konversationen durch Übergabe der vollständigen neuen Ziel-Reihenfolge zu ändern. Die
übergebene Menge an Konversations-IDs MUSS exakt der aktuellen Menge der gepinnten
Konversationen dieses Users entsprechen.

#### Scenario: Erfolgreiches Umsortieren
- **WHEN** ein User mit drei gepinnten Konversationen A, B, C eine neue Reihenfolge C, A, B
  über `PUT /api/chat/conversations/pinned-order` sendet
- **THEN** liefert `GET /api/chat/conversations` die gepinnte Gruppe künftig in der Reihenfolge
  C, A, B

#### Scenario: Reorder mit abweichender ID-Menge wird abgelehnt
- **WHEN** die übergebene ID-Liste eine Konversation enthält, die für diesen User nicht (mehr)
  gepinnt ist, oder eine aktuell gepinnte Konversation fehlt
- **THEN** antwortet das System mit 409 und ändert keine Reihenfolge

### Requirement: Neu angepinnte Konversation erscheint am Ende der gepinnten Gruppe

Das System SHALL eine neu angepinnte Konversation standardmäßig ans Ende der bestehenden
gepinnten Reihenfolge dieses Users anhängen, nicht an den Anfang.

#### Scenario: Zweite gepinnte Konversation erscheint nach der ersten
- **WHEN** ein User zunächst Konversation A und danach Konversation B anpinnt
- **THEN** liefert `GET /api/chat/conversations` die gepinnte Gruppe in der Reihenfolge A, B
