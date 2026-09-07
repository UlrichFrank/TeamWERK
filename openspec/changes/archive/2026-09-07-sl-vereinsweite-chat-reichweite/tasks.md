## 1. Ein Prädikat für die vereinsweite Chat-Reichweite

- [x] 1.1 `hasGlobalTeamGroupAccess` in `internal/chat/team_groups.go` zu `hasClubWideChatReach` umbenennen (`admin` ∪ `vorstand` ∪ `sportliche_leitung`), Doc-Kommentar: warum Gruppensicht und Kontaktreichweite zusammenfallen müssen
- [x] 1.2 `canContactUser` (`internal/chat/handler.go`) Schritt (1) auf das Prädikat umstellen
- [x] 1.3 `Users`-Suche (`GET /api/chat/users`) auf dasselbe Prädikat umstellen — Zweig-Auswahl und der vorgelagerte `callerInTrainerCircle`-Guard

## 2. Tests

- [x] 2.1 `TestCreateGroup_SLDarfFremdeKaderGruppeAlsGruppeAnlegen`: fremde Spieler-Gruppe auflösen, IDs als `memberIds` posten → 201 (prüft die Kette Auflösen → Anlegen, nicht nur den Endzustand)
- [x] 2.2 `TestCreateGroup_SpielerDarfTeamfremdeNichtGruppieren`: Gegenprobe, reiner Spieler → 403
- [x] 2.3 `TestChatUsers_SLFindetTeamfremdeNutzer`: Suche der sportlichen Leitung findet einen teamfremden Spieler

## 3. Spec

- [x] 3.1 Delta für `chat-konversationen` (beide betroffenen Requirements als MODIFIED, Szenario-Namen stabil)
- [ ] 3.2 Nach dem Merge archivieren
