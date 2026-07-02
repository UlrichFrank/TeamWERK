# Design

## Kontext

Ausgangsbefund: AirPlay von iPhone Safari auf AppleTV liefert Ton, aber kein Bild. HLS ist codecseitig kompatibel (H.264 High + AAC-LC), aber die `master.m3u8` signalisiert die Codecs nicht — tvOS bekommt keine Video-Decoder-Info und verwirft den Video-Elementary-Stream.

Wir nutzen die Änderung, um drei weitere latente Probleme zu erledigen:

1. `pix_fmt` nicht explizit → 10-bit-Quellen könnten yuv420p10 produzieren (nicht tvOS-kompatibel).
2. Stream-Token 1 h → Segment-403 bei langen Videos mitten in der Wiedergabe.
3. Kein Chromecast-Pfad → Android-/Chrome-Nutzer haben kein TV-Werfen.

Und wir ziehen den Speicher-Cleanup mit: 360p raus.

## CODECS-Ableitung

`writeMasterManifest` schreibt aktuell:

```
#EXT-X-STREAM-INF:BANDWIDTH=2800000,RESOLUTION=1280x720
720p/index.m3u8
```

Apples HLS-Authoring-Spec und tvOS-AirPlay-Receiver verlangen `CODECS="…"`:

```
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=2800000,RESOLUTION=1280x720,CODECS="avc1.640028,mp4a.40.2"
720p/index.m3u8
```

**Warum wir den Codec-String pro Video ermitteln statt hardcoden:**

Die Video-Seite wäre theoretisch stabil (`libx264 -preset medium -crf 26` → typisch `avc1.640028` = High L4.0). In der Praxis skaliert `-vf scale=-2:720` je nach Quell-Auflösung und Bitrate zu leicht unterschiedlichem Level (3.1 / 4.0 / 4.1). Vor allem aber ist die **Audio-Seite instabil**: bei `-c:a copy` (Quelle bereits AAC) kann die Quelle HE-AAC (`mp4a.40.5`) oder LC (`mp4a.40.2`) sein. Ein hartcodierter String läge dann falsch und wir hätten das gleiche Bild-fehlt-Problem in einer neuen Ecke.

Also: nach jedem erfolgreichen Transcode ein `ffprobe` gegen `seg_001.ts` der 720p-Rendition, Ergebnis in die neue Spalte `videos.codecs` (TEXT NULL), Format: `"avc1.PPCCLL,mp4a.40.X"`. `writeMasterManifest` liest die Spalte und baut den STREAM-INF-String daraus.

Codec-String-Konstruktion:

```
Video (H.264, avc1.PPCCLL):
  PP = Profile-ID hex (Baseline=42, Main=4D, High=64, High10=6E)
  CC = Constraint-Byte hex (aus ffprobe stream=profile ableitbar,
       für unsere Preset-Ausgaben typisch 00 oder 40)
  LL = Level hex (Level × 10; 3.0=1E, 3.1=1F, 4.0=28, 4.1=29)

Audio (mp4a.40.X):
  X = AAC-Object-Type (LC=2, HE-AAC v1=5, HE-AAC v2=29)
```

`ffprobe`-Aufruf:

```bash
ffprobe -v error -select_streams v:0 \
  -show_entries stream=profile,level -of default=noprint_wrappers=1 \
  <seg_001.ts>
```

liefert `profile=High` und `level=40` → String-Bau in Go.

## `pix_fmt yuv420p` erzwingen

In `runFFmpegRendition` einen Arg-Block `"-pix_fmt", "yuv420p"` vor `-c:v` einfügen. Fixt 10-bit-iPhone-Quellen ohne Verhalten für bestehende Quellen zu ändern (libx264-Default bei 8-bit-Quellen ist bereits yuv420p — explizit = sicher).

## Stream-Token-TTL an Video-Dauer

`Sign(vid, uid int)` → `Sign(vid, uid, durationSec int)`. Formel:

```go
ttl := time.Hour
if durationSec > 0 {
    calc := time.Duration(durationSec)*time.Second + 30*time.Minute
    if calc < time.Hour {
        calc = time.Hour
    }
    if calc > 4*time.Hour {
        calc = 4 * time.Hour
    }
    ttl = calc
}
exp := now().Add(ttl).Unix()
```

- **Untergrenze 1 h:** deckt Play-Klick-bis-Segment-Fetch-Toleranz für Kurzclips ab.
- **Obergrenze 4 h:** verhindert unbegrenzt gültige Tokens bei fehlerhaft gesetzten Riesen-Dauern; deckt Halbzeit + Pause + Nachspielzeit + Diskussion mit Trainer.
- **`durationSec = 0`** (Legacy oder Metadaten fehlen) → fällt zurück auf 1 h (heutiges Verhalten).

Aufrufsite: `Play` holt `duration_sec` aus der `videos`-Zeile (das Feld ist bereits vorhanden, siehe `VideoDetail`).

## CORS für Cast-Default-Receiver

Chromecast-Default-Receiver holt die `master.m3u8` und die Rendition-Playlists selbst — inkl. CORS-Preflight-Check bei manchen Firmware-Versionen. Ohne `Access-Control-Allow-Origin`-Header verweigert der Receiver das Playback. Header nur auf `ServeMaster` und `ServeRenditionFile` setzen (nicht global), Wert `*`. Sicherheit bleibt gewahrt: die Auth ist der `?st=`-Token, keine Cookies, kein Credential-Handling.

Für Segmente (`.ts`) ist CORS in der Praxis nicht nötig — Chromecast tut nur Preflight auf Manifeste. Wir setzen den Header dennoch auf beiden Routen einheitlich (keine Sonderfälle).

`OPTIONS`-Preflight: Chi routet OPTIONS nicht automatisch auf die GET-Handler. Wir hängen `chi.Middlewares` mit einem simplen OPTIONS-Handler an, der die CORS-Header setzt und 204 zurückgibt. Alternativ: `cors.Handler` aus `github.com/go-chi/cors` — für zwei Endpunkte overkill, wir schreiben's inline.

## Chromecast-Sender im Frontend

**Loading-Strategie:** SDK **nicht** global geladen (DSGVO-neutral). Erst wenn der User den Cast-Button klickt:

```ts
// web/src/lib/cast.ts
let castReady: Promise<boolean> | null = null;
export function loadCastSDK(): Promise<boolean> {
  if (castReady) return castReady;
  castReady = new Promise((resolve) => {
    (window as any).__onGCastApiAvailable = (available: boolean) => resolve(available);
    const s = document.createElement('script');
    s.src = 'https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1';
    s.crossOrigin = 'anonymous'; // verhindert Cookie-Leak an gstatic
    // KEIN integrity=… — Google versioniert das Script serverseitig ohne stabile
    // Hashes; SRI würde bei Google-Updates die Cast-Integration brechen.
    s.async = true;
    document.head.appendChild(s);
  });
  return castReady;
}
```

**Session-Start:** nach `loadCastSDK()` erfolgreich:

```ts
const context = cast.framework.CastContext.getInstance();
context.setOptions({
  receiverApplicationId: chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
  autoJoinPolicy: chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED,
});
const session = await context.requestSession();
const media = new chrome.cast.media.MediaInfo(masterURL, 'application/vnd.apple.mpegurl');
media.streamType = chrome.cast.media.StreamType.BUFFERED;
await session.loadMedia(new chrome.cast.media.LoadRequest(media));
```

**Wichtig:** die `masterURL` enthält bereits das `?st=`-Token. Chromecast überträgt die URL komplett — Auth trägt sich analog zu AirPlay.

**Button-Rendering:** eine kleine Komponente `<CastButton masterURL={…} />` in `VideoDetailPage`, sichtbar nur wenn `navigator.userAgent` Chrome-basiert und die Cast-API verfügbar (nach `loadCastSDK`). Kein Rendering auf Safari/Firefox — dort greift AirPlay nativ bzw. Cast ist eh nicht verfügbar.

## `x-webkit-airplay="allow"` am `<video>`

Reine Verstärkung des Default-Verhaltens. Safari erlaubt AirPlay per Default, aber setzen wir's explizit, ist die Absicht in Code + Reviews sichtbar und bleibt robust gegen künftige Safari-Default-Verschärfungen.

## Backfill bestehender Videos

Zwei Änderungen betreffen bestehende `ready`-Videos:

1. `master.m3u8` referenziert 360p, hat aber kein CODECS.
2. `videos.codecs`-Spalte ist neu und leer.

**Kein Re-Transcode** — die Segmente auf Disk sind gut. Nur:

- `ffprobe seg_001.ts` (720p bevorzugt, 360p Fallback wenn nur die vorhanden) → Codec-String.
- `UPDATE videos SET codecs=? WHERE id=?`
- `writeMasterManifest(processedDir)` → neue master mit CODECS + INDEPENDENT-SEGMENTS + nur 720p-STREAM-INF.
- Falls `360p/` existiert: Verzeichnis löschen (Speicher-Ersparnis auf Bestand).

**Wo läuft das:** eigener Scheduler-Job `internal/videos/backfill.go` mit Signatur `RunTVCompatBackfill(ctx, db, storageDir)`, aufgerufen aus dem bestehenden `scheduler:run`-Subcommand einmal beim Start (idempotent). Alternative wäre lazy bei jedem `Play`, aber:

- Lazy hätte den Vorteil, dass nur tatsächlich abgerufene Videos bearbeitet werden — dafür Latenz beim ersten Klick nach Deploy und Nebenläufigkeit im Play-Pfad.
- Eager-Backfill läuft einmal, ist gut testbar, und der 1-GB-VPS verkraftet ein paar ffprobes an einer Nicht-Peak-Zeit.

Wir wählen **eager**, mit Idempotenz-Check `WHERE codecs IS NULL AND status='ready'`, damit Neustarts nichts doppelt tun.

**Fehlerpfad:** ein einzelnes Video, bei dem ffprobe scheitert, wird geloggt und übersprungen — es bleibt `codecs=NULL`, `master.m3u8` unverändert, es kann nicht auf AppleTV geworfen werden, spielt aber im Browser weiter. Kein globaler Abbruch.

## Alternativen erwogen und verworfen

- **CODECS-String hardcoden:** einfacher, aber falsch bei `-c:a copy` mit HE-AAC-Quelle.
- **DASH statt/neben HLS:** DASH deckt keinen Client ab, den HLS nicht schon abdeckt. CMAF (fMP4 statt TS) würde Speicher-Doppelung vermeiden, ist aber ein größerer ffmpeg-Umbau ohne Reichweitengewinn.
- **HEVC (H.265):** ~40 % kleiner, aber Chromecast-Kompat wackelig. Widerspricht der Cast-Zielrichtung.
- **Master-Playlist on-the-fly aus DB bauen (kein `master.m3u8`-File):** eleganter, würde Backfill überflüssig machen. Verworfen, weil `master.m3u8`-Zwischenspeicher heute auf Disk liegt und Änderungen an dieser Struktur den Blast-Radius unnötig vergrößern.
- **Cast-SDK self-hosten mit SRI:** wir müssten Google-Kompat-Updates manuell nachziehen; Cast-Firmware-Änderungen brechen dann bei uns statt bei Google. Restrisiko der Non-SRI-Route bewusst getragen — Mitigation über explizite User-Aktion (kein passives Laden).
