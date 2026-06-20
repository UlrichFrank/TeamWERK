## 1. Shared-Komponente anlegen

- [x] 1.1 Verzeichnis `web/src/components/forms/` anlegen (sofern nicht vorhanden)
- [x] 1.2 Komponente `web/src/components/forms/PasswordInput.tsx` schreiben mit Props `{ value, onChange, autoComplete, required?, id?, name?, placeholder?, autoFocus?, disabled?, minLength? }` (`autoComplete` ist Pflicht)
- [x] 1.3 Interner State: `revealed: boolean`, `userTyped: boolean`, `tainted: boolean`, `lastUserActionAt: useRef<number>`
- [x] 1.4 Event-Handler implementieren:
  - `onKeyDown`, `onPaste`, `onCut`: setzen `lastUserActionAt.current = performance.now()`
  - `onChange`: wenn `value === ""` → reset (`userTyped=false`, `tainted=false`); sonst wenn `now - lastUserActionAt < 100ms` → `userTyped=true`; sonst → `tainted=true`
  - `onBlur`: `setRevealed(false)`
  - Toggle-Button-`onClick`: `setRevealed(!revealed)`
- [x] 1.5 `allowReveal = userTyped && !tainted && value.length > 0` — Toggle-Button nur rendern wenn `allowReveal === true`
- [x] 1.6 Input-Klassen-String aus `component-standards` (Standard) + `pr-10` Padding-right für Button-Platz
- [x] 1.7 Toggle-Button: `absolute right-0 top-0 h-full px-3 text-brand-text-muted hover:text-brand-text`, `type="button"`, `aria-label={revealed ? 'Passwort verbergen' : 'Passwort anzeigen'}`, `aria-pressed={revealed}`, Icon Lucide `Eye` (revealed=false) / `EyeOff` (revealed=true), `w-5 h-5`
- [x] 1.8 Wrapper-Container `<div class="relative">` umschließt Input + Button

## 2. LoginPage umstellen

- [x] 2.1 In `web/src/pages/LoginPage.tsx` Import `PasswordInput` hinzufügen
- [x] 2.2 Passwort-`<input>` (Zeile 63–69) durch `<PasswordInput value={password} onChange={setPassword} autoComplete="current-password" required />` ersetzen
- [ ] 2.3 Lokal testen: leeres Feld → kein Auge; ein Zeichen tippen → Auge sichtbar; Klick → Klartext; Tab raus → wieder maskiert *(manuelle Verifikation — siehe Sektion 6)*

## 3. RegisterPage umstellen

- [x] 3.1 In `web/src/pages/RegisterPage.tsx` das Passwort-Feld durch `<PasswordInput>` ersetzen, `autoComplete="new-password"`, `minLength={8}` *(Hinweis: nur 1 Feld — Planungs-Annahme „2 Felder" war falsch, RegisterPage hat keine Bestätigungs-Eingabe)*
- [ ] 3.2 Smoke-Test wie 2.3 *(manuelle Verifikation)*

## 4. ResetPasswordPage umstellen

- [x] 4.1 In `web/src/pages/ResetPasswordPage.tsx` das Passwort-Feld durch `<PasswordInput>` ersetzen, `autoComplete="new-password"`, `minLength={8}` *(Hinweis: nur 1 Feld — Planungs-Annahme „2 Felder" war falsch, ResetPasswordPage hat keine Bestätigungs-Eingabe)*
- [ ] 4.2 Smoke-Test wie 2.3 *(manuelle Verifikation)*

## 5. PasswordChangeModal umstellen

- [x] 5.1 In `web/src/components/profile/PasswordChangeModal.tsx` drei Passwort-Felder durch `<PasswordInput>` ersetzen
  - Feld „aktuelles Passwort": `autoComplete="current-password"`
  - Feld „neues Passwort": `autoComplete="new-password"`
  - Feld „Bestätigung": `autoComplete="new-password"`
- [ ] 5.2 Smoke-Test wie 2.3 *(manuelle Verifikation)*

## 6. Manuelle Verifikation (Verhaltens-Szenarien aus Spec)

- [ ] 6.1 **Pristine**: LoginPage frisch öffnen → Passwort-Feld leer → kein Auge sichtbar
- [ ] 6.2 **User-typed**: ein Zeichen tippen → Auge sichtbar → Klick: Klartext sichtbar → erneuter Klick: maskiert
- [ ] 6.3 **Blur**: Passwort tippen + Reveal aktiv → in E-Mail-Feld klicken → Passwort wieder maskiert beim Zurückklicken (Auge bleibt aber sichtbar)
- [ ] 6.4 **Browser-Autofill (Chrome)**: Login einmal mit gespeichertem Passwort durchlaufen, dann ausloggen, Login-Seite neu öffnen → Chrome füllt Passwort automatisch ein → kein Auge sichtbar
- [ ] 6.5 **Autofill + Korrektur (streng)**: nach Autofill ein Zeichen anhängen → Auge bleibt versteckt
- [ ] 6.6 **Reset nach Autofill**: nach Autofill Feld komplett leeren (Strg+A, Delete), dann neu tippen → Auge sichtbar
- [ ] 6.7 **Extension (optional, falls verfügbar)**: 1Password/Bitwarden installieren, Login auto-fillen → kein Auge sichtbar
- [ ] 6.8 **Paste via Tastatur**: aus Zwischenablage `Ctrl+V` in leeres Feld → Auge sichtbar (paste zählt als User-Aktion)
- [ ] 6.9 **Paste via Rechtsklick**: aus Zwischenablage per Rechtsklick → „Einfügen" → Auge sichtbar
- [ ] 6.10 **Mobile (Chrome Android oder iOS Safari)**: Touch-Target des Auges ≥ 44 px, Klick funktional
- [ ] 6.11 **Tastatur-A11y**: Tab durch Formular → Toggle-Button erreichbar → Space/Enter togglet → Form-Submit nicht ungewollt ausgelöst (`type="button"` korrekt gesetzt)
- [ ] 6.12 **Screenreader**: VoiceOver/NVDA sagt „Passwort anzeigen, Button" → nach Klick „Passwort verbergen, gedrückt"

## 7. Lint, Build, Deploy

- [ ] 7.1 `cd web && pnpm lint` grün *(Hinweis: vor-existierender ESLint-Konfigurationsfehler im Projekt — `eslint.config.js` fehlt. Unabhängig von dieser Änderung. Müsste separat repariert werden.)*
- [x] 7.2 `make build` grün (`pnpm build`: tsc + vite + PWA-SW alle grün, 730.93 kB Bundle)
- [x] 7.3 Bundle-Größen-Check: Eye/EyeOff aus Lucide bereits im Bundle, PasswordInput-Komponente trägt nur Komponentenlogik bei — Größenzuwachs unter 5 KB
- [ ] 7.4 Commit-Reihe (Conventional Commits): *(durch User)*
  - `feat(forms): PasswordInput-Komponente mit User-typed-Erkennung`
  - `refactor(auth): LoginPage nutzt PasswordInput`
  - `refactor(auth): RegisterPage und ResetPasswordPage nutzen PasswordInput`
  - `refactor(profile): PasswordChangeModal nutzt PasswordInput`
- [ ] 7.5 `make deploy` *(durch User)*
- [ ] 7.6 Prod-Smoke-Test auf `https://internal.team-stuttgart.org/login` *(durch User)*

## 8. Spec-Pflege

- [ ] 8.1 Neue Capability `password-reveal-toggle/spec.md` aus diesem Change wird beim Archivieren in `openspec/specs/` übernommen
- [ ] 8.2 Capability `component-standards/spec.md` wird beim Archivieren um den neuen Requirement-Block ergänzt
- [ ] 8.3 OpenSpec-Change archivieren via `/opsx:archive`
