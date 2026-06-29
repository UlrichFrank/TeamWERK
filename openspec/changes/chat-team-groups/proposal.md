## Why

Im „Neues Gespräch"-Modal (`/chat`, Tab „Gruppe") müssen Mitglieder bisher jeden Empfänger einzeln aus der Personen-Suche heraussuchen. Für typische Konstellationen — „alle Trainer des Teams", „alle Spieler", „alle Eltern" — bedeutet das viele Klicks und ist fehleranfällig (jemanden vergessen). Eine vorgeschlagene Auswahl pro Team × Rolle würde das massiv beschleunigen und gleichzeitig zur bestehenden Broadcast-Targeting-Logik passen.

## What Changes

- Neuer Picker-Bereich „Standard-Gruppen" im `NewConversationModal` (Tab „Gruppe") oberhalb der Personen-Liste, gefüllt mit (Trainer / Spieler / Eltern) je sichtbarem Team
- Klick auf eine Standard-Gruppe löst die Gruppe **sofort im Client zu Einzelpersonen auf** und fügt sie in die `selected[]`-Liste ein (dedup nach User-ID). Es entsteht keine neue Konversations-Klasse; am Ende werden nur User-IDs an den bestehenden `POST /api/chat/conversations` geschickt.
- Neuer Backend-Endpoint `GET /api/chat/team-groups` (Liste sichtbarer Tags mit Counts, gefiltert auf **aktive Saison**) und `GET /api/chat/team-groups/{teamId}/{kind}/members` (User-IDs+Namen)
- Sichtbarkeit: Vorstand / sportliche_leitung / admin sehen alle Teams × 3 Kinds; alle anderen nur die Teams aus ihrer `user_accessible_teams`-View. Der Caller selbst wird beim Auflösen weggefiltert (er wird beim `createGroup` ohnehin automatisch hinzugefügt).
- „Spieler"-Auflösung enthält sowohl `kader_members` als auch `kader_extended_members`. „Eltern"-Auflösung enthält die Eltern (`family_links`) beider Mitgliedstypen.

## Capabilities

### New Capabilities

- `chat-team-groups`: Server stellt sichtbare Team×Rolle-Gruppen mit Mitgliederzahlen und Mitgliederlisten bereit; Frontend nutzt sie als Bulk-Selector im Gespräch-Modal.

Konversations-Erstellung bleibt unverändert (`POST /api/chat/conversations` akzeptiert weiterhin nur `memberIds`); die Standard-Gruppen werden client-seitig aufgelöst.

## Impact

- **Backend:** Zwei neue read-only Endpoints in `internal/chat/handler.go`, keine Schemaänderung, keine neuen Migrations.
- **Frontend:** Erweiterung des bestehenden `NewConversationModal` in `web/src/pages/ChatPage.tsx` um den Picker-Block. Keine neuen Dependencies.
- **Berechtigungen:** Bestehender `canContactUser`-Check bei `createGroup` schützt weiterhin: auch wenn ein böser Client manuell User-IDs schickt, lehnt der Server ab.
- **Saison:** Picker zeigt nur Teams der aktiven Saison. Alt-Saison-Trainer können via Personen-Suche weiter ausgewählt werden.
