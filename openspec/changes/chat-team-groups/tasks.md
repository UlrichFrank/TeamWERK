## 1. Backend: TeamGroups-Endpoints

- [x] 1.1 `GET /api/chat/team-groups` in `internal/chat/handler.go`: listet sichtbare `{teamId, teamName, kind, count}`-Tags. Sichtbarkeit gemäß D3, Saison-Filter gemäß D2, Caller aus Counts ausgeschlossen.
- [x] 1.2 `GET /api/chat/team-groups/{teamId}/{kind}/members` in `internal/chat/handler.go`: liefert `[{id, name}, …]` für die aufgelöste Gruppe. Zugriff geprüft gegen Sichtbarkeitsregel; 403 bei Verstoß. Caller wird gefiltert.
- [x] 1.3 Helper `teamGroupVisibility(claims, teamID) bool` einführen und an beiden Endpoints nutzen, damit die Regel an einer Stelle steht.
- [x] 1.4 Routes in `internal/app/router.go` unter `authenticated`-Gruppe registrieren.

## 2. Backend: Tests

- [x] 2.1 Happy-Path: Spieler eines Teams sieht in `GET /chat/team-groups` genau drei Tags (Trainer/Spieler/Eltern) für sein Team mit korrekten Counts (Caller ausgeschlossen).
- [x] 2.2 Vorstand sieht alle Teams × 3 Tags der aktiven Saison.
- [x] 2.3 Sportliche Leitung sieht alle Teams × 3 Tags der aktiven Saison.
- [x] 2.4 Trainer sieht nur die Teams, in denen er Trainer ist (×3).
- [x] 2.5 Inaktive Saison-Teams werden ausgeblendet.
- [x] 2.6 `GET /chat/team-groups/{team}/spieler/members` enthält sowohl `kader_members` als auch `kader_extended_members`.
- [x] 2.7 `GET /chat/team-groups/{team}/eltern/members` enthält Eltern beider Spieler-Quellen.
- [x] 2.8 Fehlerfall: Spieler ruft `/chat/team-groups/{fremdesTeam}/spieler/members` → HTTP 403.
- [x] 2.9 Fehlerfall: ungültiger `kind`-Wert → HTTP 400.

## 3. Frontend: Picker im NewConversationModal

- [x] 3.1 `ChatPage.tsx`: neuer Typ `TeamGroup = { teamId; teamName; kind: 'trainer'|'spieler'|'eltern'; count }`.
- [x] 3.2 Im `NewConversationModal` (nur Tab `group`): laden via `GET /chat/team-groups` beim Öffnen.
- [x] 3.3 Picker-UI: oberhalb der Personen-Suche ein Abschnitt „Standard-Gruppen" (gefiltert durch dieselbe `query`-State wie Personen).
- [x] 3.4 Klick auf Tag: `GET /chat/team-groups/{teamId}/{kind}/members`, Result mit `selected[]` mergen (Set-Semantik nach `id`).
- [x] 3.5 Personen, die schon in `selected[]` sind, werden im Personen-Picker als markiert dargestellt (bestehend) — Tags werden nach Klick aus der Liste ausgeblendet, um Doppel-Klick-Loops zu vermeiden.
- [x] 3.6 Submit-Pfad unverändert: `POST /chat/conversations` mit `memberIds` aus `selected.map(s => s.id)`.

## 4. Verify

- [x] 4.1 `go vet ./...`, `go test ./...`, `golangci-lint run` grün.
- [x] 4.2 `pnpm -C web build` grün.
- [x] 4.3 `openspec validate chat-team-groups --strict` grün.
