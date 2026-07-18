import { test, expect } from './fixtures'
import { loginAsAdmin } from './fixtures'

// Zero-Knowledge-Smoke: der Tresor wird über die UI eingerichtet — Keypair-Erzeugung
// (RSA-OAEP) + PBKDF2(600k) + AES-GCM laufen im ECHTEN Chromium-WebCrypto. Genau das
// deckt kein Vitest (Node-WebCrypto) und kein Go-Test ab: die ausgelieferte Browser-Krypto
// plus die TresorPage→API-Verdrahtung. Der Seed liefert die nötige clubs-Zeile.
test('Tresor über die UI einrichten → entsperrt (echtes Browser-WebCrypto)', async ({ page }) => {
  await loginAsAdmin(page)
  await page.goto('/tresor')

  // Einrichtungs-Zustand (configured === false): zwei Passphrase-Felder + „Tresor einrichten".
  const pass = 'E2ETresorPassphrase123'
  await page.getByPlaceholder('Neue Passphrase (min. 12 Zeichen)').fill(pass)
  await page.getByPlaceholder('Passphrase bestätigen').fill(pass)
  await page.getByRole('button', { name: 'Tresor einrichten' }).click()

  // Nach Setup wird direkt entsperrt → „Tresor entsperrt" (RSA-Keygen kann ein paar Sek dauern).
  await expect(page.getByText('Tresor entsperrt')).toBeVisible({ timeout: 20_000 })
})
