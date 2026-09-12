## ADDED Requirements

### Requirement: Bevorzugung nativer HLS-Wiedergabe für AirPlay-Kompatibilität

`VideoDetailPage` MUST beim Initialisieren der Wiedergabe zuerst `video.canPlayType('application/vnd.apple.mpegurl')` prüfen. Liefert dies eine nicht-leere Zeichenkette (native HLS-Unterstützung, insbesondere Safari/WebKit auf macOS und iOS), MUST die Komponente `video.src` direkt auf die Master-URL setzen und darf `hls.js` NICHT initialisieren. Nur wenn `canPlayType` eine leere Zeichenkette liefert, MUST die Komponente auf `Hls.isSupported()` und die hls.js/MediaSource-basierte Wiedergabe zurückfallen. Diese Reihenfolge ist notwendig, weil `Hls.isSupported()` auf Safari/WebKit ebenfalls `true` liefert (MediaSource Extensions sind vorhanden), eine über MediaSource gepufferte Wiedergabe aber nicht als vollständige Session an ein AirPlay-Ziel übergeben werden kann — dort routet in der Praxis nur der Audio-Pfad.

#### Scenario: Safari/WebKit nutzt native Wiedergabe
- **WHEN** `VideoDetailPage` in einem Browser mit nativer HLS-Unterstützung gerendert wird (`canPlayType` liefert `"probably"` oder `"maybe"`)
- **THEN** wird `video.src` direkt auf die Master-URL gesetzt und `hls.js` wird nicht instanziert

#### Scenario: AirPlay überträgt Bild und Ton
- **WHEN** ein Nutzer auf Safari/iOS mit aktiver nativer Wiedergabe ein Video per AirPlay an ein Apple TV überträgt
- **THEN** werden sowohl Bild als auch Ton auf dem Apple TV wiedergegeben (Rauchtest, siehe bestehendes Szenario „AirPlay auf AppleTV" unter „HLS-Master-Manifest mit CODECS-Signalisierung" — dieses Szenario gilt nach diesem Fix erstmals als tatsächlich erfüllt)

#### Scenario: Browser ohne native HLS-Unterstützung nutzen weiterhin hls.js
- **WHEN** `VideoDetailPage` in einem Browser ohne native HLS-Unterstützung gerendert wird (`canPlayType` liefert `""`)
- **THEN** wird `hls.js` wie bisher initialisiert und die Wiedergabe läuft MediaSource-basiert
