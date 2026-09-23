# Tasks — Entbranding & Instanz-Konfiguration

## 1. Konfigurations-Schicht (Backend)
- [ ] 1.1 `internal/config`: Felder `ClubName`, `ClubShort`, `PublicURL`, `MailFrom`, `SupportEmail` + neutrale Defaults
- [ ] 1.2 Test `TestConfig_DefaultBrandingIsNeutral` + `TestConfig_BrandingOverrideFromEnv`
- [ ] 1.3 `.env.example` um alle Branding-Variablen erweitern (dokumentiert)

## 2. CORS dynamisch
- [ ] 2.1 `internal/app/router.go`: CORS-Origin aus `config.PublicURL` statt Konstante
- [ ] 2.2 Tests `TestCORS_AllowsConfiguredOrigin` + `TestCORS_RejectsForeignOrigin`

## 3. Mailer-Texte entbranden
- [ ] 3.1 `welcome_email.go` + `mailer.go`: Vereinsname/Texte aus Config
- [ ] 3.2 Test `TestWelcomeEmail_UsesConfiguredClubName`

## 3b. Welcome-Mail-Anhänge aus /dokumente (Feature)
- [ ] 3b.1 DB-Migration (nächste freie Nummer): Markierung „Datei = Welcome-Anhang"
- [ ] 3b.2 `internal/files`: Route zum Markieren/Entmarkieren (Vorstand/admin) + `Broadcast`
- [ ] 3b.3 `welcome_email.go`: Anhänge aus Dokumenten-Store laden statt `mailer.AttachmentFS`; hartcodierte PDFs/`logo.svg`-Liste entfernen
- [ ] 3b.4 Frontend `DocumentsPage.tsx`: Markierung „als Begrüßungs-Anhang" + `useLiveUpdates`
- [ ] 3b.5 Tests: `TestMarkWelcomeAttachment_RequiresVorstand`, `TestWelcomeEmail_AttachesSelectedDocuments`, `TestWelcomeEmail_NoSelectionNoAttachments`

## 4. Frontend entbranden
- [ ] 4.1 Theming: Markenfarben aus Build-Variablen in `tailwind.config.js` injizieren (Strategie A)
- [ ] 4.2 Die 2 Streu-Hex-Stellen auf `brand-*`-Tokens umstellen
- [ ] 4.3 Logo als austauschbares Asset mit neutralem Default
- [ ] 4.4 `LoginPage`, `RequestMembershipPage`, `AppShell`, `index.html`, `manifest.json` entbranden
- [ ] 4.5 `pnpm -C web build` grün, kein „Team Stuttgart" im Default-Build

## 5. Verifikation
- [ ] 5.1 Default-Build manuell prüfen: neutrales Branding durchgängig
- [ ] 5.2 Beispiel-Override (Demo-Verein) dokumentieren, nicht eingecheckte personenbezogene Werte
- [ ] 5.3 `/verify-change` (brand-Tokens, lucide-Icons, Tests, Build) grün
