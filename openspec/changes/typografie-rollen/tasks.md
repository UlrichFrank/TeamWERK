# Tasks

## 1. Fundament: Rollen-Konstanten und Gate

- [x] 1.1 `web/src/lib/typography.ts` anlegen: `PAGE_TITLE`, `ENTRY_TITLE`, `MODAL_TITLE`, `SECTION_TITLE`, `SUBSECTION_TITLE`, `OVERLINE`, `MENU_ITEM`, `MENU_ITEM_DANGER` mit den Werten aus design.md §2 und je einem Kommentar zur Rolle. Prüfen: `pnpm -C web exec tsc -b` ist grün.
- [x] 1.2 `web/src/lib/__tests__/typography.gate.test.ts` nach dem Vorbild von `buttonStyles.gate.test.ts`. Er prüft Überschriften-Regel und Menüeintrag-Regel (design.md §3), enthält eine Allowlist mit Verwaist-Check, Poison-Tests (handgesetzte `<h2>` wird gemeldet; `${SECTION_TITLE} mb-4 truncate` besteht; `<h2>` ohne Klasse wird gemeldet; `role="menuitem"` ohne Konstante wird gemeldet) und eine Liste `MIGRATION_PENDING`, die genau die heute verstoßenden Dateien enthält. Die Liste schrumpft je Gruppe; ein Eintrag, dessen Datei nicht mehr verstößt, lässt den Test fehlschlagen. Prüfen: `pnpm -C web exec vitest run src/lib/__tests__/typography.gate.test.ts` ist grün.
- [x] 1.3 `docs/agent/05-frontend.md`: Abschnitt „Typografie-Rollen“ mit Tabelle der Rollen (Rolle → Verwendung, ohne Strings abzutippen), Fundstelle `lib/typography.ts`, Verweis auf das Gate. Die Zeilen „Seitentitel“ und „Modal-Kopf“ unter Styling verweisen auf die Konstanten. `docs/agent/08-verification.md` bekommt einen Eintrag „Typografie-Gate“. Prüfen: Doku gelesen und ohne Widerspruch zu design.md.

## 2. Modal-Titel

- [x] 2.1 `EditModal` und alle Modal-/Dialog-Köpfe (`<h2>`/`<h3>` in Modals, zusätzlich per Suche Modal-Köpfe als `<div>`/`<p>` mit `text-lg font-(bold|semibold)`) auf `MODAL_TITLE` umstellen, Layout-Klassen bleiben. Die Dateien aus `MIGRATION_PENDING` entfernen, die danach sauber sind. Prüfen: Gate grün, `pnpm -C web test` grün.

## 3. Seiten- und Einstiegstitel

- [x] 3.1 Alle `<h1>` der App-Seiten auf `PAGE_TITLE` umstellen, inklusive des Template-Literals in `TermineDetailPage` (Zustandsklassen `line-through opacity-60` bleiben). Einstiegsseiten (`LoginPage`-„Anmelden“, `RegisterPage`, `ForgotPasswordPage`, `ResetPasswordPage`) bekommen `ENTRY_TITLE`, die „TeamWERK“-Marke auf der Login-Seite `PAGE_TITLE`. `SepaMandatViewerPage` und `FileViewer`-Kopfleiste sowie `MarkdownRenderer` kommen mit Begründung in die Allowlist. `MIGRATION_PENDING` nachziehen. Prüfen: Gate grün, `pnpm -C web test` grün.

## 4. Abschnitts-, Unterabschnitts- und Zwischenüberschriften

- [x] 4.1 Profil-Tabs (`components/profile/*`) und Mitglieder-Tabs (`components/admin/Member*Tab.tsx`) auf `SECTION_TITLE` umstellen (die bisherigen gedämpften Köpfe). `MIGRATION_PENDING` nachziehen. Prüfen: Gate grün, betroffene Vitest-Dateien grün.
- [ ] 4.2 Alle übrigen `<h2>`/`<h3>` in `pages/` und `components/` nach Rolle einordnen: Abschnitt einer Seite oder Karte → `SECTION_TITLE`, Gliederung darunter → `SUBSECTION_TITLE`, Versal-Zwischenzeile → `OVERLINE`. `MIGRATION_PENDING` nachziehen. Prüfen: Gate grün, `pnpm -C web test` grün.

## 5. Menüeinträge

- [ ] 5.1 `ActionMenu` und alle Dropdown-Menüs (Kalender „Weitere Aktionen“, Nutzerverwaltung, Veranstaltungsorte, Videos, `MemberStammdatenTab`, `ProfileProfilTab` und alle weiteren Funde von `w-full text-left px-… py-… text-…` in Menüs) auf `MENU_ITEM`/`MENU_ITEM_DANGER` umstellen und mit `role="menuitem"` versehen, wo es fehlt. `MIGRATION_PENDING` nachziehen. Prüfen: Gate grün, `ActionMenu`-Tests grün.

## 6. Frei nachgebaute Buttons und Tabs

- [ ] 6.1 Gefüllte Buttons ohne Konstante nach Position umstellen (design.md §4): `AdminDutyTemplatesPage` Split-Button → `HEADER_SPLIT_MAIN`/`HEADER_SPLIT_CARET`, `KalenderPage` „Heute“ → `BTN_PRIMARY`, `ProfileProfilTab` Bild-Button, `AdminKaderPage` Löschen → `BTN_DANGER`, `ChatPage` (Modal-Aktionen `w-full` + `BTN_PRIMARY`, Senden-Button), `ForgotPasswordPage`/`RegisterPage`/`ResetPasswordPage`/`RequestMembershipPage` → `BTN_PRIMARY` + `w-full`, `TerminePage` Z. ~1046 sowie die Funde mit `py-1.5 text-sm`/`py-1 text-sm` in `MemberStammdatenTab`, `MeinTeamPage`, `SpieltagDetailModal`, `MembersPage`, `VideosPage`, `AdminTrainingsPage`, `UpdateBanner`. Prüfen: `buttonStyles.gate.test.ts` grün, `pnpm -C web test` grün.
- [ ] 6.2 Seiten-Tabs in `AdminTrainingsPage` auf `TAB_BAR`/`TAB`/`TAB_ACTIVE`/`TAB_INACTIVE` umstellen. Prüfen: vorhandene Tests der Seite grün.

## 7. Abschluss

- [ ] 7.1 `MIGRATION_PENDING` ist leer. Liste und zugehörigen Code aus dem Gate entfernen. Prüfen: Gate grün.
- [ ] 7.2 Sichtprüfung im Browser bei 375 px und Desktop: Profil, Kalender (Menü offen), Dienstvorlagen, ein Modal, Login. Hierarchie und Kopfzeilen-Höhen sind stimmig. Prüfen: Screenshots gesichtet.
- [ ] 7.3 Volles Gate: `pnpm -C web build`, `pnpm -C web test`, `pnpm -C web lint`, `openspec validate typografie-rollen`. Alles grün.
