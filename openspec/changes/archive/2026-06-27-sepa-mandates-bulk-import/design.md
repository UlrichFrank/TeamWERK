## Context

Auf `/mitglieder` existiert heute ein Dropdown „Weitere Aktionen" mit zwei Einträgen (Import CSV, Export CSV). Per-Member kann ein SEPA-Mandat-Dokument auf dem Mitglieder-Detail-Tab „Bankdaten" via `POST /api/upload/sepa-mandat/{id}` hochgeladen werden (PDF/Bild, max 2 MB, gespeichert unter `<uploadDir>/sepa-mandats/<uuid>.<ext>`, Pfad in `members.sepa_mandat_path`).

Aktueller Engpass: Der Verein hat ~100 unterzeichnete Mandate als PDFs in einem lokalen Verzeichnis (Dateinamen `VornameNachname.pdf`). Einzeln zuzuordnen bedeutet pro Mitglied: Suchen → Detail öffnen → Tab Bankdaten → Datei wählen → Upload → zurück. Bei 100 Stück nicht zumutbar.

Stakeholder: Vorstand & Kassierer (führen den Import durch). Die SEPA-Mandat-Empfänger (Mitglieder, Eltern) sind nur indirekt betroffen.

Vorhandene Bausteine, die wiederverwendet werden:
- `upload.Handler.saveFile`: validiert + persistiert eine Multipart-Datei.
- `pdfAndImageTypes` / `maxSepaBytes` (2 MB) Konstanten.
- `EventHub.Broadcast` für SSE.
- Permissions-Tier `Vorstand + Kassierer` (siehe `internal/app/router.go` & `permissions/matrix_test.go`).

## Goals / Non-Goals

**Goals:**
- Ein Bulk-Pfad, der so wenig wie möglich Code dupliziert (Reuse von `saveFile`-Validierung).
- Idempotenz: Mehrfacher Import desselben Verzeichnisses ist sicher, nichts wird überschrieben oder doppelt importiert.
- Klarer Report: Der Nutzer sieht pro Datei genau, was passiert ist — `imported`, `already_exists`, `no_match`, `ambiguous`.
- Berechtigungs-Parität mit Einzel-Upload (`vorstand` + `kassierer` + `admin`).
- Keine neue DB-Schema-Änderung.

**Non-Goals:**
- Manuelles Nachzuordnen von `no_match`/`ambiguous` im Bulk-Modal (User fällt auf den Per-Member-Upload-Tab zurück).
- OCR / PDF-Parsing zur Extraktion von Name oder Mandatsdatum aus dem PDF (zu fehleranfällig; `sepa_mandat_date` wird weiter manuell gepflegt).
- ZIP-Upload als Alternative (`webkitdirectory` reicht).
- Versionierung / Audit-Log der Bulk-Imports (über das append-only Beitragslauf-Protokoll hinaus existiert kein Bedarf).
- Async-Verarbeitung mit Job-Queue. Für 50 MB Multipart-Body in einem Request ist die Latenz akzeptabel (lokales Filesystem, kein S3).

## Decisions

### 1. Match-Algorithmus: normalisierter Konkatenations-Vergleich, beide Reihenfolgen

**Entscheidung:** Pro PDF wird der Basename (ohne `.pdf`-Extension) normalisiert und mit der normalisierten Konkatenation `first_name+last_name` _und_ `last_name+first_name` jedes Mitglieds verglichen.

**Normalisierung** (Server-Side, einmalig beim Build der Lookup-Map):
1. Lowercase via `strings.ToLower`.
2. Umlaute & ß ersetzen: `ä→ae`, `ö→oe`, `ü→ue`, `ß→ss` (deutsche Konvention; deckt die typischen Vereinsmitglieder ab).
3. Entfernen: Leerzeichen, Bindestriche, Apostrophe, Unterstriche, Punkte.
4. Optional: Diakritika via `golang.org/x/text/unicode/norm` NFD + Strip nonspacing marks — _falls_ Mitglieder mit z.B. französischen Akzenten vorkommen. **Default: nicht hinzunehmen**, weil die Standardbibliothek reicht und wir keine neuen Dependencies wollen. Re-evaluieren, wenn `no_match` darauf hindeutet.

**Reihenfolge tolerant:** „MaxMustermann" und „MustermannMax" matchen beide. Verhindert, dass der User wegen Sortierfehlern manuell nacharbeiten muss.

**Mehrdeutigkeit:** Wenn nach Normalisierung mehrere Members matchen (zwei Mitglieder mit identischem Vor-/Nachnamen, z.B. Vater/Sohn), wandert die Datei in `ambiguous` und beide Kandidaten werden gemeldet. **Bewusst** wird _nicht_ auf Mitgliedsnummer oder Geburtsdatum gefallback-matcht — der User entscheidet manuell.

**Alternativen erwogen:**
- _Levenshtein/Fuzzy-Match_: Verworfen — produziert False Positives bei kurzen Namen; die Eindeutigkeit nach Normalisierung reicht für das gemeldete Datenformat.
- _Mitgliedsnummer im Dateinamen erzwingen_: Wäre robuster, aber der User hat das Format `VornameNachname.pdf` als Ist-Stand bestätigt. Umbenennen von 100 PDFs ist nicht zumutbar.
- _Match in SQL via LOWER()/REPLACE()-Kaskade_: Verworfen — die Umlaut-Substitution lässt sich in SQLite nur über mehrere geschachtelte `REPLACE` ausdrücken; im Go-Code lesbarer + testbarer.

### 2. Keine separate Preview-Route — Single-Step Import mit Report

**Entscheidung:** Es gibt **eine** Route `POST /api/members/sepa-mandates/import`, die in einem Aufruf verarbeitet, persistiert und einen Report zurückgibt.

**Rationale:** Bestehende Mandate werden nie überschrieben (Skip-Logik macht den Vorgang nicht-destruktiv). Eine Preview-Round-Trip würde nur die Latenz verdoppeln, ohne den User vor Schaden zu bewahren. Der CSV-Import hat Preview, weil er destruktiv ist (Append/Update überschreibt Felder); SEPA-Bulk ist es nicht.

**Trade-off:** User sieht `no_match`-Liste erst _nach_ dem Hochladen der vollen 50 MB. Akzeptiert, weil die Hochlade-Dauer im LAN/lokal vernachlässigbar ist.

### 3. Set `sepa_mandat=1` zusätzlich zum Pfad — abweichend vom Einzel-Upload

**Entscheidung:** Beim Bulk-Match wird `sepa_mandat_path = <neuer Pfad>` **und** `sepa_mandat = 1` gesetzt. `sepa_mandat_date` bleibt unverändert.

**Rationale:** Der Einzel-Upload setzt heute nur den Pfad — das ist eine bekannte UX-Schwäche (User muss separat den Bool im Formular toggeln). Beim Bulk-Import ist die Intention explizit „diese Mandate liegen unterzeichnet vor"; das Bool gleich mitzusetzen erspart ~100 Folge-Klicks. Das Datum bleibt offen, weil es im PDF nicht ableitbar ist und für den Beitragslauf-Export weiterhin Pflicht ist — Mitglieder ohne `sepa_mandat_date` tauchen dort als „kein SEPA-Mandat" auf, was den User zwingt, das Datum nachzupflegen.

**Alternativen erwogen:**
- _Auch `sepa_mandat_date=DATE('now')` setzen_: Verworfen — das wäre eine Lüge (das PDF wurde nicht heute unterschrieben).
- _Einzel-Upload synchron anpassen_: Out of scope für diesen Change. Wenn das gewünscht ist, separate Proposal.

### 4. Per-File-Atomarität, kein DB-Transaction-Wrapper über alle Files

**Entscheidung:** Pro PDF wird (a) die Datei auf das Filesystem geschrieben, dann (b) ein einzelnes `UPDATE members SET sepa_mandat_path=?, sepa_mandat=1 WHERE id=?` ausgeführt. Schlägt (b) fehl, wird die Datei via `os.Remove` wieder entfernt (analog zur bestehenden `UploadSepaMandat`-Logik).

**Rationale:** Eine globale Transaction über 100 Files würde bei einem einzigen Fehler alles zurückrollen — das ist die schlechtere UX. Per-File-Atomarität bedeutet: 99 erfolgreich importierte Mandate bleiben erhalten, das eine fehlerhafte landet im Report (mit Fehler-Detail), User retried gezielt.

**Trade-off:** Schreib-Fehler in der DB nach erfolgreichem File-Write hinterlässt im worst case eine verwaiste Datei, falls das `os.Remove` auch fehlschlägt. Akzeptiert — entspricht dem Verhalten des Einzel-Uploads.

### 5. Multipart-Limit 50 MB; Streaming pro File

**Entscheidung:** `http.MaxBytesReader` mit 50 MB Gesamtkappung; `r.ParseMultipartForm(memLimit)` mit Mem-Limit 8 MB (Default), Rest wird auf Disk gepuffert. Pro Part dann wieder die 2-MB-Validierung von `saveFile`.

**Rationale:** Bei ~250 KB pro Mandat-PDF deckt 50 MB ~200 Mandate ab — reicht für den größten realistischen Use-Case. Mem-Limit 8 MB bedeutet, dass nur ein Teil der Parts gleichzeitig im RAM liegt; der VPS mit 1 GB RAM verkraftet das.

**Trade-off:** Sehr große Verzeichnisse (>200 Files) müssen aufgesplittet werden. Im Report melden wir, _welche_ Files akzeptiert wurden — falls Bedarf, kann der User in zwei Tranchen importieren (idempotent dank Skip).

### 6. Frontend: `<input type="file" webkitdirectory>` mit clientseitigem `.pdf`-Filter

**Entscheidung:** `<input type="file" webkitdirectory multiple>` öffnet den Verzeichnis-Picker (Chrome, Edge, Safari ≥ 11.1, Firefox); das Frontend filtert `files.filter(f => f.name.toLowerCase().endsWith('.pdf'))` _vor_ dem Upload, zeigt eine Vorschauliste der ausgewählten PDFs, User bestätigt mit „Hochladen & Importieren".

**Rationale:** `webkitdirectory` ist non-standard, aber breit unterstützt; der User-Workflow „Verzeichnis anhängen" erfordert es. Clientseitiger Filter spart Bandbreite (keine `.DS_Store`, keine JPGs, keine `Thumbs.db`).

**Trade-off:** Mobile-Browser unterstützen `webkitdirectory` nicht zuverlässig. Akzeptiert — der Bulk-Import ist ein Desktop-Use-Case (Vorstand am Rechner).

### 7. Permissions: identisch zum Einzel-Upload

**Entscheidung:** `POST /api/members/sepa-mandates/import` wird unter den `Vorstand+Kassierer`-Tier im Router gemountet (analog `POST /api/upload/sepa-mandat/{id}`). `admin` umgeht den Function-Check via `RequireRole`-Override (Standardverhalten).

**Rationale:** Wer einzeln hochladen darf, darf auch bulk-importieren. Kassierer hat dadurch denselben Schreib-Pfad wie heute schon — keine Erweiterung der Berechtigungs-Matrix.

## Risks / Trade-offs

- **[Namens-Kollisionen (Vater/Sohn mit gleichem Namen)]** → Mitigation: `ambiguous`-Liste im Report mit beiden Kandidaten; User fällt zurück auf den Per-Member-Tab. Kein automatisches Raten.
- **[Falsch normalisierter Sonderzeichen-Name]** → Mitigation: Test-Fixture mit Umlauten + ß; bei realer Häufung von `no_match` durch Akzente einmalig NFD-Diakritika-Stripping nachziehen.
- **[Verwaiste Datei nach DB-Fehler]** → Mitigation: `os.Remove` im Fehlerpfad, Logging. Identisch zum heutigen Verhalten von `UploadSepaMandat` — kein Regress.
- **[Großer Multipart-Request blockiert HTTP-Worker]** → Mitigation: Single-Threaded Verarbeitung pro Request, aber Chi nutzt pro Request einen Goroutine; andere Requests bleiben bedient. 50 MB Cap verhindert Speicher-Explosion.
- **[`webkitdirectory` Browser-Inkompatibilität]** → Mitigation: Fallback auf `<input type="file" multiple>` (Datei-Mehrfachauswahl) — User wählt die PDFs einzeln im File-Picker. Wird in Tasks angeführt.
- **[`sepa_mandat=1` ohne Datum bleibt unauffällig]** → Mitigation: Im Beitragslauf-Filter taucht das Mitglied trotzdem als „kein SEPA-Mandat" auf (weil das Datum fehlt) — der User wird also spätestens dort daran erinnert.
