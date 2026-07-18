import { test, expect } from '@playwright/test'
import { loginAsAdmin, ADMIN } from './fixtures'

test('Login mit gültigen Credentials → App lädt (Dashboard)', async ({ page }) => {
  await loginAsAdmin(page)
  // Login-Formular ist weg → wir sind in der App.
  await expect(page.getByRole('button', { name: 'Anmelden' })).toHaveCount(0)
})

test('Login mit falschem Passwort → sichtbare Fehlermeldung, kein Redirect', async ({ page }) => {
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(ADMIN.email)
  await page.locator('input[autocomplete="current-password"]').fill('falsch-falsch')
  await page.getByRole('button', { name: 'Anmelden' }).click()
  await expect(page.getByText(/ungültig/i)).toBeVisible()
  await expect(page).toHaveURL(/\/login$/)
})
