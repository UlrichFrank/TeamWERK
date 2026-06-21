# Tasks — profile-cross-team-visibility

> Ein Commit pro Task. Konventionelle Commits, Scope = führendes Domänen-Package.

## 1. Migration

- [x] 1.1 `internal/db/migrations/003_member_cross_team_visible.up.sql` + `.down.sql`: `ALTER TABLE members ADD COLUMN cross_team_visible INTEGER NOT NULL DEFAULT 0;`

  _Commit:_ `feat(db): cross_team_visible auf members für Opt-In-Cross-Team-Sichtbarkeit`

## 2. Backend: Filter in GetParticipants

- [x] 2.1 `internal/games/handler.go` (`GetParticipants`): Caller-Funktion und Caller-Member-IDs ermitteln, Set "meine Teams im Event" bilden (eigene + Kinder via `family_links`); Query so erweitern, dass fremde Team-Member nur bei `cross_team_visible=1` zurückgegeben werden. Funktionsträger (admin/trainer/sL/vorstand) bypassen den Filter. Single-Team-Events bleiben ungefiltert.
- [x] 2.2 `go vet` + `gofmt`.

  _Commit:_ `feat(games): /participants filtert fremde Teams bei Multi-Team-Events`

## 3. Backend: Member-PUT um cross_team_visible erweitern

- [x] 3.1 `internal/members/handler.go` (`UpdateMember`): `cross_team_visible` ins Update-DTO + SQL aufnehmen, ohne Draft-Workflow. Ownership-Check: eigenes Member ODER Kind via `family_links` ODER vorstand/admin. — *Dediziertes Endpoint `PUT /api/members/{id}/cross-team-visible` in Authenticated-Tier; bestehendes `PUT /api/members/{id}` (vorstand-only) zusätzlich erweitert für Admin-Form.*
- [x] 3.2 `go vet` + `gofmt`.

  _Commit:_ `feat(members): cross_team_visible per PUT /members/{id} direkt setzbar`

## 4. Backend-Tests

- [x] 4.1 `TestGetParticipants_MultiTeam_SpielerSiehtNurEigenesTeam`.
- [x] 4.2 `TestGetParticipants_MultiTeam_OptInMachtFremdSichtbar`.
- [x] 4.3 `TestGetParticipants_MultiTeam_ElternSehenTeamsDerKinder`.
- [x] 4.4 `TestGetParticipants_MultiTeam_KindIn2TeamsSiehtBeide`.
- [x] 4.5 `TestGetParticipants_MultiTeam_TrainerSiehtAlles`.
- [x] 4.6 `TestGetParticipants_MultiTeam_VorstandSiehtAlles`.
- [x] 4.7 `TestGetParticipants_SingleTeam_KeinFilter`.
- [x] 4.8 `TestUpdateMember_CrossTeamVisible_EigenesMember` (direct save).
- [x] 4.9 `TestUpdateMember_CrossTeamVisible_EigenesKindAlsElternteil`.
- [x] 4.10 `TestUpdateMember_CrossTeamVisible_Fremd_403`.

  _Commit:_ `test(games,members): cross-team-Sichtbarkeit – Filter und Setter`

## 5. Frontend: Datenschutz-Tab im Profil

- [x] 5.1 `web/src/components/profile/ProfileDatenschutzTab.tsx` (NEU): Toggle „Sichtbarkeit für Mitglieder" (verbindlicher Toggle-Stil aus `whatsapp-sichtbarkeit`), beschreibender Text. Read-only-Block für DSGVO (Verarbeitung/Weitergabe + Datum), gesperrtes visuelles Control. Direct-Save via dediziertes `PUT /api/members/{id}/cross-team-visible`.
- [x] 5.2 `web/src/pages/ProfilePage.tsx`: Tab-Liste um `'datenschutz'` ergänzen (Reihenfolge: account, profile, member, banking, kalender, **datenschutz**, misc). Tab nur sichtbar, wenn `ownMember` existiert.

  _Commit:_ `feat(profile): Datenschutz-Tab mit Sichtbarkeitstoggle und DSGVO-Anzeige`

## 6. Frontend: Admin Member-Datenschutz-Tab um Toggle ergänzen

- [x] 6.1 `web/src/components/admin/MemberDatenschutzTab.tsx`: Toggle „Sichtbarkeit für Mitglieder" oberhalb der DSGVO-Sektion, schreibt `cross_team_visible` direkt mit beim Save.

  _Commit:_ `feat(members): Datenschutz-Tab im Admin um Sichtbarkeitstoggle ergänzen`

## 7. Frontend: TermineDetailPage — Counter und Hinweis

- [x] 7.1 `web/src/pages/TermineDetailPage.tsx`: Counter-Badges aggregieren weiterhin aus `participants` (Backend-gefiltert), nichts zu ändern — sie spiegeln automatisch nur sichtbare Zeilen.
- [x] 7.2 Pro Sektion (`TableSection`) ein Footer „Weitere Mitglieder nicht sichtbar" rendern. Backend liefert `hidden_team_ids` in der Response; Frontend setzt `hasHidden` pro Sektion.
- [x] 7.3 Leere Sektionen (`rows.length === 0`) werden weiter über `.filter(s => s.rows.length > 0)` weggelassen — auch wenn `hasHidden` wäre.
- [x] 7.4 `pnpm -C web build`, `lint`, `test` — alle grün (348 Frontend-Tests).

  _Commit:_ `feat(termine): Counter und Hinweis bei gefilterten Multi-Team-Teilnehmern`

## 8. Frontend-Tests

- [x] 8.1 `ProfileDatenschutzTab.test.tsx`: Toggle laden, umschalten, PUT-Aufruf prüfen (+DSGVO read-only).
- [x] 8.2 `TermineDetailPage.crossteam.test.tsx` (NEU): Multi-Team-Szenarien — Footer pro Team mit `hidden_team_ids`, leere Sektionen nicht gerendert, keine Hinweise wenn nichts versteckt.

  _Commit:_ `test(profile,termine): Datenschutz-Tab und gefilterte Teilnahmeliste`

## 9. Verifikation & Archive

- [x] 9.1 Build/Test/Lint/Invariants — alle grün:
  - Backend: `go vet`, `go test -race ./...` (654 Tests), `golangci-lint` (keine neuen Findings auf geänderten Dateien)
  - Frontend: `pnpm build`, `pnpm test` (355 Tests), `pnpm lint` (clean)
  - SSE-Invariante: `Broadcast("members")` ↔ `useLiveUpdates('members')`
  - `openspec validate` ok
- [ ] 9.2 OpenSpec-Proposal archivieren (offen — vor Archivierung Merge/PR abwarten).

  _Commit:_ `chore(openspec): profile-cross-team-visibility archivieren`
