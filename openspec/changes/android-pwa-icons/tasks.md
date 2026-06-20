## 0. Vorbedingung

- [x] 0.1 `origin/main` pullen/rebasen (lokaler `main` war hinter origin; PR #46 `chat-unread-app-badge` ist dort gemerged und ändert denselben `sw.ts`-Push-Handler) — alle folgenden Tasks auf diesem Stand umsetzen

## 1. Quell-SVGs & Generierungsskript

- [x] 1.1 `IconAndroid.svg` und `Handball.svg` aus dem Repo-Root nach `web/icon-src/` verschieben (`git mv`)
- [x] 1.2 `scripts/gen-icons.sh` anlegen: rendert via `rsvg-convert` `icon-maskable-512.png` (`-w 512 -h 512 -b '#FFFFFF'` aus `IconAndroid.svg`) und `badge-96.png` (`-w 96 -h 96`, transparent, aus `Handball.svg`) nach `web/public/icons/`; Skript ausführbar (`chmod +x`), mit Kommentar zur Voraussetzung `rsvg-convert`
- [x] 1.3 Skript einmal ausführen und die zwei PNGs erzeugen

## 2. Assets verifizieren (Augenschein)

- [x] 2.1 `icon-maskable-512.png` prüfen: gelber Kreis zentriert, weiße Ecken (keine Transparenz), Logo-Inhalt innerhalb der 80 %-Safe-Zone
- [x] 2.2 `badge-96.png` prüfen: transparenter Hintergrund, Pferd-im-Ball als zusammenhängende Form (auf einfarbigem Grund gegengeprüft → als reine Silhouette erkennbar)
- [x] 2.3 Beide generierten PNGs committen

## 3. Manifest konsolidieren

- [ ] 3.1 `web/vite.config.ts`: `manifest.icons` auf drei Einträge umstellen — `icon-192` (`purpose: 'any'`), `icon-512` (`purpose: 'any'`), `icon-maskable-512` (`purpose: 'maskable'`); kein `'any maskable'` mehr
- [ ] 3.2 `web/index.html`: `<link rel="manifest" href="/manifest.json" />` entfernen (VitePWA injiziert den Manifest-Link selbst); `<link rel="apple-touch-icon">` bleibt unverändert
- [ ] 3.3 `web/public/manifest.json` löschen (`git rm`)
- [ ] 3.4 `pnpm -C web build` und prüfen, dass das generierte `manifest.webmanifest` die drei Icon-Einträge mit korrekten Purposes enthält und genau ein `<link rel="manifest">` im gebauten `index.html` steht

## 4. Service Worker: Badge

- [ ] 4.1 `web/src/sw.ts`: im `push`-Handler `badge: '/icons/icon-192.png'` → `badge: '/icons/badge-96.png'` ändern; `icon:` bleibt `/icons/icon-192.png` (bunte Vorschau)

## 5. Regressions-Tests (Vitest)

- [ ] 5.1 `web/src/test/icons.test.ts` anlegen: assert `web/public/icons/icon-maskable-512.png` und `badge-96.png` existieren und > 1 KB groß sind
- [ ] 5.2 Assert Konsolidierung: `web/public/manifest.json` existiert NICHT; `web/index.html` enthält kein `rel="manifest"`
- [ ] 5.3 Assert Purposes: `web/vite.config.ts` enthält genau einen `purpose: 'maskable'`-Eintrag und keinen `'any maskable'`
- [ ] 5.4 Assert SW: `web/src/sw.ts` referenziert `/icons/badge-96.png`
- [ ] 5.5 `pnpm -C web test` grün

## 6. Manuelle Geräteverifikation (Android)

- [ ] 6.1 PWA auf Android zum Homescreen hinzufügen → Logo vollständig sichtbar, kein abgeschnittener Schriftring
- [ ] 6.2 Push-Notification auslösen → Status-Bar zeigt Pferd-Silhouette statt weißem Kreis; aufgeklappte Notification zeigt das bunte App-Icon
- [ ] 6.3 Kurzcheck iOS (sofern Gerät verfügbar): Homescreen-Icon und Push unverändert wie vorher (keine Regression durch die Manifest-Konsolidierung)

## 7. Abschluss

- [ ] 7.1 `openspec validate android-pwa-icons --strict`
- [ ] 7.2 `/verify-change` (Build/Test/Lint + Invarianten)
- [ ] 7.3 CHANGELOG-Eintrag ergänzen
