import { test, expect } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Scroll-Position beim Zurück — genuin E2E: der Scroll-Container ist das <main> in
// AppShell (overflow-auto), jsdom hat weder Layout noch Scroll-Klemmung.
//
// Was dieser Test NICHT ist: ein Nachweis, dass useScrollRestoration gebraucht wird.
// Chromium stellt verschachtelte Scroller bei History-Navigation selbst wieder her —
// hier nachgemessen auch mit mehreren Sekunden API-Latenz und über einen Reload hinweg.
// Der eigentliche Fehlerfall ist die iOS-Homescreen-PWA, die dieser Stack nicht fahren
// kann.
//
// Was er ist: ein Wächter gegen Sabotage. Ein selbstgebauter Restore, der zu früh
// aufgibt, setzt scrollTop auf 0/„zu kurz" und storniert damit den nativen Restore —
// die Position ist dann SCHLECHTER als ohne Hook. Genau diese Variante ist während der
// Entwicklung entstanden, und genau sie fällt hier durch: die künstliche API-Latenz
// zwingt jeden Restore-Mechanismus zum Nachfassen.
test.describe.configure({ retries: 0 })

// Deutlich über dem, was die Liste zum Laden braucht — der Container ist beim Zurück
// also garantiert zu kurz für die Zielposition.
const API_DELAY_MS = 4500

const MAIN = 'main'

async function scrollMetrics(page: import('@playwright/test').Page) {
  return page.evaluate(() => {
    const el = document.querySelector('main')!
    return { top: el.scrollTop, max: el.scrollHeight - el.clientHeight }
  })
}

test('Zurück stellt die Scroll-Position wieder her (ohne Karten-Klick)', async ({ page }) => {
  await loginAsAdmin(page)
  // Sehr niedriges Fenster bei Desktop-Breite (Sidebar bleibt sichtbar): die Seed-DB
  // enthält genau einen Dienst-Slot, <main> wird sonst gar nicht scrollbar.
  await page.setViewportSize({ width: 1000, height: 150 })

  await page.goto('/dienste')
  await expect(page.getByRole('heading', { name: 'Dienste' })).toBeVisible()

  // Der Inhalt kommt per API nach — erst danach steht die Scrollhöhe fest.
  await expect
    .poll(async () => (await scrollMetrics(page)).max, { timeout: 5000 })
    .toBeGreaterThan(40)
  const { max } = await scrollMetrics(page)
  const target = Math.min(60, max)

  await page.locator(MAIN).evaluate((el, top) => { el.scrollTop = top }, target)
  await expect.poll(async () => (await scrollMetrics(page)).top).toBe(target)

  // In-App-Navigation (Push) auf einen anderen Pfad → neue Seite beginnt oben.
  await page.getByRole('link', { name: 'Dokumente' }).click()
  await expect(page).toHaveURL(/\/dokumente$/)
  await expect.poll(async () => (await scrollMetrics(page)).top).toBe(0)

  // Ab jetzt lädt die Dienst-Liste langsam — beim Zurück ist <main> zunächst leer.
  await page.route('**/api/duty-board*', async route => {
    await new Promise(r => setTimeout(r, API_DELAY_MS))
    await route.continue()
  })

  // Browser-Zurück → dieselbe Position wie vorher, obwohl die Liste erst nachlädt.
  await page.goBack()
  await expect(page.getByRole('heading', { name: 'Dienste' })).toBeVisible()
  await expect.poll(async () => (await scrollMetrics(page)).top, { timeout: 15000 }).toBe(target)
})
