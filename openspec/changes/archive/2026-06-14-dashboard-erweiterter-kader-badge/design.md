## Context

Die `user_accessible_teams`-View vereinigt Stammkader, Erweiterter Kader, Trainer und Eltern ohne `is_extended`-Flag. Alle API-Endpoints, die auf dieser View aufbauen (`/teams/my`, `/dashboard`), können den Unterschied nicht direkt ablesen. Die Prüfung muss daher direkt gegen `kader_extended_members` gehen.

Da kein Spieler gleichzeitig in `kader_members` und `kader_extended_members` für dasselbe Team/Saison-Paar stecken kann (mutual exclusive), ist die Logik eindeutig: ist ein Member-ID für User+Team+Season in `kader_extended_members` und NICHT in `kader_members` → `is_extended = true`.

## Goals / Non-Goals

**Goals:**
- `GET /teams/my` liefert `isExtended` pro Team
- `GET /dashboard` → `meineTermine` liefert `isExtended` pro Event
- Dashboard-UI zeigt Badge „Erw. Kader" an beiden Stellen

**Non-Goals:**
- `MeinTeamPage` (separate Seite) bleibt unverändert — dort sieht man bereits die `extended_players`-Sektion
- Keine Änderung an `user_accessible_teams`-View (zu breit, würde andere Queries tangieren)
- Keine Push-Notifications oder E-Mail-Hinweise

## Decisions

### isExtended-Abfrage via LEFT JOIN statt Subquery

Für `/teams/my`:

```sql
SELECT DISTINCT t.id, t.name,
  CASE WHEN kem.member_id IS NOT NULL THEN 1 ELSE 0 END AS is_extended
FROM user_accessible_teams uat
JOIN teams t ON t.id = uat.team_id
JOIN seasons s ON s.id = uat.season_id
LEFT JOIN members m ON m.user_id = uat.user_id
LEFT JOIN kader_extended_members kem
  ON kem.member_id = m.id
  AND EXISTS (
    SELECT 1 FROM kader k
    WHERE k.id = kem.kader_id AND k.team_id = t.id AND k.season_id = s.id
  )
WHERE uat.user_id = ? AND s.is_active = 1
ORDER BY t.name
```

Problem: `DISTINCT` auf mehrere Felder — wenn ein User mehrere Members hat (Edge-Case), könnte derselbe Team-Eintrag mehrfach erscheinen. Einfachere Alternative: zwei separate Queries (primary + extended-only) und UNION.

**Gewählter Ansatz — UNION:**

```sql
-- Stammkader, Trainer, Eltern: is_extended=0
SELECT t.id, t.name, 0 AS is_extended
FROM user_accessible_teams uat
JOIN teams t ON t.id = uat.team_id
JOIN seasons s ON s.id = uat.season_id
WHERE uat.user_id = ? AND s.is_active = 1
AND EXISTS (
  SELECT 1 FROM kader_members km
  JOIN kader k ON k.id = km.kader_id
  JOIN members m ON m.id = km.member_id
  WHERE m.user_id = ? AND k.team_id = t.id AND k.season_id = s.id
  UNION ALL
  SELECT 1 FROM kader_trainers kt
  JOIN kader k ON k.id = kt.kader_id
  JOIN members m ON m.id = kt.member_id
  WHERE m.user_id = ? AND k.team_id = t.id AND k.season_id = s.id
  UNION ALL
  SELECT 1 FROM family_links fl
  JOIN kader_members km2 ON km2.member_id = fl.member_id
  JOIN kader k ON k.id = km2.kader_id
  WHERE fl.parent_user_id = ? AND k.team_id = t.id AND k.season_id = s.id
)

UNION

-- Nur Erweiterter Kader: is_extended=1 (wenn kein Stammzugang vorhanden)
SELECT t.id, t.name, 1 AS is_extended
FROM kader_extended_members kem
JOIN kader k ON k.id = kem.kader_id
JOIN teams t ON t.id = k.team_id
JOIN seasons s ON s.id = k.season_id
JOIN members m ON m.id = kem.member_id
WHERE m.user_id = ? AND s.is_active = 1
AND NOT EXISTS (
  SELECT 1 FROM kader_members km
  JOIN kader k2 ON k2.id = km.kader_id
  JOIN members m2 ON m2.id = km.member_id
  WHERE m2.user_id = ? AND k2.team_id = t.id AND k2.season_id = s.id
)
```

Alternativ wäre eine CTE möglich, aber UNION ist in SQLite performant genug für kleine Datensätze.

### isExtended für Dashboard-Events via CTE

Im `queryNextEvents`-Query wird eine CTE `extended_teams` vorgeschaltet, die alle `team_id`-Werte liefert, für die der User Extended ist. Das Haupt-UNION-Statement prüft dann per `CASE WHEN`:

```sql
WITH extended_teams AS (
  SELECT k.team_id
  FROM kader_extended_members kem
  JOIN kader k ON k.id = kem.kader_id
  JOIN members m ON m.id = kem.member_id
  JOIN seasons s ON s.id = k.season_id
  WHERE m.user_id = ? AND s.is_active = 1
)
SELECT …, CASE WHEN et.team_id IS NOT NULL THEN 1 ELSE 0 END AS is_extended
FROM … LEFT JOIN extended_teams et ON et.team_id = ts.team_id
```

### Badge-Design im Frontend

Kein neues Komponenten-File — inline Badge als `<span>` mit bestehenden Brand-Token-Klassen:

```tsx
{team.isExtended && (
  <span className="ml-1.5 text-xs font-medium text-brand-text-muted border border-brand-border rounded px-1.5 py-0.5">
    Erw. Kader
  </span>
)}
```

Für Termine: Zusatz direkt im `teamName`-Zeilen-Text, kein visuell schwerer Badge — der Kontext ist informativ, nicht warnend.

## Risks / Trade-offs

- **Eltern-Zugang**: Eltern gelangen via `family_links` in `user_accessible_teams`. Ihre `isExtended`-Berechnung im UNION-Query schließt sie korrekt mit `is_extended=0` ein, da sie über `family_links` + `kader_members` kommen, nicht über `kader_extended_members`. Kein Risiko.
- **Mehrere Members pro User**: Unwahrscheinlich in der Praxis, aber technisch möglich. Der UNION-Ansatz dedupliziert via `UNION` (nicht `UNION ALL`) — ist ein Team einmal mit `is_extended=0` drin, dominiert das.

## Migration Plan

Keine DB-Migration. Reine Query- und Frontend-Änderungen. Kein Rollback-Risiko (additive API-Felder).
