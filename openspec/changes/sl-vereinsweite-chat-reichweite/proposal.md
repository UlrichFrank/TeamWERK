## Why

Die sportliche Leitung sieht in „Nachrichten" bereits die Standardgruppen **aller** Teams der aktiven Saison (`hasGlobalTeamGroupAccess` in `chat-team-groups` nennt `admin`, `vorstand` und `sportliche_leitung`) und darf sie auch auflösen. Benutzbar ist diese Sicht aber nicht: `canContactUser` lässt in Schritt (1) nur `admin` und `vorstand` pauschal durch. Wählt die sportliche Leitung im Modal „Neues Gespräch" die Kader-Gruppe eines fremden Teams, kommen die Mitglieder als Chips an, das anschließende `POST /api/chat/conversations` scheitert dann aber für **jedes** teamfremde Mitglied mit HTTP 403 — die Gruppe entsteht gar nicht, sichtbar nur als „Fehler beim Erstellen".

Dieselbe Lücke betrifft die Nutzersuche: `GET /api/chat/users` liefert der sportlichen Leitung nur Team-Overlap ∪ Zugriffskreis ∪ `chat_visible`. Ein teamfremder Spieler taucht als Gruppen-Chip auf, ist über die Suche daneben aber nicht auffindbar.

## What Changes

- `canContactUser` erlaubt in Schritt (1) zusätzlich `sportliche_leitung` — damit werden Kader-Gruppen fremder Teams für die sportliche Leitung tatsächlich nutzbar (Gruppe anlegen, Mitglied nachträglich hinzufügen).
- `GET /api/chat/users` behandelt `sportliche_leitung` wie `admin`/`vorstand` (Suche über alle User), damit Suche und Gruppenauflösung dieselbe Reichweite haben.
- Beide Stellen und die bestehende Gruppen-Sichtbarkeit teilen ein Prädikat (`hasClubWideChatReach` statt `hasGlobalTeamGroupAccess`), damit die drei Definitionen nicht wieder auseinanderlaufen.
- Keine Änderung am Zugriffskreis (`callerInTrainerCircle`), an der „Alle Trainer"-Gruppe, an Broadcasts oder am Datenmodell.

## Capabilities

### New Capabilities

_(keine)_

### Modified Capabilities

- `chat-konversationen`: Schritt (1) der Kontaktprüfung und die Nutzersuche schließen `sportliche_leitung` ein.

## Impact

- `internal/chat/team_groups.go` — `hasGlobalTeamGroupAccess` → `hasClubWideChatReach` (ein Prädikat für Gruppensicht, Kontaktprüfung und Suche)
- `internal/chat/handler.go` — `canContactUser` Schritt (1) und `Users`-Suche nutzen das Prädikat
- Frontend unverändert — der Fehlerfall verschwindet, es kommt keine neue UI hinzu

## Nicht-Ziele

- Die **Mitgliedermenge** der Gruppen ändert sich nicht; die sportliche Leitung erscheint weiterhin nur dann in „Alle Trainer", wenn sie selbst Kader-Trainerin ist.
- `vorstand_beisitzer` bleibt beim Zugriffskreis (Schritt 2) und bekommt **keine** vereinsweite Reichweite.

## Test-Anforderungen

| Route | Fall | Erwartung |
|---|---|---|
| `POST /api/chat/conversations` (`type=group`) | `sportliche_leitung` ohne Teamzugehörigkeit, Mitglieder aus der aufgelösten Spieler-Gruppe eines fremden Teams | 201, Gruppe entsteht (`TestCreateGroup_SLDarfFremdeKaderGruppeAlsGruppeAnlegen`) |
| | reiner Spieler von T1, Mitglied `memberIds=[Spieler aus T2]` | 403 (`TestCreateGroup_SpielerDarfTeamfremdeNichtGruppieren`) |
| `GET /api/chat/users` | `sportliche_leitung` ohne Teamzugehörigkeit | Ergebnis enthält einen teamfremden Spieler (`TestChatUsers_SLFindetTeamfremdeNutzer`) |

**Garantierte Invarianten:**

1. **Sicht und Reichweite fallen zusammen.** Was `GET /api/chat/team-groups/{teamId}/{kind}/members`
   für einen Caller mit HTTP 200 auflöst, passiert für denselben Caller auch `canContactUser` —
   der erste Test prüft genau diese Kette (auflösen, dann anlegen) statt nur den Endzustand.
2. **Die Einschränkung für alle anderen bleibt.** Der Gegenprobe-Test hält fest, dass ein
   Nutzer ohne Vereinsfunktion weiterhin an Schritt (3) scheitert; die bestehenden Tests zu
   Zugriffskreis und „Alle Trainer" laufen unverändert weiter.
