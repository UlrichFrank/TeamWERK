## Why

Der erweiterte Kader ist inzwischen überall gleichgestellt: er sieht die Termine seiner
Mannschaft, steht in den Teilnehmerlisten, ist in den Team-Chatgruppen, taucht in der
Anwesenheitsstatistik auf, bekommt den Kalender-Feed und — seit
`terminmeldungen-erweiterter-kader` — auch alle Terminmeldungen.

**Videos sind der letzte Bereich, der das nicht mitmacht.** `visibilityFilter`
(`internal/videos/crud.go`), `userBelongsToTeam` (`internal/videos/access.go`) und
`pushRecipients` (`internal/videos/worker.go`) lösen ihre Berechtigten über
`player_memberships` auf — eine View über `kader_members`. Ein Spieler des erweiterten
Kaders findet das Video **des Spiels, in dem er selbst gespielt hat**, nicht in seiner
Liste, bekommt es über die Detail-Route nicht (403) und erfährt von seiner Existenz nichts.
Dasselbe gilt für seine Eltern.

Anders als bei den Terminen ist das kein Widerspruch zwischen Sichtbarkeit und Meldung —
beide sind hier gleich eng. Es ist eine Lücke in der Gleichstellung, und sie fällt genau
denen auf, die auf dem Video zu sehen sind.

## What Changes

- Die drei Auflösungen im Videos-Paket nehmen den **erweiterten Kader** und dessen
  **Elternteile** auf: Sichtbarkeit der Liste, Zugriff auf das einzelne Video (inklusive
  Stream) und die Ready-Benachrichtigung.
- **Statusfilter für die neuen Zweige ist `status <> 'ausgetreten'`, nicht `status = 'aktiv'`.**
  Förderkinder tragen `members.status = 'foerderkind'` (Migration `034`) — genau die
  Zielgruppe des erweiterten Kaders. Ein `= 'aktiv'` schlösse sie wieder aus und die
  Änderung liefe für den Hauptfall ins Leere. Die bestehenden Stammkader-Zweige bleiben
  unverändert bei `= 'aktiv'`.
- **Keine Änderung an den Schreibrechten.** Hochladen, Metadaten ändern und Löschen bleiben
  bei Trainer/Vorstand/Admin. Der erweiterte Kader bekommt Lese- und Streamzugriff, sonst
  nichts.

## Capabilities

### Modified Capabilities

- `video-management`: Die Sicht-Berechtigung pro Team umfasst den erweiterten Kader und
  dessen Elternteile.
- `video-transcode`: Der Empfängerkreis der Ready-Benachrichtigung umfasst den erweiterten
  Kader und dessen Elternteile.

## Impact

- `internal/videos/access.go` — `userBelongsToTeam` (deckt `CanViewVideo` und damit auch
  den Stream-Token-Pfad ab)
- `internal/videos/crud.go` — `visibilityFilter` (Liste)
- `internal/videos/worker.go` — `pushRecipients` (Ready-Meldung)
- Keine Migration, kein Schema, keine neue Route, keine Frontend-Änderung. Wirksam ab
  Deploy; Bestandsvideos werden dadurch sichtbar, ohne dass etwas nachgezogen werden muss.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `GET /api/videos` | `TestListVideos_ErwKaderSiehtTeamvideos` | Ein Mitglied nur im erweiterten Kader erhält die Videos seiner Mannschaft in der Liste |
| `GET /api/videos` | `TestListVideos_ElternDesErwKadersSehenTeamvideos` | Elternteil eines nur erweiterten Mitglieds ebenso |
| `GET /api/videos` | `TestListVideos_FoerderkindWirdNichtWegenStatusGefiltert` | Ein erweitertes Mitglied mit `status='foerderkind'` sieht die Videos — der Statusfilter darf die Zielgruppe nicht ausschließen |
| `GET /api/videos` | `TestListVideos_AusgetretenesErwMitgliedSiehtNichts` | `status='ausgetreten'` schließt weiterhin aus |
| `GET /api/videos/{id}` | `TestGetVideo_ErwKaderDarfDetailLesen` | HTTP 200 statt 403 |
| `GET /api/videos/{id}` | `TestGetVideo_FremdesTeamBleibtVerboten` | Erweitertes Mitglied einer anderen Mannschaft: 403 |
| — (Worker) | `TestPushRecipients_ErwKaderUndEltern` | Ready-Meldung erreicht erweitertes Mitglied und dessen Elternteil |

**Garantierte Invariante:** Wer ein Video sehen darf, kann auch benachrichtigt werden — und
umgekehrt. Die drei Auflösungen dürfen nicht wieder auseinanderlaufen.
