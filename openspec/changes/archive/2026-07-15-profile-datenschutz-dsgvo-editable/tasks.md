## 1. Spec-Nachführung

- [x] 1.1 Requirement „DSGVO-Anzeige (read-only)" in `profile-datenschutz-tab` durch
      neue Fassung „DSGVO-Einwilligungen mit Change-Request" ersetzen
      (Delta in `specs/profile-datenschutz-tab/spec.md`).
- [x] 1.2 Szenarien aktualisieren: Draft-Ausstehend, Anfrage-Button-Sperre ohne
      Diff, Zurückziehen.

## 2. Verifikation

- [x] 2.1 `openspec validate profile-datenschutz-dsgvo-editable --strict` grün.
- [x] 2.2 Bestehender vitest-Suite grün (dokumentiert Ist-Verhalten seit
      `7e1a91e` + Test-Fix).

## 3. Archivierung

- [ ] 3.1 Nach Merge in `main`: `openspec archive profile-datenschutz-dsgvo-editable`
      — dabei wird die neue Requirement-Fassung in `openspec/specs/…` übernommen.
