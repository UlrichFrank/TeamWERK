import { test, expect } from './fixtures'
import { loginAsAdmin } from './fixtures'

// chat-message-search: Suchtreffer öffnen springt zur richtigen Nachricht.
// Genuin E2E (docs/agent/07-testing.md): der Sprung ist Scroll-Verhalten in einem
// echten Layout — jsdom kennt weder scrollIntoView-Geometrie noch Viewport.
//
// Seed: „E2E Chat lang gelesen" hat 150 Nachrichten; Nachricht 30 trägt den
// eindeutigen Marker „Hallenadresse Musterweg 7 (E2E-Suchmarker)" (cmd/teamwerk/
// e2e_seed.go). Sie liegt VOR der Standard-100er-Seite — ein Klick auf den Treffer
// muss also den around-Cursor benutzen; über den Default-Pfad wäre sie gar nicht
// im DOM.
const BOX = '[data-windowed-scroll]'
const MARKER = 'E2E-Suchmarker'

test('Suchtreffer öffnet Konversation zentriert auf die Nachricht', async ({ page }) => {
  await loginAsAdmin(page)
  await page.goto('/chat')

  await page.getByRole('button', { name: 'Nachrichten durchsuchen' }).click()
  const dialog = page.getByRole('dialog', { name: 'Nachrichten durchsuchen' })
  await expect(dialog).toBeVisible()

  await dialog.getByLabel('Suchbegriff').fill(MARKER)

  // Genau ein Treffer, mit Konversationsname und hervorgehobenem Suchbegriff.
  const hit = dialog.getByRole('button').filter({ hasText: 'E2E Chat lang gelesen' })
  await expect(hit).toHaveCount(1)
  await expect(hit.locator('mark')).toContainText(MARKER)
  await hit.click()

  await expect(dialog).toBeHidden()

  // Zielnachricht steht im DOM UND im sichtbaren Bereich des Scroll-Containers.
  const box = page.locator(BOX)
  await expect(box).toBeVisible()
  const target = box.getByText(MARKER)
  await expect(target).toBeVisible()
  await expect
    .poll(
      async () =>
        target.evaluate((el: HTMLElement) => {
          const b = document.querySelector('[data-windowed-scroll]')!.getBoundingClientRect()
          const r = el.getBoundingClientRect()
          return r.top >= b.top && r.bottom <= b.bottom
        }),
      { timeout: 10_000 },
    )
    .toBe(true)

  // around-Fenster: Nachrichten VOR und NACH dem Treffer sind geladen (nicht nur
  // die neuesten 100), und die getroffene Zeile ist markiert.
  await expect(box.getByText('Nachricht 29', { exact: true })).toBeAttached()
  await expect(box.getByText('Nachricht 31', { exact: true })).toBeAttached()
  const row = box.locator('[data-message-id]').filter({ hasText: MARKER })
  await expect(row).toHaveClass(/bg-brand-yellow\/20/)
})

test('Suche ohne Treffer zeigt den Leerzustand', async ({ page }) => {
  // Der Seed enthält keine Mitteilungen; der Mitteilungs-Zweig ist über den
  // Go-Test TestSearch_TrefferInMitteilung abgedeckt. Hier nur: kein Treffer →
  // klarer Leerzustand statt leerer Fläche oder Ladeanzeige.
  await loginAsAdmin(page)
  await page.goto('/chat')
  await page.getByRole('button', { name: 'Nachrichten durchsuchen' }).click()
  const dialog = page.getByRole('dialog', { name: 'Nachrichten durchsuchen' })
  await dialog.getByLabel('Suchbegriff').fill('zz-kein-treffer-zz')
  await expect(dialog.getByText(/Keine Treffer/)).toBeVisible()
})
