import { test, expect, type Page } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Der eigentliche Bug-Vector: Scroll-Position nach Bild-Decode. jsdom hat kein Layout/Decode,
// Chromium schon — deshalb sind diese Tests genuin E2E (nicht per Vitest abdeckbar).
const BOX = '[data-windowed-scroll]'

// KEINE Retries in dieser Datei: (1) es sind deterministische Layout-/Scroll-Tests —
// Retries würden echte Regressionen maskieren statt Infra-Flakes abzufangen (der
// Login-Cold-Start ist über den 15s-Timeout in loginAsAdmin abgedeckt). (2) Die
// unread/Chip-Tests hängen davon ab, dass die Konversation UNGELESEN ist; das erste
// Öffnen ruft MarkRead → unread=0 in der geteilten Seed-DB, ein Retry öffnet dieselbe
// (jetzt gelesene) Konversation → Divider/Chip rendert nie → Retry wäre garantiert rot.
test.describe.configure({ retries: 0 })

async function openChat(page: Page) {
  await page.goto('/chat')
}

// Wartet, bis die erwarteten Chat-Bild-<img> im DOM UND vollständig dekodiert sind.
// AuthImage lädt das Bild per XHR als Blob → der <img> erscheint erst nach dem Fetch.
async function waitAllImagesLoaded(page: Page, expected: number) {
  await expect(page.locator(`${BOX} img[alt="Bild"]`)).toHaveCount(expected)
  await page.waitForFunction((min) => {
    const imgs = Array.from(
      document.querySelectorAll('[data-windowed-scroll] img[alt="Bild"]'),
    ) as HTMLImageElement[]
    return imgs.length >= min && imgs.every((i) => i.complete && i.naturalHeight > 0)
  }, expected)
}

test('gelesene Bild-Konversation öffnet am Ende (nach Bild-Decode)', async ({ page }) => {
  await loginAsAdmin(page)
  await openChat(page)
  await page.getByText('E2E Chat mit Bildern').click()

  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  await waitAllImagesLoaded(page, 4)

  // Nach allen Bild-Loads am Ende (Sub-Pixel-Toleranz).
  await expect
    .poll(
      async () =>
        box.evaluate((el: HTMLElement) => Math.abs(el.scrollHeight - el.clientHeight - el.scrollTop)),
      { timeout: 10_000 },
    )
    .toBeLessThanOrEqual(4)
})

test('unread-Konversation landet am Divider „3 ungelesene Nachrichten"', async ({ page }) => {
  await loginAsAdmin(page)
  await openChat(page)
  await page.getByText('E2E Chat unread').click()

  const divider = page.getByText('3 ungelesene Nachrichten')
  await expect(divider).toBeVisible()

  // Divider sitzt oben im Viewport des Scroll-Containers (scrollIntoView block:"start").
  await expect
    .poll(
      async () =>
        divider.evaluate((el: HTMLElement) => {
          const box = document.querySelector('[data-windowed-scroll]') as HTMLElement
          const b = box.getBoundingClientRect()
          const d = el.getBoundingClientRect()
          return d.top >= b.top - 2 && d.top <= b.bottom
        }),
      { timeout: 10_000 },
    )
    .toBe(true)
})

test('Deep-Link ?openUser öffnet Direkt-Chat mit Verlauf (nicht am Anfang)', async ({ page }) => {
  await loginAsAdmin(page)
  await page.goto('/chat?openUser=3') // user2 = id 3 (deterministische Seed-Reihenfolge)

  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  await expect
    .poll(async () => box.evaluate((el: HTMLElement) => el.scrollTop), { timeout: 10_000 })
    .toBeGreaterThan(0)
})

// iOS Safari — das PWA-Haupteinsatzumfeld — kennt KEIN CSS scroll-anchoring
// (overflow-anchor). Genau dort tritt der Bug auf: wächst ein Bild ÜBER dem
// Sichtbereich (Decode nach dem Öffnungs-Scroll), kompensiert der Browser das NICHT
// und die Position driftet weg. Chromium HAT scroll-anchoring und würde den Bug
// maskieren (der einmalige scrollIntoView des Alt-Codes sähe „korrekt" aus). Wir
// schalten es für den Scroll-Container ab, damit der Test das reale Safari-Verhalten
// prüft — und damit tatsächlich die JS-Verankerung (nicht der Browser) getestet wird.
async function emulateNoScrollAnchoring(page: Page) {
  await page.addStyleTag({
    content: '[data-windowed-scroll]{overflow-anchor:none !important}',
  })
}

// Wartet, bis KEINE AuthImage-Platzhalter (aria-busy) mehr im Container sind UND alle
// vorhandenen <img> dekodiert sind — count-unabhängig, deshalb robust gegen die genaue
// Bildzahl auf der geladenen 100er-Seite (die von der Seed-Verteilung abhängt).
async function waitImagesSettled(page: Page) {
  await page.waitForFunction(
    () => {
      const box = document.querySelector('[data-windowed-scroll]')
      if (!box) return false
      if (box.querySelector('[aria-busy="true"]')) return false
      const imgs = Array.from(
        box.querySelectorAll('img[alt="Bild"]'),
      ) as HTMLImageElement[]
      return imgs.every((i) => i.complete && i.naturalHeight > 0)
    },
    undefined,
    { timeout: 20_000 },
  )
}

// KERN-REGRESSION: In einem langen, bildlastigen Thread mit tief liegender Ungelesen-
// Grenze decoden Bilder ÜBER dem Divider erst nach dem Öffnungs-Scroll und schieben den
// Divider aus dem Viewport. Vor dem Fix (einmaliger scrollIntoView) driftet der Divider
// nach unten weg; mit dem intent-basierten Anker bleibt er nach Bild-Decode oben.
test('lange unread-Konversation: Divider bleibt oben NACH Bild-Decode', async ({
  page,
}) => {
  await loginAsAdmin(page)
  await openChat(page)
  await emulateNoScrollAnchoring(page)
  await page.getByText('E2E Chat lang unread').click()

  const box = page.locator(BOX)
  await expect(box).toBeVisible()

  const divider = page.getByText('40 ungelesene Nachrichten')
  await expect(divider).toBeVisible()

  // Erst nach vollständigem Bild-Decode prüfen — genau das ist der Bug-Vector.
  await expect(page.locator(`${BOX} img[alt="Bild"]`).first()).toBeVisible()
  await waitImagesSettled(page)

  // Divider sitzt weiterhin am oberen Rand des Scroll-Containers (block:"start").
  // Vor dem Fix driftet er durch die decodenden Bilder darüber deutlich nach unten.
  await expect
    .poll(
      async () =>
        divider.evaluate((el: HTMLElement) => {
          const b = (
            document.querySelector('[data-windowed-scroll]') as HTMLElement
          ).getBoundingClientRect()
          const d = el.getBoundingClientRect()
          return d.top - b.top // ~0 = ganz oben; groß = weggedriftet
        }),
      { timeout: 10_000 },
    )
    .toBeLessThanOrEqual(80)
})

// Positiv-Test: langer, bildlastiger, KOMPLETT gelesener Thread landet zuverlässig am
// Ende — auch wenn Bilder erst nach dem initialen End-Scroll ihre Höhe annehmen.
test('lange gelesene Konversation öffnet am Ende (nach Bild-Decode)', async ({
  page,
}) => {
  await loginAsAdmin(page)
  await openChat(page)
  await emulateNoScrollAnchoring(page)
  await page.getByText('E2E Chat lang gelesen').click()

  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  await expect(page.locator(`${BOX} img[alt="Bild"]`).first()).toBeVisible()
  await waitImagesSettled(page)

  await expect
    .poll(
      async () =>
        box.evaluate((el: HTMLElement) =>
          Math.abs(el.scrollHeight - el.clientHeight - el.scrollTop),
        ),
      { timeout: 10_000 },
    )
    .toBeLessThanOrEqual(4)
})

// Chip-Fall: erste Ungelesene liegt VOR der geladenen 100er-Seite → Chip statt Divider,
// Container landet oben und bleibt nach Bild-Decode oben; „Ältere laden" erhält die
// Position (kein Sprung ans Ende), auch wenn voran-gestellte Bilder decoden.
test('viele-ungelesen-Konversation: Chip oben, „Ältere laden" erhält Position', async ({
  page,
}) => {
  await loginAsAdmin(page)
  await openChat(page)
  await emulateNoScrollAnchoring(page)
  await page.getByText('E2E Chat viele ungelesen').click()

  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  await expect(
    page.getByText(/80 weitere\s+ungelesene Nachrichten älter/),
  ).toBeVisible()

  await expect(page.locator(`${BOX} img[alt="Bild"]`).first()).toBeVisible()
  await waitImagesSettled(page)

  // Container steht oben (divider-chip-Anker), auch nach Bild-Decode.
  await expect
    .poll(async () => box.evaluate((el: HTMLElement) => el.scrollTop), {
      timeout: 10_000,
    })
    .toBeLessThanOrEqual(4)

  // „Ältere laden": Position der bisher sichtbaren Nachrichten muss erhalten bleiben.
  // Invariante: Δ(scrollTop) ≈ Δ(scrollHeight) (der Alt-Content bewegt sich nicht, nur
  // oben kommt Höhe dazu). Vor dem loadOlder-Fix springt die Ansicht durch später
  // decodende, voran-gestellte Bilder weg.
  const before = await box.evaluate((el: HTMLElement) => ({
    top: el.scrollTop,
    height: el.scrollHeight,
  }))
  await page.getByRole('button', { name: 'Ältere Nachrichten laden' }).click()
  // Warten, bis die voran-gestellte Seite im DOM ist (mehr Bubbles) + Bilder decodiert.
  await page.waitForFunction(
    (prevH) => {
      const el = document.querySelector('[data-windowed-scroll]') as HTMLElement
      return !!el && el.scrollHeight > prevH
    },
    before.height,
    { timeout: 15_000 },
  )
  await waitImagesSettled(page)

  await expect
    .poll(
      async () =>
        box.evaluate(
          (el: HTMLElement, prev) =>
            Math.abs(
              el.scrollTop - prev.top - (el.scrollHeight - prev.height),
            ),
          before,
        ),
      { timeout: 10_000 },
    )
    .toBeLessThanOrEqual(8)
})

// KERN-REGRESSION (chat-conversation-switch-flicker): openConversation setzte
// activeConv synchron, ließ `messages` aber bis zum Auflösen des Fetches unangetastet —
// im Verzögerungsfenster stand der neue Header über der Nachrichtenliste der VORHERIGEN
// Konversation. Ein Rennen zwischen Netzwerk und Paint (in Chromium meist zu knapp, um es
// zu treffen — siehe design.md „Ergebnis 1"). Künstliches Verzögern der Nachrichten-Antwort
// macht das Rennen deterministisch (design.md, Entscheidung 4), ganz ohne WebKit/Video.
test('Konversationswechsel zeigt keinen Fremdinhalt', async ({ page }) => {
  await loginAsAdmin(page)
  await openChat(page)

  // Nur der GET auf die Nachrichten-Route wird verzögert, und erst ab dem ZWEITEN Aufruf
  // (= Wechsel zu B) — das Öffnen von A bleibt ungebremst, damit das Verzögerungsfenster
  // eindeutig dem Wechsel zuzuordnen ist. Das Glob endet ohne Wildcard auf „/messages" und
  // trifft daher NICHT die `?before=`-Pagination von loadOlderMessages.
  let messagesFetchCount = 0
  await page.route('**/api/chat/conversations/*/messages', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue()
      return
    }
    messagesFetchCount += 1
    if (messagesFetchCount >= 2) {
      await new Promise((resolve) => setTimeout(resolve, 800))
    }
    await route.continue()
  })

  const box = page.locator(BOX)

  // A = "E2E Chat" (8 Textnachrichten, B = "E2E Chat lang gelesen" (150 Textnachrichten).
  // Beide bewusst gewählt, weil ALLE ihre Nachrichten vom Admin selbst gesendet sind →
  // unreadCount ist für beide immer 0, unabhängig davon, ob/wie oft dieser Test gegen die
  // geteilte Seed-DB läuft. Anders als "E2E Chat unread"/"E2E Chat lang unread"/"E2E Chat
  // viele ungelesen" (von den Bestandstests für Divider/Chip benötigt) verbraucht das
  // Öffnen hier keinen Unread-Zustand, auf den andere Tests angewiesen sind.
  await page.getByText('E2E Chat', { exact: true }).click()
  await expect(box.getByText('E2E Nachricht 1', { exact: true })).toBeVisible()

  await page.getByText('E2E Chat lang gelesen', { exact: true }).click()

  // Innerhalb des 800ms-Verzögerungsfensters: Header zeigt bereits B — aber KEIN
  // Nachrichtentext aus A darf im Nachrichten-Container stehen. Der kurze Timeout (deutlich
  // unter 800 ms) ist hier Absicht: ohne Fix bleibt A's Text die vollen 800 ms sichtbar
  // (die Liste wird nicht vor dem Fetch geleert) — ein langer/Default-Timeout würde erst
  // NACH Auflösen des Fetches prüfen (dann sind ohnehin nur noch B's Nachrichten im DOM)
  // und den Bug verdecken. `span.font-semibold.text-brand-text.truncate` ist die einzige
  // Stelle, die `convName(activeConv)` im Header rendert (eindeutig, im Unterschied zu
  // `getByText`, das auch den gleichnamigen Sidebar-Eintrag träfe).
  await expect(
    page.locator('span.font-semibold.text-brand-text.truncate'),
  ).toHaveText('E2E Chat lang gelesen', { timeout: 300 })
  await expect(
    box.getByText('E2E Nachricht 1', { exact: true }),
  ).not.toBeVisible({ timeout: 300 })

  // Nach Auflösen der Verzögerung: B's echte Nachrichten stehen im DOM.
  await expect(box.getByText('Nachricht 150', { exact: true })).toBeVisible()
})

// HÖHENSTABILITÄT, nicht Position: Die Anker-Tests oben prüfen, WO der Container nach dem
// Bild-Decode steht — nicht, ob sich die Inhaltshöhe dabei überhaupt ändern musste. Genau
// das war der blinde Fleck: der AuthImage-Platzhalter (leerer div mit aspect-ratio) trägt
// in der shrink-to-fit-Sprechblase 0 zur Breite bei und kollabiert auf 24×16 px, das
// fertige <img> wird 344×262 px. Jeder eintreffende Blob wächst die Blase, der Anker
// korrigiert nachträglich — unter iOS Safari (kein scroll-anchoring) pro Bild ein sichtbarer
// Sprung. Mit korrekt reserviertem Platzhalter ist Δ scrollHeight ≈ 0.
//
// Deterministisch gemacht durch Anhalten der Medien-Antworten (page.route): lokal ist der
// Blob-Fetch im Millisekunden-Bereich, ohne Anhalten gibt es keinen stabilen Moment, in dem
// alle Platzhalter sichtbar sind. „E2E Chat mit Bildern" hat 4 Bilder, ALLE mit Server-Dims
// (seedChatMedia → seedImage(…, true)); die langen Threads mischen absichtlich Bilder ohne
// Dims (6-rem-Fallback) und taugen deshalb nicht für Δ≈0.
//
// ZWEITER BLINDER FLECK (Vorher/Nachher reicht nicht): Der <img> bemisst sich ohne explizite
// Breite an seiner intrinsischen Größe — und die ist bei einer frisch gesetzten Blob-URL noch
// NICHT bekannt: im Einfüge-Moment ist er `complete=false`, naturalWidth=0 und misst 0×0
// (Chromium UND WebKit), der Scroll-Inhalt schrumpft um die volle Bildhöhe und wächst beim
// Eintreffen der Daten OHNE DOM-Mutation wieder. Vorher/Nachher-Messung sieht davon nichts,
// weil beide Zustände gleich hoch sind; unter iOS Safari (kein scroll-anchoring, `load` erst
// nach dem nächsten Paint) ist das Wachsen oberhalb des Sichtbereichs ein sichtbarer Sprung
// pro Bild. Deshalb misst der Test zusätzlich SYNCHRON im Einfüge-Moment (MutationObserver,
// registriert vor dem Freigeben der Routen): jeder eingefügte <img> muss bereits seine
// Zielhöhe haben, und scrollHeight darf in diesem Moment nicht unter die Ausgangshöhe fallen.
test('Bild-Platzhalter mit Dims: Inhaltshöhe bleibt beim Decode stabil', async ({ page }) => {
  await loginAsAdmin(page)

  // Medien-Antworten anhalten, bis die Ausgangshöhe gemessen ist.
  const pending: Array<() => void> = []
  await page.route('**/api/media/*', async (route) => {
    await new Promise<void>((resolve) => pending.push(resolve))
    await route.continue()
  })

  await openChat(page)
  await page.getByText('E2E Chat mit Bildern').click()

  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  await expect(page.locator(`${BOX} [aria-busy="true"]`)).toHaveCount(4)

  const before = await box.evaluate((el: HTMLElement) => ({
    height: el.scrollHeight,
    // Breite der Platzhalter: ein kollabierter Platzhalter ist schmaler als die Blase.
    minPlaceholderWidth: Math.min(
      ...Array.from(el.querySelectorAll('[aria-busy="true"]')).map(
        (p) => p.getBoundingClientRect().width,
      ),
    ),
  }))

  // Einfüge-Moment beobachten: Höhe jedes neuen <img> und scrollHeight, gemessen im
  // MutationObserver-Callback (Microtask nach dem Commit, VOR dem Eintreffen der Blob-Daten).
  await box.evaluate((el: HTMLElement) => {
    const w = window as unknown as {
      __insertedImgs: Array<{ h: number; complete: boolean; scrollHeight: number }>
    }
    w.__insertedImgs = []
    const mo = new MutationObserver((records) => {
      for (const r of records)
        for (const n of Array.from(r.addedNodes)) {
          if (n instanceof HTMLImageElement) {
            w.__insertedImgs.push({
              h: n.getBoundingClientRect().height,
              complete: n.complete,
              scrollHeight: el.scrollHeight,
            })
          }
        }
    })
    mo.observe(el, { childList: true, subtree: true })
  })

  // Routen freigeben → Blobs kommen, <img> ersetzen die Platzhalter.
  pending.splice(0).forEach((release) => release())
  await waitAllImagesLoaded(page, 4)

  const after = await box.evaluate((el: HTMLElement) => ({
    height: el.scrollHeight,
    minImgWidth: Math.min(
      ...Array.from(el.querySelectorAll('img[alt="Bild"]')).map(
        (i) => i.getBoundingClientRect().width,
      ),
    ),
    minImgHeight: Math.min(
      ...Array.from(el.querySelectorAll('img[alt="Bild"]')).map(
        (i) => i.getBoundingClientRect().height,
      ),
    ),
    inserted: (
      window as unknown as {
        __insertedImgs: Array<{ h: number; complete: boolean; scrollHeight: number }>
      }
    ).__insertedImgs,
  }))

  // Ohne Fix: ~4 × 430 px Wachstum (Platzhalter 16 px → Bild ~444 px).
  expect(Math.abs(after.height - before.height)).toBeLessThanOrEqual(4)
  // Kein Platzhalter war schmaler als das Bild, das ihn ersetzt hat.
  expect(before.minPlaceholderWidth).toBeGreaterThanOrEqual(after.minImgWidth - 1)

  // Einfüge-Moment: alle 4 <img> beobachtet, jeder hatte sofort seine Zielhöhe (nicht 0),
  // und der Scroll-Inhalt ist zwischendurch nie unter die Ausgangshöhe gefallen.
  // Ohne explizite <img>-Breite: h=0 für jeden Eintrag, scrollHeight um 4 × Bildhöhe kleiner.
  expect(after.inserted).toHaveLength(4)
  for (const ins of after.inserted) {
    expect(ins.h).toBeGreaterThanOrEqual(after.minImgHeight - 1)
    expect(before.height - ins.scrollHeight).toBeLessThanOrEqual(4)
  }
})
