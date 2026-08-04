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

  // Wieder austragen — der Test MUSS den Seed-Zustand zurücklassen, wie er ihn
  // vorgefunden hat. Die Seed-DB lebt über alle Versuche eines Laufs hinweg und
  // in CI gilt `retries: 2`: bliebe der Slot beansprucht, wäre jeder Retry
  // zwangsläufig rot („Eintragen" existiert dann nicht mehr) und würde die
  // eigentliche Ursache des ersten Fehlversuchs verdecken. Deckt zugleich den
  // Rückweg des Golden Path ab.
  await page.getByRole('button', { name: 'Austragen' }).click()
  await expect(page.getByRole('button', { name: 'Eintragen' }).first()).toBeVisible()

  // Abmelden → /login. Die Query ist bewusst offen: räumt der Logout-Handler
  // zuerst auf, landet man auf „/login"; rendert PrivateRoute zuerst (user ist
  // dann schon null), hängt es den Rücksprung als „/login?next=%2Fdienste" an
  // (App.tsx). Beide Reihenfolgen sind korrekt — auf „$" festgenagelt war der
  // Test von diesem Rennen abhängig und darum flaky.
  await page.getByRole('button', { name: 'Abmelden' }).click()
  await expect(page).toHaveURL(/\/login(\?|$)/)
})
