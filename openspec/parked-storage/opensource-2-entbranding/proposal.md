## Why

TeamWERK ist aktuell hart auf **Team Stuttgart** zugeschnitten: Vereinsname, Domain (`internal.team-stuttgart.org`, CORS), Logo, Markenfarben, Begrüßungstexte und vereins-spezifische PDF-Anhänge (Satzung, Leitbild, Gebührenordnung) stecken im Code bzw. in Assets. Damit ein anderer Verein TeamWERK **selbst hosten** kann (Self-Hosted-Single-Tenant, ein Deploy pro Verein), müssen diese Werte aus **Konfiguration** statt aus Hardcoding kommen.

Bewusst **kein** Multi-Tenancy (eine Instanz für N Vereine) — das wäre ein großer Refactor des Single-DB-Modells und widerspricht dem schlanken VPS/SQLite-Ansatz. Stattdessen: jeder Verein deployt seine eigene Instanz, konfiguriert über ENV + DB-Vereinsstammdaten.

## What Changes

- **Vereinsidentität aus Config:** Vereinsname, Kurzname, Produktions-Domain, Support-/Absender-E-Mail aus ENV/`config`
- **CORS dynamisch** aus konfigurierter Domain statt hartcodiert
- **Theming konfigurierbar:** Markenfarben (`brand-*`) und Logo aus einer Instanz-Konfiguration ableitbar statt fix in `tailwind.config.js`/Assets
- **Texte austauschbar:** Begrüßungs-E-Mail-Text, Login-/Beitrittsseiten-Texte als instanz-spezifische Werte — mit neutralem Default
- **Welcome-Mail-Anhänge als Feature (NEU):** Die heute hartcodierten, eingebetteten PDFs (`satzung/gebuehrenordnung/leitbild`) entfallen; stattdessen markiert der **Vorstand im `/dokumente`-Bereich Dateien als Begrüßungs-Anhang**. Die Welcome-Mail lädt die ausgewählten Dateien aus dem Dokumenten-Store statt aus `mailer.AttachmentFS`
- **SEPA-Stammdaten** (Gläubiger-ID/IBAN/BIC/Kontoinhaber) bleiben DB-Config (bereits so), werden dokumentiert
- Default-Branding = **neutrale Demo-Identität** („Beispielverein"), kein Team-Stuttgart-Default

## Capabilities

### New Capabilities

- `instance-branding`: Vereinsidentität, Theming und austauschbare Texte/Assets einer TeamWERK-Instanz aus Konfiguration beziehen, mit neutralem Default.
- `welcome-attachments`: Vorstand wählt im Dokumente-Bereich, welche Dateien der Begrüßungs-E-Mail beiliegen — instanz-spezifisch statt hartcodiert.

### Modified Capabilities

*(keine bestehende Capability geändert — additive Konfigurationsschicht)*

## Impact

- `internal/config/` — neue Felder (Vereinsname, Domain, Branding-Pfade)
- `internal/app/router.go` — CORS aus Config statt Konstante
- `internal/mailer/` — Texte aus Config; neutrale Defaults
- `internal/members/welcome_email.go` — lädt Anhänge aus dem Dokumenten-Store statt `mailer.AttachmentFS`
- `internal/files/` — Markierung „Welcome-Anhang" je Datei (CRUD durch Vorstand)
- Neue DB-Migration (nächste freie Nummer): Flag/Verknüpfung „Datei ist Welcome-Anhang"
- Frontend `DocumentsPage.tsx` — Toggle/Markierung „als Begrüßungs-Anhang"
- Frontend: `tailwind.config.js` Theming aus Build-/Runtime-Variablen; `LoginPage`, `RequestMembershipPage`, `AppShell`, `index.html`/Manifest entbrandet
- `.env.example` dokumentiert alle Branding-Variablen
- Keine neue externe Abhängigkeit, kein zusätzlicher RAM-Footprint
- Single-Tenant bleibt — keine Schema-Änderung für Mandantenfähigkeit

## Test-Anforderungen

| Route / Einheit | Testname | Erwarteter Status / Invariante |
|---|---|---|
| CORS-Middleware | `TestCORS_AllowsConfiguredOrigin` | 200/204 + `Access-Control-Allow-Origin` = konfigurierte Domain |
| CORS-Middleware | `TestCORS_RejectsForeignOrigin` | kein `Allow-Origin`-Header für fremde Domain |
| `config` | `TestConfig_DefaultBrandingIsNeutral` | ohne ENV → neutraler Default („Beispielverein"), nie „Team Stuttgart" |
| `config` | `TestConfig_BrandingOverrideFromEnv` | ENV-Werte überschreiben Default korrekt |
| `mailer` | `TestWelcomeEmail_UsesConfiguredClubName` | Begrüßungstext enthält konfigurierten Vereinsnamen, keinen Hardcode |
| `members` (welcome) | `TestWelcomeEmail_AttachesSelectedDocuments` | nur als „Welcome-Anhang" markierte Dokumente hängen an |
| `members` (welcome) | `TestWelcomeEmail_NoSelectionNoAttachments` | keine Markierung → Mail ohne Anhang, kein Fehler |
| `POST /api/files/{id}/welcome-attachment` (o. ä.) | `TestMarkWelcomeAttachment_RequiresVorstand` | 403 für Nicht-Vorstand, 200/204 + Broadcast für Vorstand |

**Garantierte Invariante:** Ohne instanz-spezifische Konfiguration startet TeamWERK mit neutralem Default-Branding; an keiner Stelle erscheint „Team Stuttgart" im ausgelieferten Default.
