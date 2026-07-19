import { randomUUID } from 'node:crypto'
import { test, expect } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Deckt den kompletten Sende-Zyklus ab (Eingabe → API → Rendering in der eigenen
// Bubble). Der UUID-Marker macht den Test unabhängig vom Bestand und von
// Wiederholungen. Genuin E2E: prüft das reale Zusammenspiel Frontend↔Backend,
// nicht nur die Render-Logik.
test('Nachricht senden erscheint in eigener Bubble', async ({ page }) => {
  await loginAsAdmin(page)
  await page.goto('/chat')
  // Exakt die Text-Gruppe öffnen (nicht „E2E Chat mit Bildern"/„… unread").
  await page.getByText('E2E Chat', { exact: true }).click()

  const marker = `E2E-Test-${randomUUID()}`
  const input = page.getByPlaceholder('Nachricht schreiben…')
  await input.fill(marker)
  await input.press('Enter')

  await expect(page.getByText(marker)).toBeVisible({ timeout: 5000 })
})
