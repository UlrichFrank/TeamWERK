## ADDED Requirements

### Requirement: Poll des Staffel-Spielplans in abgeleiteten Zeitfenstern

Das System SHALL je Staffel, die einem Kader der aktiven Saison zugeordnet ist, den
Spielplan über die öffentliche Handball4All-JSON-Schnittstelle abrufen
(`cmd=ps`, mit `og`, optional `o`, `p`, `cl`, `ca=1`) und Spielplan, Ergebnisse und
Tabellenstand speichern.

Das System SHALL eine Staffel genau dann abfragen, wenn sie am laufenden Tag mindestens
eine Begegnung hat, der früheste Anwurf dieses Tages mindestens zwei Stunden zurückliegt
und mindestens eine Begegnung des Tages noch keinen Spielbericht-Verweis (`sGID`) trägt.
Die Abfragekadenz SHALL 15 Minuten betragen.

Das System SHALL die Poll-Fähigkeit ausschließlich aus den gespeicherten Begegnungen
ableiten und KEIN eigenes Zustandsfeld dafür führen.

Das System SHALL zusätzlich einmal täglich Katalog und Spielplan aller zugeordneten
Staffeln unabhängig von Spieltagen aktualisieren.

#### Scenario: Staffel vor Anwurf plus zwei Stunden wird nicht abgefragt
- **WHEN** eine Staffel heute eine Begegnung mit Anwurf 16:00 hat und es 17:30 Uhr ist
- **THEN** erfolgt kein Abruf für diese Staffel

#### Scenario: Staffel mit offenen Berichten wird abgefragt
- **WHEN** der früheste heutige Anwurf mehr als zwei Stunden zurückliegt und mindestens eine heutige Begegnung kein `sGID` trägt
- **THEN** wird der Spielplan dieser Staffel abgerufen und gespeichert

#### Scenario: Vollständig versorgter Spieltag beendet den Poll
- **WHEN** jede heutige Begegnung einer Staffel ein `sGID` trägt
- **THEN** erfolgt für diese Staffel am selben Tag kein weiterer Abruf

#### Scenario: Tag ohne Begegnung erzeugt keinen Abruf
- **WHEN** eine Staffel am laufenden Tag keine Begegnung hat
- **THEN** erfolgt außerhalb des täglichen Katalog-Laufs kein Abruf für diese Staffel

### Requirement: Auflösung der Staffel-Kennung über den Live-Katalog

Das System SHALL die Handball4All-Klassen-ID (`gClassID`) einer Staffel bei jedem Lauf aus
dem Katalog (`cmd=po`) auflösen, indem der gespeicherte Staffelcode gegen `gClassSname`
verglichen wird, und SHALL die Klassen-ID NICHT dauerhaft als Zuordnungsschlüssel
speichern.

Das System SHALL Bezirks-Staffeln über den Parameter `o` auflösen, während `og` auf der
Organisation des Verbands bleibt.

Das System SHALL einen Staffelcode, der im Katalog nicht vorkommt, als sichtbaren Fehler
melden und für diese Staffel keinen Abruf durchführen.

#### Scenario: Verbands-Staffel wird aufgelöst
- **WHEN** der Staffelcode `mB-RL-BW` aufgelöst wird
- **THEN** wird der Katalog mit `og` der Verbandsorganisation und ohne `o` abgefragt

#### Scenario: Bezirks-Staffel wird über den Unterorganisations-Parameter aufgelöst
- **WHEN** der Staffelcode `gD-BOL-SRM` aufgelöst wird
- **THEN** wird der Katalog mit `og` der Verbandsorganisation und `o` des Bezirks abgefragt

#### Scenario: Unbekannter Staffelcode
- **WHEN** ein gespeicherter Staffelcode in keinem abgefragten Katalog vorkommt
- **THEN** wird ein Fehler mit dem Code protokolliert und die Staffel im laufenden Durchgang übersprungen

### Requirement: Abruf des Spielberichts anhand des Bereitschaftssignals

Das System SHALL für jede Begegnung, die ein `sGID` ungleich null trägt und für die noch
kein Bericht gespeichert ist, das öffentliche Spielbericht-PDF abrufen.

Das System SHALL Begegnungen ohne `sGID` NICHT abrufen und KEINEN zeitbasierten
Abrufversuch unternehmen.

Das System SHALL ausgehende Anfragen ausschließlich über HTTPS mit Timeout und einem
Projekt-eigenen `User-Agent` durchführen und die PDF-Abrufe seriell mit Pause ausführen.

Das System SHALL einen fehlgeschlagenen Abruf als wiederholbar behandeln und die Zahl der
Versuche je Bericht mitführen.

#### Scenario: Begegnung ohne Bericht wird nicht abgerufen
- **WHEN** eine Begegnung `sGID` null trägt
- **THEN** erfolgt kein PDF-Abruf für diese Begegnung

#### Scenario: Begegnung mit Bericht wird einmal abgerufen
- **WHEN** eine Begegnung erstmals ein `sGID` trägt
- **THEN** wird das PDF abgerufen und ein Bericht angelegt
- **THEN** erfolgt bei einem späteren Poll kein erneuter Abruf für dieselbe Begegnung

#### Scenario: Netzfehler führt zu erneutem Versuch
- **WHEN** der PDF-Abruf mit einem Transportfehler endet
- **THEN** bleibt der Bericht im Zustand `pending`, der Versuchszähler wird erhöht und der Abruf beim nächsten Poll wiederholt

### Requirement: Dauerhafte Ablage des Spielbericht-PDF

Das System SHALL das abgerufene PDF dauerhaft unter dem konfigurierten Ablagepfad
speichern und über eine authentifizierte Route zum Download anbieten.

Das System SHALL beim Ausliefern einen expliziten `Content-Type: application/pdf` aus dem
gespeicherten Wert setzen und `Content-Disposition: attachment` verwenden.

#### Scenario: Bericht wird als Beleg heruntergeladen
- **WHEN** ein eingeloggter Nutzer das PDF eines gespeicherten Berichts abruft
- **THEN** antwortet der Server mit HTTP 200, `Content-Type: application/pdf` und `Content-Disposition: attachment`

#### Scenario: Bericht ohne abgelegte Datei
- **WHEN** zu einem Bericht keine Datei existiert
- **THEN** antwortet der Server mit HTTP 404

### Requirement: Parsing von Mannschaftsliste und Spielverlauf aus der PDF-Textebene

Das System SHALL die Textebene des PDF mit Koordinaten auslesen und daraus zwei Quellen
gewinnen: die Mannschaftslisten beider Teams und den Spielverlauf.

Das System SHALL die Spaltengrenzen der Mannschaftsliste aus den Positionen der Kopfzeile
des Dokuments ableiten und NICHT als feste Koordinaten hinterlegen.

Das System SHALL bei einer nicht interpretierbaren Kopfzeile mit einer klaren Meldung
abbrechen, statt Werte zu raten.

Das System SHALL eine Verlaufszeile unbekannter Form mit ihrem Rohtext und der Art
`other` speichern, statt sie zu verwerfen.

Das System SHALL Platzhalter-Einträge ohne echten Namen nicht als Spieler erfassen.

#### Scenario: Mannschaftsliste wird spaltenrichtig gelesen
- **WHEN** ein Bericht mit Kopfzeilen für Tore, Siebenmeter, Hinausstellungen, Disqualifikation und Gesamtstrafe geparst wird
- **THEN** werden die Werte jeder Spielerzeile der Spalte zugeordnet, deren Kopfbereich sie treffen

#### Scenario: Unbekannte Kopfzeile bricht ab
- **WHEN** die Kopfzeile der Mannschaftsliste nicht erkannt wird
- **THEN** endet der Bericht im Zustand `parse_failed` mit einem Grund, der die Kopfzeile benennt

#### Scenario: Unbekannte Verlaufszeile geht nicht verloren
- **WHEN** eine Zeile des Spielverlaufs keiner bekannten Ereignisform entspricht
- **THEN** wird sie mit Art `other` und ihrem Rohtext gespeichert

#### Scenario: Platzhalter erzeugt keinen Spieler
- **WHEN** eine Zeile der Mannschaftsliste als Namen einen Platzhalter ohne Person trägt
- **THEN** entsteht dafür kein Spieler-Datensatz

### Requirement: Gestufte Kreuzprobe der Zahlen

Das System SHALL die Summe der Tor-Ereignisse des Spielverlaufs gegen den Endstand aus dem
Berichtskopf prüfen. Weichen sie ab, SHALL der Bericht den Zustand `parse_failed` mit
Grund erhalten und es SHALL KEIN Spieler-Datensatz und KEIN Verlaufs-Ereignis zu diesem
Bericht gespeichert werden.

Das System SHALL zusätzlich die Summenspalten der Mannschaftsliste gegen den Spielverlauf
prüfen. Weichen sie ab, SHALL der Bericht dennoch gespeichert und die Abweichung als
Warnung am Bericht festgehalten und in der Oberfläche sichtbar gemacht werden.

Das System SHALL das PDF eines fehlgeschlagenen Berichts erhalten, damit ein späterer Lauf
denselben Bericht ohne erneuten Fremdabruf verarbeiten kann.

#### Scenario: Endstand und Verlauf stimmen überein
- **WHEN** die Summe der Tor-Ereignisse dem Endstand des Berichtskopfs entspricht
- **THEN** wird der Bericht mit Zustand `parsed` gespeichert

#### Scenario: Endstand weicht vom Verlauf ab
- **WHEN** die Summe der Tor-Ereignisse nicht dem Endstand des Berichtskopfs entspricht
- **THEN** erhält der Bericht den Zustand `parse_failed` mit Grund
- **THEN** existiert kein Spieler-Datensatz und kein Verlaufs-Ereignis zu diesem Bericht

#### Scenario: Mannschaftsliste weicht vom Verlauf ab
- **WHEN** eine Summenspalte der Mannschaftsliste nicht zur Zählung aus dem Spielverlauf passt, der Endstand aber stimmt
- **THEN** wird der Bericht mit Zustand `parsed` gespeichert und trägt eine Warnung, die die Abweichung benennt

### Requirement: Fremde Begegnungen erzeugen keine Spieltermine

Das System SHALL alle Begegnungen einer Staffel speichern, auch solche ohne Beteiligung des
eigenen Vereins, und SHALL dabei KEINE Zeile in der Termin-Tabelle `games` anlegen oder
verändern.

Das System SHALL eine Begegnung mit einem bestehenden eigenen Spiel verknüpfen, wenn deren
BWHV-Spielnummer einer vorhandenen `games.external_id` entspricht.

#### Scenario: Poll legt keine Termine an
- **WHEN** eine Staffel mit 90 Begegnungen gepollt wird, von denen keine eine bekannte `games.external_id` trägt
- **THEN** ist die Zahl der Zeilen in `games` unverändert

#### Scenario: Eigenes Spiel wird verknüpft
- **WHEN** eine Begegnung eine BWHV-Spielnummer trägt, die als `games.external_id` existiert
- **THEN** wird die Begegnung mit diesem Spiel verknüpft

### Requirement: Live-Aktualisierung ohne Benachrichtigung

Das System SHALL nach einem Poll, der Daten verändert hat, ein SSE-Ereignis senden, damit
offene Sitzungen nachladen.

Das System SHALL zu eingetroffenen Spielberichten KEINE Push-Nachricht versenden und
KEINEN Eintrag im Ereignis-Protokoll erzeugen.

#### Scenario: Geänderte Daten lösen ein Live-Update aus
- **WHEN** ein Poll neue Ergebnisse oder einen neuen Bericht gespeichert hat
- **THEN** wird ein SSE-Ereignis gesendet

#### Scenario: Kein Bericht löst eine Benachrichtigung aus
- **WHEN** ein Spielbericht abgerufen und gespeichert wurde
- **THEN** entsteht keine Push-Nachricht und kein Eintrag im Ereignis-Protokoll

### Requirement: Manueller Anstoß des Polls

Das System SHALL eine Route bereitstellen, mit der ein Nutzer mit Vereinsfunktion
`vorstand` oder Systemrolle `admin` den Poll einer Staffel unabhängig vom Zeitfenster
auslösen kann. Die Route SHALL nach erfolgreichem Lauf ein SSE-Ereignis senden.

#### Scenario: Vorstand stößt den Poll an
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` den Poll einer Staffel auslöst
- **THEN** antwortet der Server mit HTTP 200 und es wird ein SSE-Ereignis gesendet

#### Scenario: Standard-Nutzer darf nicht anstoßen
- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` und ohne Systemrolle `admin` den Poll auslöst
- **THEN** antwortet der Server mit HTTP 403
