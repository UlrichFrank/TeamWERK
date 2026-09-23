## Why

Bevor TeamWERK als Open-Source-Projekt (AGPL-3.0) öffentlich wird, muss sichergestellt sein, dass **kein einziger personenbezogener Datensatz** im öffentlichen Repository landet — weder im aktuellen Tree noch in der Git-Historie. Beim Open-Sourcing wird die **gesamte Historie** öffentlich, nicht nur der HEAD.

Ein Audit (2026-06-21) hat den getrackten Stand **und** die Historie (619 Commits) geprüft:

- `teamwerk_dump.sql` (394 KB) — **bestätigter Produktiv-DB-Dump**: 199 `members`-Inserts, 183 reale E-Mail-Adressen (u. a. `andrea@diefranks.eu`). Getrackt + in Historie (`780e93b`). → muss raus.
- `storage/files/*.pdf` — **Vereins-Dokumente** (Willkommensbrief, Beitragsordnung, Satzung), **kein** individuelles Personen-PII, aber vereins-spezifisch/branded. → raus, künftig instanz-konfigurierbar (Feature in ②).
- `deploy/stammverein-mapping-*.sql` — **kein Personenbezug** (0 IBAN/E-Mail/Geburtsdatum), nur Vereinsnamen-Mappings. → vereins-intern, dennoch raus.
- `testdata/test_mitglieder.csv` / `test_eltern.csv` — **synthetisch** (generische Namen), enthalten aber IBANs. → bleiben; IBANs werden auf nachweislichen Test-Bereich umgestellt.
- Repo-Root-PDFs/HTML (`Gebuehrenordnung.pdf`, OpenSpec-SEPA) und Affinity-`*.af`. → raus.

Der `teamwerk_dump.sql` allein macht ein Veröffentlichen ohne Bereinigung zum DSGVO-Verstoß. Dieser Change ist **harter Vorgänger** für alle weiteren Open-Source-Pakete.

## What Changes

- **Vollständiges PII-Audit** aller getrackten Dateien und der Git-Historie
- Verifikation, ob `teamwerk_dump.sql` und `testdata/*.csv` echte oder synthetische Daten enthalten
- **Strategie:** History-Rewrite via `git-filter-repo` — entfernt bekannte PII-Blobs aus allen 619 Commits, erhält Commit-Granularität (Detail + Restrisiko-Mitigation in `design.md`)
- Aufbau eines **sauberen, PII-freien Tree und einer bereinigten History** als Grundlage des Public-Repos
- `testdata/*.csv` bleiben (synthetisch); IBANs werden auf nachweislichen Test-Bereich umgestellt
- `.gitignore`- und Pre-Commit-Guard, der künftige PII-Commits (DB-Dumps, CSV mit IBAN/Adressen) blockiert
- PII-Audit-Checkliste als wiederverwendbares Artefakt im Repo

## Capabilities

### New Capabilities

- `public-repo-hygiene`: Garantien darüber, welche Datenklassen niemals im öffentlichen Repo (Tree oder Historie) erscheinen dürfen, plus mechanischer Guard dagegen.

### Modified Capabilities

*(keine)*

## Impact

- **Kein Anwendungscode betroffen** — reine Repo-/Prozess-Arbeit (das aus den entfernten PDFs entstehende Feature liegt in ②)
- Bereinigte History via `git-filter-repo`; bekannte PII-Blobs aus allen Commits entfernt
- Entfernt aus gesamter History: `teamwerk_dump.sql` (Echt-Dump, bestätigt), `storage/files/*`, `internal/mailer/attachments/*.pdf`, `deploy/stammverein-mapping-*.sql`, Repo-Root-PDFs/HTML, Affinity-`*.af`
- Bleibt: `testdata/*.csv` (synthetisch, IBANs werden auf Test-Bereich umgestellt)
- Erweitert: `.gitignore`, neuer Pre-Commit-Hook-Schritt (PII-Pattern-Scan)
- Risiko bei Fehlentscheidung: irreversibler PII-Leak — daher Audit + Vier-Augen vor Push
