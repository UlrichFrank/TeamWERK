import { test, expect } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Golden Path: Login → Dashboard → Dienste → offenen Slot eintragen → Abmelden.
// Der Seed legt eine aktive Saison + Team + duty_type + einen freien Slot (Zukunftsdatum)
// an; Admin sieht auf dem Board alle Slots der aktiven Saison (kein Team-/Kader-Bezug nötig).
test('Golden Path: Login → Dienste → Slot eintragen → Abmelden', async ({ page }) => {
  await loginAsAdmin(page)

  // Nav-Link „Dienste" liegt im zugeklappten Akkordeon — direktes goto ist robuster.
  await page.goto('/dienste')
  await expect(page.getByRole('heading', { name: 'Dienste' })).toBeVisible()

  // Offener Slot → „Eintragen" (Admin hat keine Proxy-Kinder → Ein-Klick-Claim).
  const claim = page.getByRole('button', { name: 'Eintragen' }).first()
  await expect(claim).toBeVisible()
  await claim.click()

  // Erfolg: Zeile wechselt auf „Austragen" (claimed_by_me), kein „Eintragen" mehr.
  await expect(page.getByRole('button', { name: 'Austragen' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Eintragen' })).toHaveCount(0)

  // Abmelden → /login.
  await page.getByRole('button', { name: 'Abmelden' }).click()
  await expect(page).toHaveURL(/\/login$/)
})
