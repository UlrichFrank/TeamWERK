## Context

`members.member_number` ist `TEXT`, nullable, mit `UNIQUE INDEX idx_members_member_number … WHERE member_number IS NOT NULL` (Dubletten DB-seitig verhindert). Der Create-Handler (`internal/members/handler.go` ~Z.323-364) vergibt `MAX(CAST(member_number AS INTEGER))+1` nur, wenn das Feld leer ist; sonst übernimmt er den Client-Wert. Der Update-Handler (~Z.545-631) schreibt `member_number` direkt aus dem Request und leert es bei Status `honorar`. Das Frontend (`MemberStammdatenTab.tsx` ~Z.251-260) zeigt ein frei editierbares Textfeld. Lokale DB: 198 Mitglieder, keine Dubletten/nicht-numerischen Werte, Bereich 2..285, 21 ohne Nummer (13 `honorar` korrekt, 8 Nicht-`honorar` = Konflikte). Lokale DB ist evtl. veraltet ggü. Produktion.

## Goals / Non-Goals

**Goals:**
- Mitgliedsnummer immer systemseitig vergeben (höchste numerische + 1), Client-Wert beim Anlegen ignorieren.
- Nummer read-only; nur `admin` darf nachträglich korrigieren; Dubletten als 409 statt DB-Fehler.
- Nummern-Konflikte (Dublette / nicht-numerisch / fehlend bei Nicht-`honorar`) in der `/mitglieder`-Übersicht sichtbar machen.

**Non-Goals:**
- Kein automatischer Backfill fehlender Nummern (Admin korrigiert manuell).
- Keine Lücken-Wiederverwendung.
- Keine Schema-/Migrations-Änderung (Unique-Index besteht bereits).
- Kein Eingriff in CSV-Import-Verhalten über das Nötige hinaus.

## Decisions

- **Auto-Vergabe als Helper:** `nextMemberNumber(ctx, db) (string, error)` kapselt `SELECT MAX(CAST(member_number AS INTEGER)) FROM members WHERE member_number GLOB '[0-9]*'`. `GLOB`-Filter stellt sicher, dass nicht-numerische Altwerte die Max-Bestimmung nicht verfälschen (CAST von `"M-100"` ergibt sonst 0, das wäre zwar unkritisch, der Filter macht die Absicht aber explizit). Create ruft den Helper **immer** auf und ignoriert `req.MemberNumber`.
  - *Alternative:* Lücken füllen (kleinste freie Nummer) — verworfen, Anforderung ist explizit „höchste + 1".
- **Override nur für Admin im Update-Handler:** `claims := auth.ClaimsFromCtx(r.Context())`; `member_number` wird nur dann in das `UPDATE` übernommen, wenn `claims.Role == "admin"`. Für Nicht-Admins wird der bestehende DB-Wert beibehalten (vor dem UPDATE lesen oder `member_number` aus dem SET weglassen). `honorar`-Logik bleibt unverändert.
  - *Alternative:* Route komplett für Nicht-Admins sperren — verworfen, Vorstand muss andere Felder weiter editieren.
- **409 statt DB-Fehler:** Vor dem Admin-Override prüfen, ob die Zielnummer bereits einem anderen Mitglied gehört (`SELECT id FROM members WHERE member_number=? AND id<>?`). Falls ja → `http.Error(w, …, http.StatusConflict)`. Der Unique-Index bleibt als zweite Verteidigungslinie.
- **Konflikt-Flag im List-Endpoint:** `GET /api/members` liefert pro Item ein zusätzliches Feld (z.B. `member_number_conflict: "duplicate" | "non_numeric" | "missing" | ""`). Berechnung in Go aus der bereits geladenen Liste bzw. per kleiner Zusatzabfrage für Dubletten — kein neuer Endpoint, damit die Übersicht ohne zweiten Request auskommt. Honorar (kein Nummernzwang) wird vom Typ `missing` ausgenommen.
  - *Alternative:* separater `GET /api/members/number-conflicts` — verworfen (Übersicht braucht die Info ohnehin pro Zeile; Redaction-Regeln gelten dann automatisch mit).
- **Frontend:** `MemberStammdatenTab.tsx` rendert das Feld read-only (`disabled`/reines Anzeige-Element), außer `user.role === 'admin'`. Mitglieder-Listenseite zeigt bei `member_number_conflict !== ''` ein `AlertTriangle`-Badge mit `brand-danger`/Tooltip.

## Risks / Trade-offs

- **Race zweier paralleler Creates vergibt dieselbe Nummer** → Unique-Index lässt das zweite INSERT fehlschlagen; Vergabe + INSERT in einer Transaktion (`BEGIN IMMEDIATE`) bzw. Retry bei Unique-Verletzung. Bei der geringen Last (kleiner Verein, 1 GB VPS) unkritisch.
- **Konflikt-Berechnung pro Request** → bei ~200 Mitgliedern vernachlässigbar; Dubletten-Erkennung über eine Aggregat-Abfrage statt N+1.
- **Redaction:** Trainer sehen `member_number` ohnehin redacted — das Konflikt-Flag darf für redacted Items keine Nummer leaken; Flag-Typ ist abstrakt (kein Nummernwert), daher unkritisch. Trotzdem in Tests prüfen.
- **Lokale DB ≠ Produktion** → echte Konflikte zeigen sich erst nach Deploy in der Übersicht; das ist gewollt (Sichtbarkeit statt Annahme).

## Migration Plan

Keine DB-Migration. Reiner Code-Change (Backend + Frontend). Deploy via `make deploy`. Rollback = vorheriges Binary; keine Datenänderung, daher gefahrlos reversibel.

## Open Questions

- Keine offenen Punkte — Editierbarkeit (Admin-Override), Backfill (keiner) und Konflikt-Surface (Übersicht) sind geklärt.
