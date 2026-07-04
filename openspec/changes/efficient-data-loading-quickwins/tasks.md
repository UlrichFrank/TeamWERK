# Tasks — efficient-data-loading-quickwins

> Additiver Change, kein Vertragsbruch. Ein Commit pro Task. Backend zuerst.

## 1. Foundation: ETag-Helfer

- [x] 1.1 `internal/httpcache/etag.go` anlegen: `Serve(w, r, etag, cacheControl, bodyFn)` mit `If-None-Match`→`304`. Foundation-Package (importiert keine Domain).
- [x] 1.2 `internal/arch/arch_test.go` um Klassifizierung von `httpcache` (Foundation) ergänzen.
- [x] 1.3 `internal/httpcache/etag_test.go`: `TestServe_NoneMatchReturns304`, `TestServe_SetsCacheControl`.

  _Commit:_ `feat(httpcache): schwacher-ETag/304-Helfer als Foundation-Package`

## 2. Backend: Cache-Header auf Immutable-Routen

- [ ] 2.1 `GET /api/encryption-pubkey` (`internal/config/vault.go`): ETag + `Cache-Control: public, max-age=86400`.
- [ ] 2.2 `GET /api/push/vapid-public-key` (`internal/notifications/handler.go`): ETag + `Cache-Control: public, max-age=31536000, immutable`.
- [ ] 2.3 Tests: `TestEncryptionPubkey_ETag_304`, `TestVapidKey_CacheControlImmutable`.

  _Commit:_ `feat(config,notifications): Immutable-Cache-Header für pubkey/VAPID`

## 3. Backend: ETag/304 auf Referenzrouten

- [ ] 3.1 `GET /api/seasons` (`internal/config/handler.go`): schwacher ETag aus `COUNT`+`MAX(updated_at)`, `Cache-Control: private, no-cache`.
- [ ] 3.2 `GET /api/venues` (`internal/venues/handler.go`), `GET /api/age-class-rules` (`internal/config/handler.go`): analog.
- [ ] 3.3 Tests: `TestSeasons_ETagChangesOnMutation` (Happy + `304`-Revalidierung), Fehlerfall unverändert.

  _Commit:_ `feat(config,venues): ETag/304-Revalidierung für Referenzrouten`

## 4. Backend: duty-types-Liste trimmen

- [ ] 4.1 `internal/duties/handler.go:90` (`ListTypes`): `instruction_md` aus Listen-Serialisierung entfernen, `has_instruction bool` ergänzen. Detail-Pfad behält Volltext.
- [ ] 4.2 ETag/304 auf `GET /api/duty-types` (analog Task 3).
- [ ] 4.3 Tests: `TestDutyTypes_ListOmitsInstructionMd`, `TestDutyTypes_DetailKeepsInstructionMd`.

  _Commit:_ `feat(duties): duty-types-Liste liefert has_instruction statt Volltext`

## 5. Frontend: Client-TTL-Cache + Single-Flight

- [ ] 5.1 `web/src/lib/api.ts`: Referenz-Allowlist mit TTL, In-Memory-Map, Single-Flight für parallele Requests; SSE-Invalidierung (`seasons`/`settings`/`venues`/`duties`).
- [ ] 5.2 `AdminDutyTypesPage.tsx` bzw. Typ-Detail: Volltext aus Detail-Route laden.
- [ ] 5.3 `pnpm -C web build` + `lint` + Frontend-Test für Cache-Hit/Invalidierung.

  _Commit:_ `feat(pwa): Client-TTL-Cache + Single-Flight für Referenzdaten in api.ts`

## 6. Frontend: Coalescing + Service Worker

- [ ] 6.1 `web/src/hooks/useLiveUpdates.ts`: 300-ms-Coalescing-Fenster, deduplizierte Event-Typen, `__version:`-Pfad unberührt.
- [ ] 6.2 `web/src/sw.ts`: Referenzrouten auf `StaleWhileRevalidate`; `api-cache` `maxEntries`/`maxAgeSeconds`.
- [ ] 6.3 `pnpm -C web build` + `lint` + Test für Coalescing (mehrere Events → ein Callback).

  _Commit:_ `feat(pwa): useLiveUpdates-Coalescing + SW-StaleWhileRevalidate für Referenzdaten`

## 7. Abschluss

- [ ] 7.1 `/verify-change`.
- [ ] 7.2 `openspec validate efficient-data-loading-quickwins --strict`.
- [ ] 7.3 Proposal archivieren.

  _Commit:_ `chore(pwa): archiviere efficient-data-loading-quickwins`
