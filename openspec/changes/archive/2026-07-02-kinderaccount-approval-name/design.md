## Context

`approveChildRequest` legt für einen `is_child=1`-Antrag ein Proxy-Kinder-Konto an (`email=NULL`, `login_name`, `can_login=0`). Das INSERT lässt `first_name`/`last_name` weg, obwohl die Werte als Parameter vorliegen. Alle namensanzeigenden Pfade lesen aus `users.first_name`/`last_name`, daher bleibt der Name leer (Liste, Impersonate-Anzeige, später abgeleitetes Mitglied).

## Goals / Non-Goals

**Goals:**
- Neu angelegte Kinder-Konten tragen den echten Kindnamen in `users`.
- Bestehende namenlose Kinder-Konten werden verlässlich nachgefüllt.
- Kein Regress am Bestandsverhalten (login_name, can_login=0, Eltern-Mail, kein family_link).

**Non-Goals:**
- Keine Änderung der Impersonate-Regel (`can_login=0` bleibt ohne „Testen als").
- Keine Mitglieds-Erzeugung beim Approve (Approve legt bewusst nur ein Konto an — siehe D5).
- Kein Umbau der login_name-Generierung.

## Decisions

**D1 — Namensquelle ist der Antrag, nicht der `login_name`.** `login_name` entsteht über `normalizeLoginName` (Umlaut-Transliteration, `[A-Za-z0-9-]`-Strip, optionales Kollisions-Suffix, `internal/auth/loginname.go`) und ist damit **verlustbehaftet** — „Müller-Schäfer" → `Mueller-Schaefer`, „Lena.Schmidt2" bei Kollision. Für neue Konten liegen `firstName`/`lastName` ohnehin im Handler vor; für den Backfill ist `membership_requests` (Originalname, nach Approve als `status='approved'` erhalten) die einzige verlustfreie Quelle.

**D2 — Code-Fix: Spalten ins INSERT aufnehmen.** In `approveChildRequest`:

```sql
INSERT INTO users (email, login_name, first_name, last_name, password, role, can_login, recovery_email)
  VALUES (NULL, ?, ?, ?, '', 'standard', 0, ?)
```
Parameter-Reihenfolge: `loginName, firstName, lastName, recoveryEmail`. Minimaler, wurzelnaher Eingriff — Liste, Impersonate-Anzeige und späteres `CreateMemberFromUser` sind damit alle mitgeheilt.

**D3 — Backfill konservativ, nur bei eindeutigem Match.** Migration `016` aktualisiert nur Konten mit `can_login=0 AND email IS NULL AND login_name IS NOT NULL AND (first_name IS NULL OR first_name='')`. Match gegen `membership_requests` über `LOWER(recovery_email)=LOWER(parent_email) AND is_child=1 AND status='approved'`. Zur Disambiguierung bei mehreren Kindern derselben Eltern-Adresse zusätzlich `LOWER(login_name)=LOWER(first_name||'.'||last_name)` (greift für ASCII-Namen ohne Suffix). Rows ohne eindeutigen Match bleiben unverändert (kein falsches Ratewerk) und werden ggf. manuell korrigiert. Angesichts des einen bekannten Prod-Kontos ist Ambiguität praktisch ausgeschlossen.

**D4 — `down`-Migration ist ein dokumentierter No-op.** Ein Backfill von `''`-Werten ist nicht sinnvoll reversibel (der vorherige Zustand war „leer" = der Bug). Die `.down.sql` enthält nur einen erklärenden Kommentar.

**D5 — Spec an den Code angleichen: Approve legt nur ein Konto an.** Die bisherige Requirement „Approve … erzeugt Konto, **Mitglied** und Eltern-Mail" beschrieb (Statement + drei Szenarien) die Anlage eines verknüpften `members`-Datensatzes. Der Code legt bewusst **kein** Mitglied an (`handler.go:493-497`). Bestätigt: „Mitglied" war eine Fehlformulierung — der Approve legt nur einen Nutzer an. Die Requirement wird daher per `REMOVED` + korrigiertem `ADDED` ersetzt (Header ohne „Mitglied", Szenario „Kein Mitglied und kein automatischer Eltern-Link"). Reine Spec-Korrektur, kein Code-Verhalten geändert.

## Risks / Trade-offs

- **[Backfill trifft ein Konto nicht]** (Umlaut-/Suffix-`login_name` + mehrdeutige Eltern-Adresse) → bewusst lieber leer lassen als falsch füllen; manueller Fix als Fallback. Bei einem bekannten Prod-Konto vernachlässigbar.

## Open Questions

- Keine offen.
