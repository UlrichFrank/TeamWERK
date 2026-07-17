## ADDED Requirements

### Requirement: Client verkleinert Bilder vor dem Upload
Das Frontend SHALL vor jedem `POST /api/match-reports/{id}/images` die ausgewählte Datei clientseitig verkleinern: Zielgröße ≤ 1 MB, längste Kante ≤ 1920 px, Ausgabeformat **JPEG** (nur JPEG — WebP ist im Server-MIME-Filter `image/jpeg`+`image/png` nicht enthalten und würde `HTTP 400 unsupported_mime` erzeugen). Bereits kleine Dateien (`file.size ≤ 1 MB`) werden unverändert übernommen. Der Server-seitige 8-MB-Cap in `internal/matchreports/images.go` bleibt unverändert und dient als Backstop.

#### Scenario: Kamera-JPG > 8 MB wird akzeptiert
- **WHEN** die/der Nutzer:in ein 12 MB großes JPG aus der Handy-Galerie auswählt
- **THEN** verkleinert das Frontend die Datei auf ≤ 1 MB JPEG, sendet `POST /api/match-reports/{id}/images` und der Server antwortet mit HTTP 201

#### Scenario: PNG unter Zielgröße bleibt unverändert
- **WHEN** die/der Nutzer:in ein 400 KB großes PNG auswählt
- **THEN** wird die Datei unverändert (ohne Recompression) an den Server gesendet und der Server antwortet mit HTTP 201

#### Scenario: HEIC vom iPhone kann nicht verkleinert werden
- **WHEN** die/der Nutzer:in ein HEIC/HEIF-Foto auswählt, das der Browser nicht als `ImageBitmap` dekodieren kann
- **THEN** überspringt das Frontend die Verkleinerung, sendet die Datei unverändert, der Server antwortet mit HTTP 400 `unsupported_mime`, und das Frontend zeigt „IMG_XXXX.HEIC — Format nicht unterstützt (nur JPG/PNG)"

### Requirement: Multi-Select-Upload mit Gesamt-Cap 10
Das Frontend SHALL im Bild-Upload-Auswahldialog des Spielbericht-Formulars die gleichzeitige Auswahl mehrerer Dateien erlauben (`<input type="file" multiple>`). Die ausgewählten Dateien werden **sequenziell** (nicht parallel) hochgeladen. Übersteigt die Summe aus bereits am Bericht hängenden Bildern (`report.images.length`) plus Anzahl der neu gewählten Dateien den Gesamt-Cap von 10, kürzt das Frontend die Auswahl **vor** dem ersten Upload auf die noch freien Slots und zeigt eine sichtbare Meldung („Nur die ersten N Bilder werden hochgeladen — Limit 10 erreicht"). Der Server-seitige Cap `MaxImages=10` (HTTP 400 `too_many_images`) bleibt der Backstop.

#### Scenario: Auswahl von 3 Bildern in leerem Bericht
- **WHEN** die/der Nutzer:in bei 0 vorhandenen Bildern drei Dateien im Picker auswählt
- **THEN** lädt das Frontend sie sequenziell nacheinander hoch, jeder Upload sendet einen eigenen `POST /images`-Request und das Formular zeigt am Ende drei Bild-Kacheln

#### Scenario: Auswahl übersteigt Cap
- **WHEN** der Bericht 7 Bilder hat und die/der Nutzer:in 5 Dateien im Picker auswählt
- **THEN** kürzt das Frontend die Auswahl auf die ersten 3 Dateien, lädt diese hoch, zeigt die Meldung „Nur die ersten 3 Bilder werden hochgeladen — Limit 10 erreicht" und sendet keine weiteren `POST /images`-Requests

#### Scenario: Vollständig blockiert wenn Cap bereits erreicht
- **WHEN** der Bericht bereits 10 Bilder hat
- **THEN** rendert das Frontend den „Bild wählen"-Button nicht, sodass keine weiteren Uploads gestartet werden können

#### Scenario: Upload-Reihenfolge folgt Auswahl
- **WHEN** die/der Nutzer:in drei Dateien in Reihenfolge A, B, C auswählt
- **THEN** wird A zuerst hochgeladen, dann B, dann C — jeder folgende Upload beginnt erst nach Abschluss (Erfolg oder Fehler) des vorigen

### Requirement: Sichtbare Fehleranzeige bei Bild-Upload
Das Frontend SHALL bei jedem gescheiterten `POST /api/match-reports/{id}/images` eine sichtbare, persistente Fehlermeldung im Bilder-Bereich anzeigen — pro fehlgeschlagene Datei ein Eintrag mit Dateiname und der übersetzten Fehlerursache. Die Anzeige wird beim nächsten Upload-Klick zurückgesetzt. Server-Fehlercodes werden wie folgt in deutsche User-Texte übersetzt:

| Server-`error`          | User-Text                                            |
|-------------------------|------------------------------------------------------|
| `too_many_images`       | Limit von 10 Bildern erreicht                        |
| `unsupported_mime`      | Format nicht unterstützt (nur JPG/PNG)               |
| `image_too_large`       | Datei ist zu groß nach Verkleinerung                 |
| `bad_multipart`         | Datei konnte nicht gelesen werden                    |
| `in_progress` / `already_published` / `not_found` | Bericht ist nicht mehr editierbar |
| _sonstiges / Netzfehler_| Upload fehlgeschlagen — bitte erneut versuchen       |

#### Scenario: Server lehnt HEIC ab
- **WHEN** ein `POST /images` mit HTTP 400 `{"error":"unsupported_mime"}` antwortet
- **THEN** zeigt das Frontend „<dateiname> — Format nicht unterstützt (nur JPG/PNG)" als sichtbaren Alert-Eintrag

#### Scenario: Mehrere Fehler bei Multi-Select
- **WHEN** drei Dateien hochgeladen werden und Dateien 1 und 3 mit `unsupported_mime` scheitern, Datei 2 erfolgreich ist
- **THEN** zeigt das Frontend nach Abschluss zwei Alert-Einträge (für Datei 1 und 3), Datei 2 erscheint als neue Bild-Kachel

#### Scenario: Netzfehler
- **WHEN** ein `POST /images` mit einem Netzwerkfehler abbricht (keine HTTP-Antwort)
- **THEN** zeigt das Frontend „<dateiname> — Upload fehlgeschlagen — bitte erneut versuchen"

#### Scenario: Fehleranzeige wird beim nächsten Klick zurückgesetzt
- **WHEN** nach einer Fehleranzeige die/der Nutzer:in den „Bild wählen"-Button erneut betätigt und eine weitere Datei auswählt
- **THEN** wird die alte Fehlerliste geleert und nur neue Fehler (falls welche entstehen) werden angezeigt

### Requirement: Upload-Fortschritt im Auswahl-Button
Das Frontend SHALL während eines laufenden Multi-Uploads am „Bild wählen"-Button den Fortschritt als `x/y` anzeigen (z. B. „Lade 3/5…") und den Button-Zustand `disabled` setzen, damit während der Sequenz keine parallele Auswahl gestartet werden kann.

#### Scenario: Fortschrittsanzeige bei 5-fach-Auswahl
- **WHEN** die/der Nutzer:in 5 Dateien auswählt und der Loop gerade Datei 3 hochlädt
- **THEN** zeigt der Button den Text „Lade 3/5…" und ist deaktiviert

#### Scenario: Button wieder aktiv nach Abschluss
- **WHEN** alle 5 Uploads abgeschlossen sind (unabhängig von Erfolg/Fehler pro Datei)
- **THEN** zeigt der Button wieder „Bild wählen" und ist bedienbar
