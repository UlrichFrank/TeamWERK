## 1. Fix und Verifikation im Frontend

- [x] 1.1 `web/src/components/AuthImage.tsx`: Platzhalter-`div` bekommt bei Server-Dims zusätzlich `width: ${naturalWidth}px` (neben `aspectRatio`); Kommentar am Style erklärt die shrink-to-fit-Ursache (leerer `div` trägt 0 zur max-content-Breite bei). Fallback ohne Dims unverändert.
- [x] 1.2 `web/src/components/__tests__/AuthImage.dimensions.test.tsx`: Assertion ergänzen — mit `naturalWidth`/`naturalHeight` trägt der Platzhalter (`[aria-busy="true"]`) einen `width`-Style mit der natürlichen Breite; ohne Dims weiterhin nur `min-height`, kein `width`.
- [x] 1.3 `web/e2e/chat-scroll.spec.ts`: neuer Test „Bild-Platzhalter mit Dims: Inhaltshöhe bleibt beim Decode stabil" — `page.route` hält `GET /api/media/*` an, „E2E Chat mit Bildern" öffnen, auf 4 `[aria-busy="true"]` warten, `scrollHeight` messen, Routen freigeben, `waitAllImagesLoaded(page, 4)`, erneut messen; `|Δ| ≤ 4`. Vor dem Fix ausführen und rot dokumentieren (erwartet ~1700 px), nach dem Fix grün.
- [x] 1.4 (nicht nötig — E2E zeigt nach dem Fix Δ ≤ 4 px, Tailwind-Preflight `img{display:block}` greift wie erwartet) Falls 1.3 nach dem Fix eine Restdifferenz > 4 px zeigt: `display: block` explizit am `<img>` in `AuthImage` setzen (Tailwind `block`), nicht die Toleranz anheben. Sonst Task als „nicht nötig" abhaken.
- [x] 1.5 `make test-e2e`: alle acht Tests in `chat-scroll.spec.ts` grün (die sieben bestehenden unverändert).

## 2. Dokumentation

- [x] 2.1 `docs/agent/06-gotchas.md`: Absatz „Bild-Platzhalter in Sprechblasen" — `aspect-ratio` auf einem leeren Element in einem shrink-to-fit-Container (Flex-Kind mit `items-start`/`items-end`) reserviert nichts, weil das Element 0 zur max-content-Breite beiträgt; Platzhalter brauchen eine explizite `width` (gedeckelt über `max-width: 100%`). Verweis darauf, dass der Fix aus `chat-image-dimensions` deshalb bis zu diesem Change wirkungslos war und dass die Anker-Tests nur Position, nicht Höhe prüfen.
- [x] 2.2 `openspec/specs/chat-open-at-unread/spec.md`: Purpose-Zeile („TBD – created by archiving …") beim Archivieren durch einen Satz ersetzen; die Delta-Spec dieses Changes mergen.

## 3. Abschluss

- [x] 3.1 `/verify-change` ausführen (Build/Test/Lint, brand-Tokens, `openspec validate`).
- [x] 3.2 Change archivieren (`/opsx:archive`) und Commit `fix(chat): Bild-Platzhalter reserviert Breite in shrink-to-fit-Sprechblase (iOS-Flackern)`.
