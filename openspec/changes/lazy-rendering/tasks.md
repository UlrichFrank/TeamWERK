# Tasks — lazy-rendering

> Frontend-only, kein API-/Schema-Change. Ein Commit pro Task. Tests via vitest.

## 1. Windowing-Grundlage

- [x] 1.1 Windowing-Ansatz (A hand-rolled vs. B schlankes Utility, siehe `design.md`) wählen; geteilte Komponente/Hook in `web/src/components/` bzw. `web/src/hooks/`.
- [x] 1.2 Test: `renders_only_visible_rows` (N≫Viewport → nur sichtbare + Puffer im DOM).
- [x] 1.3 `pnpm -C web build` + `lint`; Bundle-Delta via `make metrics` prüfen.

  _Commit:_ `feat(pwa): Windowing-Grundlage für lange Listen`

## 2. Lange Listen virtualisieren

- [x] 2.1 `MembersPage`, `DutyPage`/`DutySlotList`, `ChatPage`-Historie auf Windowing umstellen (bestehende Endpoints/„Mehr laden").
- [x] 2.2 Tests je Ansicht (nur Viewport gerendert; Scrollen tauscht Zeilen).

  _Commit:_ `feat(pwa): Members/Duty-Slots/Chat virtualisiert rendern`

## 3. VideosPage: Seiten erhalten

- [x] 3.1 `VideosPage.tsx`: `video-*`-Events per ID patchen/entfernen statt `fetchPage(0, true)`; „N neue"-Chip für `video-queued`; Scroll-Position erhalten.
- [x] 3.2 Test: `keeps_loaded_pages_on_sse_event`.

  _Commit:_ `feat(videos): geladene Seiten bei Live-Update erhalten statt Reset`

## 4. On-Demand-Rosters (MeinTeam)

- [ ] 4.1 `MeinTeamPage.tsx`: Roster erst bei Fokus/Aufklappen laden; geladene Rosters in der Session behalten.
- [ ] 4.2 Test: `roster_loads_only_when_expanded`.

  _Commit:_ `feat(teams): Team-Rosters on-demand statt eager laden`

## 5. Abschluss

- [ ] 5.1 `/verify-change`.
- [ ] 5.2 `openspec validate lazy-rendering --strict`.
- [ ] 5.3 Proposal archivieren.

  _Commit:_ `chore(pwa): archiviere lazy-rendering`
