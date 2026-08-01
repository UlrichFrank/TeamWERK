# Ordner-Eigentümerrecht und Team-Principals

## Why

Ein Trainer legt unter `/dokumente` einen Ordner an und vergibt darin Lese- und Schreibrechte an
eine andere Person. Ab diesem Moment hat er auf seinen eigenen Ordner **selbst keine Rechte mehr** —
er kann nicht mehr schreiben und sieht den Ordner nicht einmal mehr in der Liste.

Ursache ist die Nearest-Ancestor-Wins-Auflösung (`internal/files/handler.go:91`, `resolveAccess`):
Der Pfad-Walk stoppt beim ersten Ordner, der **überhaupt irgendeine** Zeile in `folder_permissions`
besitzt — unabhängig davon, ob eine davon auf den Aufrufer passt. Solange ein frisch angelegter
Ordner leer ist, erbt der Ersteller korrekt vom Elternordner. Die allererste ACL-Zeile kappt diese
Vererbung und lässt ihn selbst ohne Treffer zurück. `file_folders.created_by` existiert, wird bei
der Rechteauflösung aber nirgends gelesen — es gibt kein Eigentümer-Konzept.

Die Wirkung ist eine **Selbst-Aussperrung ohne Rückweg**: `ListPermissions` und `DeletePermission`
verlangen beide `can_write`, der Betroffene kann den Berechtigungsdialog also nicht mehr öffnen, um
seinen eigenen Fehler zu korrigieren. Nur ein Admin kommt noch heran.

Nearest-Ancestor-Wins bleibt trotzdem richtig: die additive Vorgänger-Auflösung war ein
Sicherheitsloch (`archive/2026-06-13-folder-permissions-fix` — eine `everyone: read`-Freigabe weiter
oben hebelte jede Einschränkung im Unterordner aus). Dieser Change ergänzt daher eine dem Walk
**vorgelagerte** Eigentümerregel, statt das Auflösungsmodell umzubauen.

Zweiter, unabhängiger Bedarf: Berechtigungen lassen sich heute nur an `everyone`, `role`,
`club_function` und einzelne Personen vergeben. Für den Alltagsfall „dieser Ordner ist für mA1"
muss ein Trainer jede Person einzeln eintragen und bei jedem Kaderwechsel nachpflegen.

## What Changes

- **Eigentümer-Vorrang (absolut, unterbaumweit).** `created_by` eines Ordners **oder eines
  beliebigen Vorfahren** gewährt dem Nutzer immer `can_read` **und** `can_write` — geprüft *vor*
  dem Pfad-Walk und unabhängig von jeder ACL. Wer einen Ordner angelegt hat, kann sich weder selbst
  noch durch andere daraus aussperren, und behält den Zugriff auch in Unterordnern, die andere
  darin anlegen.
- **Heilt Bestandsdaten ohne Datenmigration.** `file_folders.created_by` ist seit Migration `001`
  `NOT NULL` und für jeden existierenden Ordner gefüllt. Bereits kaputte Eigentümerfälle
  funktionieren unmittelbar nach dem Deploy wieder — kein Backfill, kein manuelles Reparieren.
- **Zwei neue Principal-Typen: `team` und `team_parents`.** `principal_ref` ist eine `teams.id`.
  `team` matcht die Mitglieder des Kaders dieser Mannschaft in der **aktiven Saison** — Spieler
  (`kader_members`), erweiterten Kader (`kader_extended_members`) und Trainer (`kader_trainers`).
  `team_parents` matcht die über `family_links` verknüpften Elternteile derselben Kader-Menge.
  Beides wird **bei jedem Zugriff** aufgelöst, nichts wird eingefroren: Kaderwechsel und
  Saisonwechsel wirken sofort, ohne dass jemand Berechtigungen nachpflegt.
- **Modal-Struktur bleibt unverändert.** Das Typ-Dropdown in `PermissionsModal` bekommt zwei
  Einträge („Team", „Eltern"), beide blenden dasselbe zweite Dropdown mit den Kaderteams der
  aktiven Saison ein — exakt das Muster, das `role`, `club_function` und `user` heute schon nutzen.
  Quelle ist das bestehende `GET /api/teams/names`, beschriftet mit `buildTeamShortNames`
  („mA1", „mA2", „wB").
- **Eigentümer wird sichtbar.** `GET /api/folders/{id}/permissions` liefert einen nicht löschbaren
  Pseudo-Eintrag `principal_type: "owner"` mit dem Namen des Erstellers. Ohne ihn wäre das stärkste
  Recht am Ordner das einzige, das in keiner Oberfläche auftaucht.
- **Doppelte Auflösungslogik wird zusammengeführt.** `files.resolveAccess` und
  `policy.FolderAccess` sind heute ~65 Zeilen wortgleiche Logik; nur `FolderContents` nutzt die
  Policy-Variante, die übrigen 14 Aufrufstellen die lokale. Jede Regeländerung müsste in beide —
  sonst verhält sich `GET /folders/{id}/contents` anders als der Rest der API.
- **Widersprüchliche Alt-Spec wird entfernt.** `file-permissions` trägt weiterhin das Requirement
  „Additive Berechtigungsvererbung / Kein DENY möglich", das der Juni-Fix fachlich abgelöst, aber
  nie gestrichen hat. Zwei sich widersprechende Specs zur selben Frage sind eine Falle für den
  nächsten Bearbeiter.

**Explizite Nicht-Ziele.** Das Auflösungsmodell bleibt Nearest-Ancestor-Wins; es kommt **kein**
explizites Vererbungs-Flag. Nutzer, die ihren Zugriff über `club_function` geerbt hatten und ihn
durch eine fremde ACL-Zeile verloren haben, werden **nicht** automatisch geheilt — nur Eigentümer.
Solche Fälle bleiben manuelle ACL-Pflege. Es gibt keine Eigentümer-Übertragung in der UI.

## Capabilities

### New Capabilities

Keine.

### Modified Capabilities

- `folder-permission-resolution`: neues, dem Pfad-Walk vorgelagertes Requirement für den
  Eigentümer-Vorrang; neues Requirement für die Auflösung der Principal-Typen `team` und
  `team_parents` gegen die aktive Saison. Die bestehenden Nearest-Ancestor- und
  Family-Context-Requirements bleiben unverändert gültig.
- `folder-permission-ux`: Team-Auswahl im Berechtigungsdialog und Darstellung der neuen Typen in
  der Berechtigungsliste; Eigentümer als nicht löschbarer Pseudo-Eintrag.
- `file-permissions`: Principal-Typen von vier auf sechs erweitert; das überholte Requirement
  „Additive Berechtigungsvererbung" wird entfernt.

## Impact

**Datenbank**
- Neue Migration `038_folder_permissions_team_principals` — Table-Rebuild von
  `folder_permissions`, weil SQLite den `CHECK (principal_type IN (…))` nicht per `ALTER TABLE`
  erweitern kann. Kleine Tabelle, keine eingehenden Fremdschlüssel. Der `down`-Pfad muss
  `team`/`team_parents`-Zeilen löschen, bevor er den alten CHECK wiederherstellt (dokumentierter
  Datenverlust in Rückrichtung).

**Backend**
- `internal/policy/folders.go`: Eigentümer-Vorrang, Team-Auflösung, lazy Principal-Kontext.
- `internal/files/handler.go`: `resolveAccess` entfällt, alle 14 Aufrufstellen gehen auf
  `policy.FolderAccess`; `AddPermission` validiert die neuen Typen; `ListPermissions` liefert
  `display_name` für Team-Einträge und den Owner-Pseudo-Eintrag.
- Keine neue Route, keine Router-Änderung. `Files.*` steht bereits vollständig in der
  `broadcastAllowlist` (`internal/arch/broadcast_test.go:61-69`) — kein Broadcast-Bedarf.

**Frontend**
- `web/src/pages/DocumentsPage.tsx`: zwei Einträge in `PRINCIPAL_TYPE_LABELS`, ein gemeinsamer
  bedingter Select-Block für beide Team-Typen, Lazy-Loader nach dem Vorbild von
  `loadPickerUsers`, Anzeige in `permLabel`, Owner-Zeile ohne Löschen-Button.

**Risiko**
- Der Eigentümer-Vorrang ist ein **absolutes, unbefristetes** Recht: Es überlebt den Verlust der
  Vereinsfunktion und ist nur durch Ändern von `file_folders.created_by` entziehbar. Bewusst so
  entschieden; der Pseudo-Eintrag macht es wenigstens sichtbar.
