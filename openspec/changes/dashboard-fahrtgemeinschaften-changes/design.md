## Context

Das Dashboard zeigt pro User einen personalisierten Überblick. Die Sektion „Fahrtgemeinschaften" wird durch `queryCarpoolingHint` im `internal/dashboard/handler.go` befüllt und gibt bisher nur das nächste Auswärtsspiel mit aggregierten Zählern zurück.

Paarungsanfragen, Zusagen und Ablehnungen sind in `mitfahrt_paarungen` mit `status` und `updated_at` persistent. Gelöschte Einträge in `mitfahrgelegenheiten` lösen jedoch `ON DELETE CASCADE` auf `mitfahrt_paarungen` aus — Paarungen zu einem gelöschten Eintrag verschwinden spurlos.

Der Push-Notification-Change (push-notifications-mitfahrgelegenheiten) ist vollständig implementiert. Die Delete-Handler senden bereits vor dem DELETE einen Push an betroffene User.

## Goals / Non-Goals

**Goals:**
- Dashboard zeigt personalisierten Status: eigener Eintrag (biete/suche), aktive Paarungen, Ereignisse der letzten 48 h
- Lösch-Ereignisse werden vor dem CASCADE-DELETE in `carpooling_events` gespeichert
- Kein weiterer Endpunkt nötig — Daten kommen über die bestehende `GET /api/dashboard`-Route

**Non-Goals:**
- Kein vollständiger Audit-Log aller Carpooling-Aktionen
- Kein persistentes „gelesen"-Marking für Events
- Keine Push-Änderungen (die laufen bereits korrekt)
- Kein Anzeigen von Events für vergangene Spiele

## Decisions

### 1. Event-Log nur für Löschungen

**Entscheidung:** `carpooling_events` speichert ausschließlich `biete_deleted` und `suche_deleted`.

**Warum:** Alle anderen Ereignisse (Anfrage, Zusage, Ablehnung, Stornierung) sind direkt aus dem Live-Stand von `mitfahrt_paarungen` ableitbar — mit `status` und `updated_at` (48-h-Fenster). Nur Löschungen hinterlassen keine DB-Spur. Ein vollständiger Event-Log wäre Overengineering.

**Alternative:** Alles in den Log schreiben (einheitlichere Abfrage im Dashboard). Abgelehnt: doppelte Datenhaltung, höherer Schreibaufwand, kein echter Mehrwert für die anderen Typen.

### 2. 48-h-Fenster für Paarungsereignisse aus Live-Daten

**Entscheidung:** `rejected`-Paarungen und frisch `confirmed`-Paarungen werden nur angezeigt wenn `updated_at >= now - 48h`.

**Warum:** Ohne Zeitfenster würden alle abgelehnten Paarungen der Saison im Widget auftauchen. 48 h ist lang genug um eine verpasste Push-Notification zu kompensieren, kurz genug um das Widget übersichtlich zu halten.

### 3. Event-Cleanup implizit via Spielfilter

**Entscheidung:** Kein Cron-Job. Events mit `game_id` eines vergangenen Spiels werden schlicht nicht mehr im Dashboard-Query zurückgegeben (Filter `DATE(g.date) >= DATE('now')`).

**Warum:** Einfachste Lösung ohne zusätzliche Infrastruktur. Die DB wächst minimal (wenige Events pro User pro Saison).

### 4. Write-Point im Delete-Handler vor dem DELETE

**Entscheidung:** `carpooling_events`-Einträge werden *vor* dem `DELETE FROM mitfahrgelegenheiten` geschrieben, in derselben Transaktion.

**Warum:** Nach dem DELETE sind die Paarungsdaten weg (CASCADE). Transaktion sichert Konsistenz: entweder beide Writes erfolgreich oder keiner.

**Betroffene User bei biete_deleted:** alle User mit `pending` oder `confirmed` Paarung gegen diesen biete-Eintrag.  
**Betroffene User bei suche_deleted:** der biete-User, falls eine `pending` oder `confirmed` Paarung existiert.

### 5. Dashboard-Antwortstruktur erweitern (kein neuer Endpunkt)

**Entscheidung:** `CarpoolingHint`-Struct um `MyEntry`, `Paarungen` und `RecentEvents` erweitern. Kein separater API-Endpunkt.

**Warum:** Das Dashboard lädt alles in einem einzigen `GET /api/dashboard`-Request. Ein separater Endpunkt würde einen zweiten Netzwerk-Request erzwingen und die Ladelogik im Frontend aufteilen.

## Risks / Trade-offs

- **Transaktion im Delete-Handler:** Der bestehende Delete-Handler verwendet kein explizites `BEGIN`/`COMMIT`. Er muss auf eine Transaktion umgestellt werden. → Standardmuster in Go (`db.BeginTx`), gut handhabbar.
- **48-h-Fenster ist willkürlich:** Wer länger nicht schaut, verpasst abgelehnte Paarungsinfos. → Akzeptiert; Push-Notification kompensiert den Real-Time-Fall.
- **Kein „ungelesen"-Indikator:** Events sehen immer gleich aus, egal ob gerade eingetreten oder 47 h alt. → Bewusst einfach gehalten; Zeitstempel im Widget schafft ausreichend Kontext.

## Migration Plan

1. Migration `018_carpooling_events.up.sql` + `.down.sql` anlegen
2. Backend deployen (Binary enthält Migration) → `make deploy` führt `migrate up` automatisch aus
3. Kein Rollback-Risiko: nur neue Tabelle und erweitertes Dashboard-JSON, bestehende Features unberührt
