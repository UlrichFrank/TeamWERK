import { test, expect, type Page } from '@playwright/test'

export const ADMIN = { email: 'e2e@test.local', password: 'E2ETestPassword!' }

// loginAsAdmin füllt das Login-Formular und wartet, bis die App (Dashboard unter "/") geladen ist.
export async function loginAsAdmin(page: Page) {
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(ADMIN.email)
  await page.locator('input[autocomplete="current-password"]').fill(ADMIN.password)
  await page.getByRole('button', { name: 'Anmelden' }).click()
  // Nach erfolgreichem Login navigiert LoginPage auf "/" (Dashboard). Großzügiger
  // Timeout: der erste Login nach Server-Start ist durch bcrypt + JWT-Aufwärmen
  // spürbar langsamer (sonst gelegentlich flaky beim Cold-Start).
  await expect(page).toHaveURL(/\/$/, { timeout: 15000 })
}

export { test, expect }
