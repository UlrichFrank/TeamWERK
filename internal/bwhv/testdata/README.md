# Fixtures für internal/bwhv

Alle Tests dieses Pakets laufen gegen diese Dateien. **Kein Test ruft Handball4All
live ab** — dieselbe Regel wie in `internal/h4aimport/testdata/`.

## Herkunft

Die JSON-Dateien sind gekürzte, ansonsten wörtliche Antworten der öffentlichen
Schnittstelle `spo.handball4all.de/service/if_g_json.php`, abgerufen am
20.09.2026:

| Datei | Aufruf |
|---|---|
| `catalog_bw.json` | `cmd=po&og=216&p=142` (Verbandsebene, auf 6 Staffeln gekürzt) |
| `catalog_srm.json` | `cmd=po&o=251&og=216&p=142` (Bezirk SRM, auf 4 Staffeln gekürzt) |
| `schedule_mb_rl_bw.json` | `cmd=ps&og=216&p=142&cl=161291&ca=1` (auf 3 gespielte + 5 künftige Begegnungen gekürzt) |
| `error_permission_denied.json` | `cmd=po&og=251` — die og/o-Falle, wörtlich |

## Personenbezogene Daten: keine

`spielbericht_905272.pdf` ist **nicht** der echte Bericht, sondern ein
synthetischer Nachbau mit **identischer Geometrie**: dieselben Textpositionen,
Schriftgrößen, Spaltenkoordinaten, Spielnummer, Staffel, Halle, Endstand und
Zeitstempel — aber durchgängig erfundene Personen. Verifiziert wurde, dass beide
Dateien auf dieselbe Struktur parsen (26 Kadereinträge, 18 Verlaufsschlüssel,
0 mehrdeutige Zuordnungen, gleiche Trikotnummern-Kollisionen `[16, 46]`).

Der echte Bericht trägt Klarnamen und Jahrgänge von rund 26 Personen,
überwiegend Minderjährigen. Er gehört damit nicht in ein Repository, das nach
`openspec/specs/public-repo-hygiene/` künftig öffentlich wird — dort gilt
„kein einziger personenbezogener Datensatz, weder im Tree noch in der Historie".
Aus demselben Grund sind die Schiedsrichter-Namen in `schedule_mb_rl_bw.json`
ersetzt.

Dass die Anwendung im Betrieb echte Berichte verarbeitet, ist davon unberührt —
das ist eine Entscheidung über die Datenbank der Instanz, nicht über das Repo
(`design.md` §2).

## Erneuern

Der Generator liegt nicht im Repo. Ein neues PDF-Fixture entsteht, indem man aus
einem echten Bericht die Textläufe mit Koordinaten extrahiert, alle Personennamen
konsistent ersetzt (dieselbe Person überall gleich, auch zwischen Kader und
Spielverlauf) und die Läufe an unveränderten Positionen neu emittiert.
