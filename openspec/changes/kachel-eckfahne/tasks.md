## 1. Kachelform

- [x] 1.1 `eckfahne`, `eckfahne-lg`, `fahne-*` als Utilities in `web/src/index.css` (`@property … inherits: false`)
- [x] 1.2 Alle Kacheln/Modals: `rounded-xl` → `rounded-xs`, `border-t-4` → `border-t-2 eckfahne`
- [x] 1.3 Statusfarben der Fahne (`lib/eventColors.ts`, abgesagt/vergangen grau)
- [x] 1.4 `<main>`: `sm:border-l-[3px] sm:eckfahne sm:eckfahne-lg`, keine Rundung
- [x] 1.5 Lade-Platzhalter und Benutzerhandbuch angleichen
- [x] 1.6 Gate-Test `eckfahne.gate.test.ts`, Test-Selektoren nachziehen
- [x] 1.7 Doku `docs/agent/05-frontend.md`, `06-gotchas.md`

## 2. Beta-Instanz

- [x] 2.1 `deploy/teamwerk-beta.service`, `deploy/nginx-teamwerk-beta.conf`
- [x] 2.2 Make-Targets `setup-beta`, `deploy-beta`
- [x] 2.3 Matomo auf Nicht-Prod-Hosts aus
- [x] 2.4 DNS-Record `beta.teamwerk.team-stuttgart.org` (manuell, Zone bei Mittwald)
- [x] 2.5 Zertifikat per certbot, sobald DNS auflöst
- [x] 2.6 Doku `docs/agent/10-deployment.md`
