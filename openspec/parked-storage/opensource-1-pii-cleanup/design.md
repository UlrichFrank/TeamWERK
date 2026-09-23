# Design — PII-Cleanup & Public-Repo-Strategie

## Entscheidung: History-Rewrite mit `git-filter-repo`

| Kriterium | Frischer Repo (Squash) | `git-filter-repo` (Rewrite) |
|---|---|---|
| PII-Leak-Risiko aus Alt-Commits | null | **Restrisiko (übersehene Blobs)** |
| Aufwand bei 619 Commits | gering | mittel–hoch |
| Erhält Commit-Granularität | nein | **ja** |
| Erhält Mitautoren-History | nein | **ja** |
| Reversibilität bei Fehler | hoch (privat bleibt) | gering (force-push) |

**Entscheidung (vom Maintainer gewählt):** History-Rewrite via `git-filter-repo`. Die Commit-Granularität und Mitautoren-Historie bleiben erhalten. Bekannte PII-Blobs werden aus **allen 619 Commits** entfernt.

**Restrisiko & Mitigation (zwingend):** Nach `git-filter-repo` müsste man jeder Person vertrauen, die den alten Repo geklont hat; GitHub-Caches/Forks/PR-Refs können alte Blobs konservieren. Daher:

1. Rewrite gegen einen **frischen Clone** (`--no-blob-protection` vermeiden; auf Spiegel arbeiten).
2. **Pattern-Scan über die NEUE History** als Gate: `git log --all -p | grep -E '<PII-Patterns>'` muss leer sein (IBAN-Regex, reale Mail-Domains, `INSERT INTO members`, Dump-Marker).
3. Der **bisherige öffentliche** Stand (falls bereits irgendwo gepusht) gilt als kompromittiert — neuer Repo oder Force-Push + Lösch-/Neuanlage des Remote, abhängig davon, ob schon publik.
4. Da der Repo **noch nicht öffentlich** ist, ist das Restrisiko allein auf lokale Klone begrenzt — vor dem ersten Public-Push abschließen.

## Konkrete `git-filter-repo`-Pfade (Entfernung aus gesamter History)

```
teamwerk_dump.sql                         # bestätigt: echter Produktiv-Dump
storage/files/                            # vereins-Dokumente (→ Feature in ②)
internal/mailer/attachments/*.pdf         # club-PDFs (→ Feature in ②)
deploy/stammverein-mapping-*.sql          # PII-unkritisch, aber vereins-intern
Gebuehrenordnung.pdf                      # Repo-Root
"OpenSpec Proposal_ SEPA-Beitragslauf.pdf"
sepa-beitragslauf-proposal.html
*.af / Handball.af / IconAndroid.af       # Affinity-Arbeitsdateien (bereits gitignored)
```
`testdata/*.csv` werden **nicht** entfernt (synthetisch, bleiben — IBANs werden auf Test-Bereich umgestellt, siehe Tasks).

## PII-Datenklassen (was niemals public darf)

1. **Echte Personendaten** — Namen realer Mitglieder, Adressen, Geburtsdaten, Telefon, E-Mail
2. **Finanzdaten** — IBAN, BIC, Kontoinhaber, SEPA-Mandatsreferenzen, Gläubiger-ID
3. **Hochgeladene Dokumente** — `storage/files/*` (Mandate, Einwilligungen, Atteste)
4. **DB-Artefakte** — `*.db`, `*.db-wal`, `*.db-shm`, `*_dump.sql`
5. **Vereins-Stammdaten mit Personenbezug** — `deploy/stammverein-mapping-*.sql`

## Synthetische Daten als Ersatz

Wo echte Daten Tests/Seed dienten, treten **eindeutig synthetische** Daten an ihre Stelle:
- IBANs aus dem offiziellen Test-IBAN-Bereich (z. B. `DE…` mit dokumentiert-fiktiven Prüfziffern)
- Namen aus einem Fantasie-Pool, klar als Demo erkennbar (`Beispielverein`, `Erika Mustermann`)
- Detaillierte Seed-Daten → Teil von ③ (Self-Hosting-Demo)

## Mechanischer Guard

Pre-Commit-Schritt scannt gestagete Dateien gegen PII-Pattern (IBAN-Regex, `*_dump.sql`, `*.db`, CSV mit Adress-/Geburtsdatum-Spaltenköpfen) und bricht den Commit ab. Verhindert Rückfall nach der Bereinigung.

## Verifikationen (durchgeführt 2026-06-21)

- [x] `teamwerk_dump.sql` ist ein **Echt-Dump** — 199 `members`-Inserts, 183 reale Mail-Adressen (u. a. `andrea@diefranks.eu`, „Andrea Frank"). → Entfernen.
- [x] `testdata/*.csv` wirken **synthetisch** (generische Namen Müller/Schmidt). → Behalten; IBANs auf nachweislichen Test-Bereich umstellen.
- [x] `storage/files/*.pdf` sind **Vereins-Dokumente** (Willkommensbrief, Beitragsordnung, Satzung), **kein** Personen-PII. → Aus Repo entfernen; künftig instanz-konfigurierbar (Feature in ② / `/dokumente`).
- [x] `deploy/stammverein-mapping-*.sql` **ohne Personenbezug** (0 IBAN/Mail/Geburtsdatum). → Vereins-intern, dennoch aus Public-Repo entfernen.
