// Nimmt die Screenshots für die Schulungsfolien auf — gesteuert über shots.txt.
// Aufruf: node shots.mjs <ids.json> <shots.txt> <ausgabe-ordner> <base-url>
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'

const require = createRequire(new URL('../../../web/package.json', import.meta.url))
const { chromium } = require('@playwright/test')

const [idsFile, listFile, outDir, BASE] = process.argv.slice(2)
const ids = JSON.parse(readFileSync(idsFile, 'utf8'))
const PASSWORD = 'Schulung2026!'

// Personas: Login und Gerät. Die Adressen setzt anon.py.
const PERSONAS = {
  vorstand: { email: 'vorstand@beispiel.de', mobile: false },
  trainer: { email: 'trainer@beispiel.de', mobile: false },
  'vorstand-mobil': { email: 'vorstand@beispiel.de', mobile: true },
  'trainer-mobil': { email: 'trainer@beispiel.de', mobile: true },
}

// Zeilen: persona | route | datei | klick (optional) | scrollen (optional)
const shots = readFileSync(listFile, 'utf8').split('\n')
  .map(l => l.trim()).filter(l => l && !l.startsWith('#'))
  .map(l => {
    const [persona, route, file, click, scroll] = l.split('|').map(s => s.trim())
    if (!PERSONAS[persona]) throw new Error(`Unbekannte Persona "${persona}" in: ${l}`)
    const resolved = route.replace(/\{(\w+)\}/g, (_, k) => {
      if (ids[k] == null) throw new Error(`Platzhalter {${k}} ohne Wert (ids.json)`)
      return ids[k]
    })
    return { persona, route: resolved, file, click, scroll }
  })

const browser = await chromium.launch()
for (const persona of [...new Set(shots.map(s => s.persona))]) {
  const { email, mobile } = PERSONAS[persona]
  // Ausgabebreite: Desktop 1600 px, Handy 600 px (Folien brauchen nicht mehr)
  const ctx = await browser.newContext(mobile
    ? { viewport: { width: 390, height: 844 }, deviceScaleFactor: 600 / 390, isMobile: true, hasTouch: true, locale: 'de-DE', timezoneId: 'Europe/Berlin' }
    : { viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1600 / 1440, locale: 'de-DE', timezoneId: 'Europe/Berlin' })
  const page = await ctx.newPage()
  await page.goto(BASE + '/login')
  await page.locator('input[autocomplete="username"]').fill(email)
  await page.locator('input[type="password"]').fill(PASSWORD)
  await page.locator('input[type="password"]').press('Enter')
  await page.waitForFunction(() => !location.pathname.startsWith('/login'), null, { timeout: 15000 })

  for (const s of shots.filter(x => x.persona === persona)) {
    await page.goto(BASE + s.route)
    await page.waitForLoadState('networkidle').catch(() => {})
    if (s.click) {
      await page.getByRole('button', { name: s.click, exact: true }).first().click()
      await page.waitForLoadState('networkidle').catch(() => {})
    }
    if (s.scroll) {
      await page.getByText(s.scroll, { exact: true }).first().evaluate(el => el.scrollIntoView({ block: 'start' }))
      await page.evaluate(() => window.scrollBy(0, -24))
    }
    await page.waitForTimeout(1200)
    await page.screenshot({ path: `${outDir}/${s.file}.jpg`, type: 'jpeg', quality: 82 })
    console.log(`  ${s.file}.jpg  ←  ${persona} ${s.route}`)
  }
  await ctx.close()
}
await browser.close()
