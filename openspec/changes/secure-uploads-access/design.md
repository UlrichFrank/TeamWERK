## Context

`/api/uploads/*` liefert vor allem Mitglieder-/Nutzerfotos, die im Frontend über `<img src="/api/uploads/...">` eingebettet werden. `<img>`-Requests senden **keinen** `Authorization`-Header — ein simples Verschieben hinter die `auth.Middleware` (Bearer) würde die Bilddarstellung brechen. Das Projekt hat dieses Problem für SEPA-Mandate bereits gelöst: ein kurzlebiges, signiertes Download-Token (`SepaDownloadToken` → `SepaDownload`, Capability `file-download-token`), das als Query-Parameter an die URL gehängt wird.

## Goals / Non-Goals

**Goals:**
- Kein tokenloser, unauthentifizierter Abruf von Fotos mehr.
- Bildanzeige im Frontend bleibt funktionsfähig (kein Bearer im `<img>`).
- Berechtigungsprüfung (wer darf das Foto sehen) beim Token-Ausstellen.

**Non-Goals:**
- Keine Verschlüsselung der Fotos at rest (anderes Schutzniveau als Bankdaten; nicht Teil dieses Befunds).
- Kein Umbau des SEPA-Pfads (bleibt wie ist; dient als Vorlage).

## Decisions

**D1 — Download-Token statt Bearer.** Wiederverwendung des `file-download-token`-Musters: Ein authentifizierter `POST/GET`-Endpoint stellt nach Berechtigungsprüfung ein kurzlebiges, signiertes Token (HMAC) für eine konkrete Datei aus; `ServeUpload` validiert das Token, bevor es streamt. Alternative „Bearer-geschützter XHR + Blob-URL im Frontend" verworfen: mehr Frontend-Komplexität, kein Caching-Vorteil, weicht vom etablierten SEPA-Muster ab.

**D2 — Berechtigung beim Ausstellen, nicht beim Streamen.** Die teure/fachliche Sichtbarkeitsprüfung (`policy.MemberCan`-analog) passiert bei der Token-Ausgabe; `ServeUpload` prüft nur noch Token-Signatur + Ablauf + Datei-Bindung. Hält den Streaming-Pfad billig und die Autorisierung an einem Ort.

**D3 — Defense-in-Depth bleibt.** UUIDv4-Dateinamen, `..`-Abwehr und `filepath.Join`-Re-Rooting bleiben erhalten; zusätzlich `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store`, damit Token-URLs nicht über Referrer/History/Proxy-Cache weiterleaken.

**D4 — Doc-Kommentar korrigieren.** Der irreführende `// Auth required`-Kommentar wird an die reale (jetzt tatsächlich geschützte) Mountierung angeglichen, damit künftige Maintainer nicht fälschlich Schutz annehmen.

## Risks / Trade-offs

- **[Frontend-Bildstellen vergessen → kaputte Bilder]** → alle `photoURL`-Verwender in `web/src/` erfassen und auf Token-URL umstellen; visuelle Prüfung der betroffenen Seiten.
- **[Token-URL leakt trotzdem]** → kurze Gültigkeit + `no-referrer`/`no-store`; Token bindet an konkrete Datei, nicht an ein Pauschalrecht.
- **[Cache-Verlust bei Fotos]** → `private, no-store` verhindert Shared-Cache; akzeptabel, Fotos sind klein und selten.

## Migration Plan

Ein-Schritt-Deploy von Backend + Frontend zusammen (Breaking für tokenlose Direktzugriffe). Rollback durch Zurücknehmen beider Commits. Vor Deploy prüfen, dass keine externen Konsumenten direkt auf `/api/uploads/...` verlinken.

## Open Questions

- Token-Ausgabe als eigener Endpoint pro Foto oder gebündelt (z.B. Token bereits in der `photoURL` der Mitglieds-API mitliefern)? Empfehlung: Token in der jeweiligen API-Antwort mitgeben, die das Foto referenziert — spart einen Roundtrip und bündelt die Sichtbarkeitsprüfung mit der ohnehin erfolgenden Mitglieds-Autorisierung.
