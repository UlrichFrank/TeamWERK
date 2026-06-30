# Halber Beitrag bei Ein-/Austritt und im ersten Abrechnungsjahr

## Why

Der SEPA-Beitragslauf rechnet heute **bewusst** jeden eingeschlossenen Mitglied mit dem
**vollen** Jahresbeitrag ab (keine anteilige Berechnung). Das ist beitragsrechtlich zu
grob: Wer unterjährig ein- oder austritt, war nur einen Teil des Abrechnungsjahres
Mitglied und soll nicht den vollen Jahresbeitrag zahlen. Zusätzlich soll der **erste
Beitragslauf des Vereins** als einmalige Startkonzession nur den halben Beitrag erheben.

Die Satzung kennt dafür **keine monatsgenaue** Anteilsregel, sondern einen pauschalen
**halben Beitrag** (exakt 50 %). Das passt zum „bewusst einfach"-Prinzip des Laufs.

## What Changes

- **Exakte Halbierung (50 %)** des Jahresbeitrags eines Mitglieds, wenn **mindestens eine**
  der folgenden Bedingungen zutrifft (Ermäßigungen **stapeln nicht** — halb ist die
  Untergrenze, nie ein Viertel):
  - **R3 Eintritt im Abrechnungsjahr:** `join_date` liegt im Saisonfenster
    `[start_date, end_date]`.
  - **R2 Austritt im Abrechnungsjahr:** Mitglied ist `ausgetreten` und `exit_date` liegt
    im Saisonfenster.
  - **R4 Erstes Abrechnungsjahr (einmalig):** Die Saison ist als `is_inaugural` markiert →
    **alle** zahlen halb.
- **Inklusions-Wechsel für unterjährige Austritte:** Bisher werden `ausgetreten`-Mitglieder
  vollständig aus dem Lauf ausgeschlossen. Neu: Wer **im laufenden Abrechnungsjahr**
  ausgetreten ist (`exit_date` im Saisonfenster), wird **wieder einbezogen** und mit dem
  halben Beitrag abgerechnet. Wer früher ausgetreten ist (oder ohne `exit_date`), bleibt
  ausgeschlossen.
- **Schema:** neue Spalte `members.exit_date DATE` (Austrittsdatum) und
  `seasons.is_inaugural INTEGER NOT NULL DEFAULT 0`.
- **Eintrittsdatum verpflichtend:** `members.join_date` wird für neu angelegte/bearbeitete
  Mitglieder zum **Pflichtfeld** (App-Validierung). Bestandsmitglieder erhalten per
  Migration ein implizites Eintrittsdatum **vor** dem ersten regulären Saisonstart, damit
  R3 für sie nie greift (= voller Beitrag ab Jahr 2; im Erstjahr ohnehin halb via R4).
- **UI:** Austrittsdatum-Feld im Mitglieds-Edit (Pflicht bei Status `ausgetreten`);
  „Erstes Abrechnungsjahr"-Schalter in der Saisonverwaltung; Hinweis-Badge in der
  Beitragslauf-Vorschau („halber Beitrag — Eintritt/Austritt/erstes Jahr").

## Impact

- **Affected specs:** `sepa-beitragslauf` (MODIFIED: Voller-Jahresbeitrag-Regel,
  Ein-/Ausschlussregeln, Beitragsberechnung; ADDED: Halbierungsregel, Pflicht-Eintrittsdatum).
- **Affected code:**
  - `internal/db/migrations/014_*` (neu)
  - `internal/beitragslauf/` (`compute.go`, `query.go`, `handler.go`)
  - `internal/members/handler.go` (exit_date CRUD, join_date-Pflicht)
  - `internal/config/` (seasons: is_inaugural)
  - `web/src/pages/MemberDetailPage.tsx`, Saison-Admin-Seite, `BeitragslaufPage.tsx`
- **Migration:** additiv + Backfill von `join_date`; nullbar in der DB, Pflicht nur in der
  App-Validierung → kein Zwangs-Backfill von Altdaten nötig.
- **Bestehende Tests:** `TestPreview_NeumitgliedZahltVollenBeitrag` kehrt sich um
  (Neumitglied zahlt jetzt **halb**).
