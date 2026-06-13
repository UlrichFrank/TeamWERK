## Context

`resolveAccess()` in `internal/files/handler.go` ermittelt Lese-/Schreibrechte für einen Ordner durch einen einzigen SQL-Query gegen alle Vorfahren-Ordner (`WHERE folder_id IN (...)`). Alle Treffer werden additiv vereint (OR). Dadurch kann eine restriktive Regel in einem Unterordner durch eine weiträumigere Regel im Elternordner ausgehebelt werden.

Das aktuelle Modell bietet außerdem keine Möglichkeit, Elternteil-Nutzern automatisch die Zugriffsrechte ihrer verknüpften Kinder zu gewähren, obwohl `family_links` diese Beziehung bereits modelliert.

## Goals / Non-Goals

**Goals:**
- Nearest-Ancestor-Wins: Der nächste Ordner mit eigenen Berechtigungen bestimmt den Zugang exklusiv.
- Family-Context: Elternteil-Nutzer erhalten über `family_links` die Vereinsfunktionen und User-IDs ihrer Kinder in der Berechtigungsprüfung berücksichtigt.
- Keine API-Änderungen, keine Datenbankmigrationen.

**Non-Goals:**
- Explizite Deny-Regeln (kein `can_deny`-Flag)
- Vererbungsschalter pro Ordner (`inherit_permissions`-Flag)
- Änderungen am Frontend

## Decisions

### Entscheidung 1: Nearest-Ancestor-Wins statt Explicit-Deny

**Ansatz:** `resolveAccess()` iteriert `folderPath()` von Ziel → Root und führt für jeden Ordner einen separaten Query aus. Beim ersten Ordner mit mindestens einer Berechtigungszeile wird der Zugang ausschließlich anhand dieser Zeilen entschieden; die Iteration bricht ab.

Ordner ohne eigene Berechtigungen erben transparent vom nächsten Vorfahren mit Regeln. Kein Ordner ohne Regeln in der gesamten Kette → kein Zugriff (false, false).

**Verworfen: Explicit-Deny-Bits** — Zu komplex für Endnutzer; die Unterscheidung „kein Zugriff weil kein Eintrag" vs. „kein Zugriff weil Deny" ist in der UI schwer kommunizierbar.

**Verworfen: inherit_permissions-Flag** — Erfordert UI-Änderungen und ein neues Konzept, das erklärt werden muss. Nearest-Ancestor-Wins liefert dasselbe Ergebnis ohne zusätzliches Datenbankfeld.

**Performance:** Typische Ordnerhierarchie ist 2–3 Ebenen tief. Im Worst Case `n` sequenzielle Queries (n = Tiefe), was bei SQLite im einstelligen µs-Bereich liegt. Kein Caching notwendig.

### Entscheidung 2: Family-Context als einmaliger Pre-Fetch

**Ansatz:** Zu Beginn von `resolveAccess()` wird — sofern der Nutzer nicht Admin ist — ein einziger Query ausgeführt:

```sql
SELECT COALESCE(m.user_id, 0), COALESCE(mcf.function, '')
  FROM family_links fl
  JOIN members m ON m.id = fl.member_id
  LEFT JOIN member_club_functions mcf ON mcf.member_id = m.id
 WHERE fl.parent_user_id = ?
```

Das Ergebnis liefert zwei Sets: `linkedUserIDs []int` und `linkedFunctions []string`. Diese werden im Matching-Loop neben den eigenen Claims verwendet:

- `principal_type=club_function`: matcht wenn `claims.HasFunction(ref)` ODER `ref ∈ linkedFunctions`
- `principal_type=user`: matcht wenn `claims.UserID == ref` ODER `ref ∈ linkedUserIDs`

Nutzer ohne `family_links`-Einträge erhalten leere Sets — kein Overhead im Matching.

**Verworfen: Claims-Erweiterung im JWT** — Würde bedeuten, Kind-Funktionen beim Login in den Token zu kodieren. Bei Änderungen der `family_links` wäre der Token bis zu 15 Minuten veraltet. DB-Query pro Request ist sauberer und immer aktuell.

## Risks / Trade-offs

**[Semantikänderung für bestehende Ordner]** → Ordner, die bisher additiv von Eltern geerbt haben, können nach dem Deploy den Zugriff verlieren, wenn sie eigene (aber unvollständige) Berechtigungen haben. Vor dem Deploy sollte überprüft werden, ob solche Ordner existieren. SQL-Diagnose: `SELECT folder_id, COUNT(*) FROM folder_permissions GROUP BY folder_id` — Ordner mit Einträgen, deren Eltern breitere Rechte haben, müssen ggf. ergänzt werden.

**[N Queries pro resolveAccess-Aufruf]** → Im absoluten Worst Case (Ordner 10 Ebenen tief, alle ohne eigene Regeln) entstehen 10 Queries. Bei SQLite WAL auf lokalem Disk vernachlässigbar. Kein Risiko für den VPS.

## Migration Plan

1. Deploy des neuen Binaries (inkl. `resolveAccess`-Änderung)
2. Kein SQL-Migrate notwendig
3. Rollback: altes Binary einspielen (Deploy ist idempotent)

Empfohlen vor Deploy: einmalige manuelle Prüfung ob Unterordner mit eigenen (restriktiven) Berechtigungen existieren, die bisher unbewusst auf elterliche `everyone`-Regeln vertrauten.
