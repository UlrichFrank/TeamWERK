import { test, expect, type Page } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Der eigentliche Bug-Vector: Scroll-Position nach Bild-Decode. jsdom hat kein Layout/Decode,
// Chromium schon — deshalb sind diese Tests genuin E2E (nicht per Vitest abdeckbar).
const BOX = '[data-windowed-scroll]'

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
