# Design

## Context

Siehe proposal.md (Why). Ausgangslage im Code:

- `messages.media_id` und `broadcasts.media_id` (Migration `028`) sind je **eine** nullable Spalte. Der Tabellen-CHECK `(length(body) > 0 OR media_id IS NOT NULL)` hängt an ihr.
- `messageSelect` (`internal/chat/handler.go`) joint `media` für Breite und Höhe; `ListMessages`, `GetMessage` und `ListBroadcasts` bilden daraus die Felder `mediaId`/`mediaUrl`/`mediaWidth`/`mediaHeight`.
- `media.canSee` (`internal/media/handler.go`) gibt ein Bild über `m.media_id = ?` bzw. `b.media_id = ?` frei.
- `SendMessage` und `CreateBroadcast` prüfen nur `SELECT COUNT(*) FROM media WHERE id = ?`, also weder Eigentümer noch Mehrfachverwendung.
- Konversationen und Mitteilungen werden hart gelöscht (`DELETE FROM conversations`, `DELETE FROM broadcasts`), Nachrichten selbst nur weich (`deleted_at`).
- Frontend: `ChatPage.tsx` hält genau ein `pendingImage` (Chat) bzw. `image` (Mitteilungs-Dialog), lädt es über `compressImage` + `POST /api/media/upload` hoch und rendert ein `AuthImage` je Sprechblase. Die Lightbox kennt nur eine URL.
- Layout-Gotcha (`06-gotchas.md`, Chat-Bild-Platzhalter): die Sprechblase ist shrink-to-fit. Jede Bild-Box braucht schon vor dem Laden ihre endgültige Größe, sonst springt der Verlauf unter iOS Safari.

## Goals / Non-Goals

**Goals:**
- Ein Album aus 1–10 Bildern je Nachricht bzw. Mitteilung, in fester Reihenfolge.
- Ältere, noch gecachte PWA-Clients zeigen weiter wenigstens das erste Bild und können weiter Einzelbilder senden.
- Die Layout-Stabilität der Sprechblase bleibt auch beim Raster erhalten.

**Non-Goals:**
- Andere Anhänge als Bilder (PDF, Video). Die MIME-Allowlist von `media` bleibt unverändert.
- Ein Album nachträglich bearbeiten (Bilder ergänzen, entfernen, umsortieren). `EditMessage`/`EditBroadcast` bleiben reine Textänderungen.
- Bilder in Antwort-Zitaten, in der Chat-Suche oder in der Konversationsliste anzeigen. Dort bleibt es beim heutigen Verhalten.
- Aufräumen verwaister `media`-Zeilen und Dateien (hochgeladen, aber nie gesendet). Das ist heute schon so und bleibt ein eigenes Thema.

## Decisions

### 1. Zuordnungstabellen statt Spaltenliste; `media_id` bleibt als Position 0

Migration `071_chat_album.up.sql`:

```sql
CREATE TABLE message_media (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    media_id   INTEGER NOT NULL REFERENCES media(id),
    position   INTEGER NOT NULL CHECK (position BETWEEN 0 AND 9),
    PRIMARY KEY (message_id, position)
);
CREATE UNIQUE INDEX idx_message_media_media ON message_media(media_id);
-- broadcast_media analog mit broadcast_id → broadcasts(id) ON DELETE CASCADE
INSERT INTO message_media (message_id, media_id, position)
  SELECT id, media_id, 0 FROM messages WHERE media_id IS NOT NULL;
INSERT INTO broadcast_media (broadcast_id, media_id, position)
  SELECT id, media_id, 0 FROM broadcasts WHERE media_id IS NOT NULL;
```

`messages.media_id`/`broadcasts.media_id` werden weiter beim Senden gesetzt, und zwar auf das Bild an Position 0.

- **Warum nicht die Spalte ersetzen:** Der CHECK hängt an `media_id`. Ihn zu ändern hieße, `messages` per Tabellen-Neuaufbau zu migrieren, mit FKs von `message_reads`, `message_reactions`, `chat_polls` und `reply_to_id`. Das wäre riskant und nicht additiv, `make deploy-rollback` wäre danach nicht mehr gefahrlos. Außerdem lesen ältere Clients `mediaId`.
- **Warum Unique auf `media_id` über beide Tabellen getrennt:** Der Index erzwingt „eine media-ID hängt an höchstens einer Nachricht" mechanisch. Dass eine ID nicht zugleich an einer Nachricht und an einer Mitteilung hängt, prüft der Handler. Die Bestandsdaten verletzen den Index nicht: Jedes heutige Bild ist frisch hochgeladen und genau einmal referenziert. Das prüft eine Vorab-Abfrage auf Prod (Migration Plan).
- **Alternative JSON-Array-Spalte:** verworfen. `canSee` bräuchte `json_each` in einem Hot-Path, und die Reihenfolge wäre nicht per Index absicherbar.

**Invariante:** `messages.media_id = (SELECT media_id FROM message_media WHERE message_id = m.id AND position = 0)` bzw. NULL. Ein Test prüft das nach `SendMessage`.

### 2. Ein gemeinsamer Validierungs- und Schreibpfad

`resolveAlbum(ctx, tx, userID, body) ([]int, error)` im chat-Package:
1. Prüft, dass `mediaId` und `mediaIds` nicht beide gesetzt sind; normalisiert die Kurzform zu `[]int{mediaId}`.
2. Lehnt mehr als 10 Einträge und Duplikate ab.
3. Prüft mit **einer** Abfrage, dass alle IDs existieren, `uploaded_by = userID` tragen und weder in `message_media` noch in `broadcast_media` stehen. Ist die Trefferzahl ≠ `len(ids)`, folgt 400.

Gibt es Text oder ein Bild, schreiben `SendMessage` und `CreateBroadcast` in **einer** Transaktion: Insert der Nachricht bzw. Mitteilung mit `media_id = ids[0]`, danach die Zuordnungszeilen. Ein Unique-Verstoß durch ein Rennen zweier Requests mit derselben ID endet als 400 bzw. Rollback, nicht als halbe Nachricht.

Der Fehlercode ist einheitlich `invalid_media` (über `httpx`, Hard Rule für neue Pfade). So verrät der Server nicht, ob eine fremde ID existiert, fremd ist oder schon benutzt wurde. Die übrigen Fehler dieser Handler bleiben in diesem Change beim bisherigen `http.Error`. Die Umstellung auf `httpx` erzwingt der Ratchet der Betriebshärtung, nicht dieser Change.

**Warum die Eigentümerprüfung auch für `mediaId` (Kurzform):** Sie schließt die in der Proposal beschriebene Lücke. Ein Client, der regulär hochlädt und sendet, ist davon nie betroffen.

### 3. Lesen: Batch-Nachladen statt JOIN-Vervielfachung

`messageSelect` bleibt bei einer Zeile je Nachricht und liefert `media_id`/`width`/`height` weiter für die Altfelder. Danach lädt `loadMessageMedia(ctx, ids []int) map[int][]mediaItem` mit **einer** Abfrage

```sql
SELECT mm.message_id, mm.media_id, med.width, med.height
FROM message_media mm JOIN media med ON med.id = mm.media_id
WHERE mm.message_id IN (…) ORDER BY mm.message_id, mm.position
```

alle Albumbilder der Seite (höchstens 100 Nachrichten → höchstens 1000 Zeilen). Ein JOIN in `messageSelect` würde die Zeilen vervielfachen und die Paginierung mit `LIMIT 100` brechen. `ListBroadcasts` bekommt dasselbe über `broadcast_media`. Für gelöschte Nachrichten (`deleted_at IS NOT NULL`) bleibt `media` leer. Der Body wird dort genauso schon geleert.

### 4. `canSee` über die Zuordnungstabellen

Die zwei Bedingungen auf `m.media_id`/`b.media_id` werden durch `message_media` bzw. `broadcast_media` ersetzt:

```sql
OR EXISTS(SELECT 1 FROM message_media mm
          JOIN messages m ON m.id = mm.message_id
          JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
          WHERE mm.media_id = ? AND cm.user_id = ?)
```

Das ist vollständig, weil die Migration jedes Bestandsbild als Position 0 übernimmt und jeder neue Sendepfad die Tabelle befüllt. Der Unique-Index auf `media_id` dient zugleich als Lookup-Index. Die Spalten werden hier bewusst nicht mehr abgefragt: zwei Wahrheiten für dieselbe Frage wären genau die Drift, die der Unique-Index verhindern soll.

### 5. Push und SSE unverändert in der Anzahl

Eine Nachricht ist ein `INSERT`, also gibt es ein `chat:new-message` und eine Push je Empfänger. Nur die Vorschau ändert sich: `imagePreview(n)` liefert „Bild" bzw. „N Bilder" und greift wie bisher nur bei leerem Text.

### 6. Frontend: Upload nacheinander, Raster mit fester Geometrie

- **Auswahl:** `<input type="file" accept="image/*" multiple>`; Einfügen aus der Zwischenablage nimmt **alle** Bilddateien. `pendingImages: {file, previewUrl}[]` ersetzt `pendingImage`. Bei mehr als 10 werden die ersten 10 übernommen, der Rest verworfen, mit Toast „Höchstens 10 Bilder je Nachricht".
- **Upload:** nacheinander (`for … await`), nicht per `Promise.all`. `compressImage` dekodiert jedes Bild in ein Canvas, zehn parallele Decodes von Handy-Fotos sind auf älteren iPhones ein Speicherrisiko. Der Senden-Button zeigt „Bild k/N". Scheitert ein Upload, bricht der Versand ab; die Auswahl bleibt erhalten und lässt sich erneut senden. Schon hochgeladene IDs verfallen als verwaiste `media`-Zeilen (Non-Goal oben).
- **Raster (`ChatImageGrid`, neue Komponente):** 1 Bild wird wie heute dargestellt (`AuthImage` mit Server-Dimensionen). Ab 2 Bildern entsteht ein Raster mit fester Kachelgeometrie und `object-cover`: 2 Bilder → 2 Spalten; 3 → ein großes Bild über zwei kleinen; ab 4 → 2×2 mit „+N" auf der vierten Kachel. Die Rasterbreite ist fest (`w-64 max-w-full`), jede Kachel quadratisch (`aspect-square`). Die Höhe hängt also nur von der Bildanzahl ab, nie vom Ladezustand. Das erfüllt die Regel aus dem Chat-Bild-Gotcha ohne Server-Dimensionen, und genau deshalb heißt die Wahl `object-cover` statt natürlicher Seitenverhältnisse.
- **Lightbox:** Der Zustand wird zu `{ urls: string[], index: number }`. Pfeil-Buttons (`ChevronLeft`/`ChevronRight` mit `aria-label`), die Tasten ←/→ und ein Zähler „k / N". Ein Klick auf eine Kachel öffnet die Lightbox an diesem Index.
- **Mitteilungs-Dialog und -Detail** verwenden denselben Picker-Zustand und dasselbe `ChatImageGrid`.

**Alternative: natürliches Seitenverhältnis je Kachel (Masonry).** Verworfen, weil die Höhe dann an den Server-Dimensionen jedes einzelnen Bildes hinge. Bilder ohne Dimensionen würden wieder nachträglich springen, und die Rasterlogik wäre deutlich komplexer.

## Risks / Trade-offs

- [Unique-Index scheitert in der Migration an doppelt referenzierten Bestandsbildern] → Vor dem Deploy auf Prod `SELECT media_id, COUNT(*) FROM messages WHERE media_id IS NOT NULL GROUP BY media_id HAVING COUNT(*) > 1` (analog für `broadcasts` und über beide Tabellen hinweg) ausführen. Bei Treffern Migration anpassen (nur die erste Referenz übernehmen), bevor deployt wird.
- [Ältere PWA-Version sieht von einem Album nur das erste Bild] → bewusst akzeptiert. Der Service Worker aktualisiert sich beim nächsten Start, das Problem besteht also nur bis dahin.
- [Verwaiste `media`-Zeilen nach abgebrochenem Mehrfach-Upload] → je Fall höchstens 9 Dateien à ≤ 1 MB; Aufräumen ist als Non-Goal ausgeklammert.
- [`object-cover` schneidet Bildränder im Raster ab] → In der Lightbox erscheint immer das ganze Bild. Bei genau einem Bild gibt es keinen Beschnitt.
- [Die strengere Eigentümerprüfung bricht einen unbekannten Client, der fremde media-IDs wiederverwendet] → Im Repo gibt es keinen solchen Aufrufer (Frontend lädt immer frisch hoch), geprüft per `grep "/media/upload"`.

## Migration Plan

1. Prod-Vorabprüfung auf doppelt referenzierte `media_id` (siehe Risks).
2. Deploy über `make deploy`; Migration `071` läuft automatisch, ist additiv und braucht keinen manuellen Schritt.
3. Rollback: `make deploy-rollback`. Das alte Binary ignoriert die neuen Tabellen und liest `media_id` = erstes Bild. Albumbilder ab Position 1 sind dann unsichtbar, gehen aber nicht verloren. Kein `migrate down`.
