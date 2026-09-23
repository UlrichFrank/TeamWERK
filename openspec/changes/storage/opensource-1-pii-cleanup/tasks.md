# Tasks — PII-Cleanup & Public-Repo

## 1. Audit & Verifikation (durchgeführt 2026-06-21)
- [x] 1.1 `teamwerk_dump.sql` = Echt-Dump (199 members, 183 reale Mails) → entfernen
- [x] 1.2 `testdata/*.csv` synthetisch → behalten, IBANs auf Test-Bereich (Task 2.x)
- [x] 1.3 `storage/files/*.pdf` = Vereins-Dokumente (Willkommen/Beitragsordnung/Satzung), kein Personen-PII → entfernen + Feature in ②
- [x] 1.4 `deploy/stammverein-mapping-*.sql` ohne Personenbezug → dennoch entfernen
- [x] 1.5 Repo-Root-PDFs/HTML + `*.af` → entfernen
- [ ] 1.6 Audit-Ergebnis als `docs/pii-audit-checklist.md` festhalten (wiederverwendbar)

## 2. Bereinigung des Tree
- [ ] 2.1 Bestätigte Echt-Daten + vereins-interne Dateien aus dem Tree entfernen (siehe Pfadliste design.md)
- [ ] 2.2 `testdata/*.csv`: IBANs gegen Test-Bereich prüfen, ggf. auf dokumentiert-fiktive Test-IBANs umstellen (Namen sind bereits generisch)
- [ ] 2.3 `make test` läuft mit den (synthetischen) testdata grün

## 3. Guard gegen Rückfall
- [ ] 3.1 `.gitignore` um Public-relevante PII-Klassen erweitern
- [ ] 3.2 Pre-Commit-Schritt: PII-Pattern-Scan (DB/Dump/CSV/IBAN) hinzufügen
- [ ] 3.3 Guard-Test: ein bewusst kontaminierter Test-Commit wird abgelehnt

## 4. History-Rewrite & Public-Repo
- [ ] 4.1 `git-filter-repo` auf frischem Mirror-Clone mit dokumentierter Pfadliste (design.md)
- [ ] 4.2 Pattern-Scan-Gate über die NEUE History (`git log --all -p | grep -E '<PII-Patterns>'` = leer)
- [ ] 4.3 Vier-Augen-Review der bereinigten History (Stichprobe + Scan-Ergebnis)
- [ ] 4.4 Erst nach grünem Gate öffentlich pushen; vorher als nicht-öffentlich bestätigen
- [ ] 4.5 Falls bereits irgendwo gepusht: alten Remote als kompromittiert behandeln (neu anlegen)
