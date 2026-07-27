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
