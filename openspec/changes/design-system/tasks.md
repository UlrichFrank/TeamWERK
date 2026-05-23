## 1. Token-Layer

- [x] 1.1 `web/tailwind.config.js`: neue semantische Tokens eintragen (`brand-surface-card`, `brand-text`, `brand-text-muted`, `brand-text-subtle`, `brand-border`, `brand-border-subtle`, `brand-danger`, `brand-danger-light`, `brand-info`)
- [ ] 1.2 Vite-Dev-Server starten und prüfen, dass alle neuen Token-Klassen korrekt gerendert werden

## 2. Globale Komponenten

- [x] 2.1 `AppShell.tsx`: Unicode `☰` → `<Menu>`, `✕` → `<X>`, `▸` → `<ChevronRight>`, `▾` → `<ChevronDown>` ersetzen; aria-label auf Icon-only-Buttons setzen
- [x] 2.2 `ActionMenu.tsx`: `⋮` → `<MoreVertical w-4 h-4>`; Button-Klassen auf Standard anpassen
- [x] 2.3 `MobileCard.tsx`: Klassen auf `brand-*`-Tokens migrieren
- [x] 2.4 `EditModal.tsx`: Modal-Klassen auf `border-t-4 border-brand-yellow`-Standard anpassen; Schließen-Button `aria-label="Schließen"` + `<X w-5 h-5>`
- [x] 2.5 `Pagination.tsx`: `«` → `<ChevronsLeft>`, `»` → `<ChevronsRight>`; Button-Klassen vereinheitlichen
- [x] 2.6 `Accordion.tsx`: Chevrons bereits Lucide — Klassen auf `brand-text-muted` prüfen und anpassen

## 3. Shared-Komponenten

- [x] 3.1 `BrandCheckbox.tsx`: custom Inline-SVG durch Lucide-Icon ersetzen (z.B. `SlidersHorizontal`); Klassen auf Token-Standard
- [x] 3.2 `AutoAssignModal.tsx`: Button-Klassen auf Primary/Danger-Standard, Modal auf `border-t-4 border-brand-yellow`, Inputs auf Standard-Input-String
- [x] 3.3 `CopyKaderModal.tsx`: gleiche Migration wie AutoAssignModal
- [x] 3.4 `PositionStatus.tsx`: raw Farben durch `brand-*`-Tokens ersetzen
- [x] 3.5 `KaderMemberSearch.tsx` / `KaderTrainerSearch.tsx`: Input-Klassen auf Standard-Input-String

## 4. Auth-Seiten

- [x] 4.1 `LoginPage.tsx`: Input-Klassen und Button-Klassen auf Standard bringen
- [x] 4.2 `RegisterPage.tsx`: Input- und Button-Klassen; Alert-Klassen auf `brand-danger-light`
- [x] 4.3 `ForgotPasswordPage.tsx`: Input- und Button-Klassen
- [x] 4.4 `ResetPasswordPage.tsx`: Input- und Button-Klassen
- [x] 4.5 `RequestMembershipPage.tsx`: Card-, Input-, Button- und Alert-Klassen

## 5. Admin-Seiten

- [x] 5.1 `AdminClubPage.tsx`: Card-Standard, Input-Standard, Button unten (Formular → „Speichern")
- [x] 5.2 `AdminSeasonsPage.tsx`: Card-Standard, Input-Standard, Button oben rechts neben h1 → „Neue Saison"; Danger-Button für Deaktivierung
- [x] 5.3 `AdminTeamsPage.tsx`: Card-Standard, Input-Standard, Button oben rechts neben h1; Tabellen-Standard; `✓`/`✗` → `<Check>/<X>`
- [x] 5.4 `AdminDutyTypesPage.tsx`: Card-Standard, Input-Standard, Button oben rechts neben h1; Tabellen-Standard; Mobile-Modal auf `border-t-4`-Standard
- [x] 5.5 `AdminDutyTemplatesPage.tsx`: Card-Standard, Button oben rechts (bereits vorhanden — nur Klassen prüfen); Tabellen-Standard; `⚠` → `<AlertTriangle>`
- [x] 5.6 `AdminDutyTemplateDetailPage.tsx`: Card-Standard, Input-Standard, Button-Klassen, `✕` → `<X>`, `🗑` → `<Trash2>`; Danger-Buttons für Löschen
- [x] 5.7 `AdminKaderPage.tsx`: Tabellen-Standard, Input-Standard, Button-Klassen; `✓`/`✗` → `<Check>/<X>`
- [x] 5.8 `AdminUsersPage.tsx`: Tabellen-Standard, Input-Standard, Button oben rechts neben h1; Danger-Button für Sperren/Löschen

## 6. Mitglieder-Seiten

- [x] 6.1 `MembersPage.tsx`: Tabellen-Standard, Input-Standard (Suche), Button oben rechts neben h1
- [x] 6.2 `MemberDetailPage.tsx`: Card-Standard, Input-Standard, Button-Klassen; `✓`/`✗` → `<Check>/<X>`

## 7. Dienst-Seiten

- [x] 7.1 `DutyPage.tsx`: Card-Standard, Tabellen-Standard, Button-Klassen; Alert-Standard; Danger-Buttons
- [x] 7.2 `MembershipRequestsPage.tsx`: Card-Standard, Button-Klassen; Danger-Button für Ablehnen (`✗` → `<X>`, `✓` → `<Check>`)

## 8. Spielplan-Seiten

- [x] 8.1 `SpielplanPage.tsx`: Card-Standard, Tabellen-Standard, Button-Klassen; `📋`→`<Calendar>`, `⚽`→`<Home>`, `✈`→`<MapPin>`, `⚠`→`<AlertTriangle>`
- [x] 8.2 `SpieltagDetailPage.tsx`: Card-Standard, Input-Standard, Button-Klassen; `🗑`→`<Trash2>`, `⚠`→`<AlertTriangle>`; Danger-Buttons für Löschen

## 9. Profil-Seiten

- [x] 9.1 `ProfilePage.tsx`: Card-Standard, Input-Standard, Button-Klassen; Alert-Standard

## 10. Dashboard-Migration

- [x] 10.1 `DashboardPage.tsx`: Statusbadges auf Brand-Tokens migrieren (`bg-green-100`→`bg-brand-success-light`, `bg-yellow-100`→`bg-brand-warning-light`, `bg-blue-100`→`bg-brand-info/10`)
- [x] 10.2 `DashboardPage.tsx` — TeamStats: `text-red-500`→`text-brand-danger`, `text-yellow-500`→`text-brand-warning`, `text-brand-green` bleibt
- [x] 10.3 `DashboardPage.tsx` — Muted-Text: alle `text-black/50`→`text-brand-text-muted`, `text-black/40`→`text-brand-text-subtle`
- [x] 10.4 `DashboardPage.tsx` — NextGamesList: `🏠`→`<Home w-4 h-4>`, `🚌`→`<MapPin w-4 h-4>`
- [x] 10.5 `DashboardPage.tsx` — Skeleton-Loader: `bg-black/5`→`bg-brand-border-subtle`

## 11. Abschluss

- [x] 11.1 `pnpm run build` ohne TypeScript-Fehler
- [ ] 11.2 Visuellen Check im Browser: Dashboard, Mitgliederliste, Admin-Seite, Spieltag-Detail
- [ ] 11.3 Mobile-Check (Hamburger, Sidebar-Overlay, Card-Layout in Tabellen)
