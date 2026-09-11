## Context

Befunde und Fundstellen: `docs/reviews/2026-09-11-architektur-review.md`, Abschnitt Security.
Zwei Muster im Repo geben die Richtung vor: `internal/media/handler.go` (Sniffing zuerst,
`extByMime`, Endung aus dem Typ) und die Permission-Matrix (`internal/permissions/matrix_test.go`)
als Drift-Schutz. Beides wird hier angewandt, nicht neu erfunden.

## Goals / Non-Goals

**Goals:** die sieben Befunde schließen, jeder mit einem Test, der ohne den Fix rot ist; keine
Verhaltensänderung für berechtigte Nutzer.

**Non-Goals:** die Fixture-Matrix für Objekt-Autorisierung (Welle 2), `internal/httpx`,
Löschkonzept/Art.-15-Export, FTS5, Rotation des Vereins-Keypairs.

## Decisions

**1. Upload: Sniffing ist die einzige Wahrheit.** `http.DetectContentType` auf die ersten
512 Byte, danach `sniffImageType` als Fallback; die Client-Angabe `Content-Type` wird nur noch
für die Fehlermeldung gelesen. Endung aus einer Map Typ→Endung (kein `filepath.Ext(hdr.Filename)`
mehr). Für PDF-Uploads (SEPA-Mandat) gilt dasselbe mit `application/pdf`; clientseitig
verschlüsselte Blobs (`IsClientEncryptedBytes`) behalten ihren bestehenden Sonderpfad, werden
aber immer mit `attachment` ausgeliefert. `streamFile` setzt `Content-Type` aus dem gespeicherten
Wert bzw. Sniffing, nie aus der Endung, und `Content-Disposition: attachment` für alles, was
kein `image/*` ist. Bestandsdateien mit falscher Endung sind damit ebenfalls entschärft (die
Auslieferung entscheidet, nicht der Dateiname).

**2. Tresor: `extractable=false` in IndexedDB.** WebCrypto erlaubt nicht-exportierbare
`CryptoKey`-Objekte, die per Structured Clone in IndexedDB persistieren. Ein XSS könnte den
Schlüssel dann *benutzen* (solange die Seite offen ist), aber nicht *exfiltrieren*; nach
Timeout/Logout ist er weg. Das schließt genau den Pfad, der die Zusage dauerhaft bricht.
`sessionStorage` bleibt für den Timeout-Zeitstempel. Bestandsformat wird nicht migriert: ein
`sessionStorage['vk']` wird beim Mount entfernt, der Nutzer entsperrt einmal neu.
Alternative verworfen: Schlüssel nur im Speicher halten (bricht Reload/Navigation, die
`VaultContext` bewusst überlebt).

**3. Medien-Sichtbarkeit folgt dem referenzierenden Objekt.** `media.Serve` ist erlaubt, wenn
(a) `media.uploaded_by = user`, oder (b) eine `messages`-Zeile mit dieser `media_id` in einer
Konversation liegt, in der der Nutzer Mitglied ist (`conversation_members`, inklusive
ausgetretener Mitglieder wie `isMember`, denn der Verlauf bleibt lesbar), oder (c) eine
`broadcasts`-Zeile mit dieser `media_id` eine `broadcast_reads`-Zeile für den Nutzer hat.
Sonst 404 (nicht 403), damit die Existenz einer ID nicht erratbar ist. Eine Query mit `EXISTS`.

**4. Objekt-Gates nutzen die vorhandenen Helfer.** `trainings.GetSession` → `hasKaderAccess`
(wie die Schwesterrouten); `games.SaveLineup` → Trainer eines beteiligten Teams nach dem Muster
`canRecordGameAttendance` (admin und `sportliche_leitung` weiterhin vereinsweit);
`chat.MarkRead`/`LeaveConversation` → `isActiveMember`; `duties.Fulfill`/`CashSubstitute` →
`policy.CanFulfillAssignment` (admin, trainer, sportliche_leitung: das bestehende Recht aus
Router-Tier, Capability `fulfill_duties` und Matrix; das Review hatte irrtümlich vorstand/kassierer
genannt, was die Route faktisch admin-only gemacht und Trainern das Abhaken genommen hätte),
`RowsAffected == 0` → 404, Fehler geprüft. Ob Vorstand/Kassierer das Recht zusätzlich bekommen,
ist eine fachliche Frage für einen eigenen Change (Tier, Policy, Capability, Matrix gemeinsam).

**5. HSTS im Go-Prozess, nicht in Nginx.** `securityHeaders(hstsEnabled)` existiert bereits;
`make deploy` ergänzt `HSTS_ENABLED=true` idempotent in `/etc/teamwerk/env` (gleiche Schleife
wie `TRAINING_DIARY_DIR`). Nginx bleibt auskommentiert, ein Header reicht. `max-age` 2 Jahre,
`includeSubDomains`; kein `preload` (Alias-Domain-Übergang offen).

**6. `JWT_SECRET` ≥ 32 Byte und ≠ Beispielwert.** Fail-fast in `config.Load` mit klarer
Meldung. `.env.example` bekommt einen leeren Wert plus Kommentar mit Generator-Befehl
(`openssl rand -base64 48`).

**7. Impersonation: Claim + Log + Event.** `Claims.ImpersonatedBy *int` (JSON `impersonated_by`,
`omitempty`), gesetzt nur von `Impersonate`. `slog.Warn("impersonation", actor, target)`. Event-Log
an den Admin selbst (Kategorie `admin`, Titel „Als <Name> angemeldet", URL `/nutzer`), damit
die Spur drei Tage im Dashboard und nicht nur im Log liegt. Die Kategorie braucht eine Migration
(`CHECK`-Constraint, Tabellen-Rebuild nach Muster `049`). Das Frontend zeigt den Banner schon
heute; der Claim macht das Token serverseitig unterscheidbar (Grundlage für spätere Sperren
sensibler Aktionen unter Impersonation).

**8. Abhängigkeiten.** Alle Alerts sind npm und haben gepatchte Versionen (`react-router` 7.18,
`vitest`/`@vitest/mocker` 4.1.11, `undici` 7.29, `fast-uri` 3.1.6, `browserslist` 4.28.7,
`baseline-browser-mapping` 2.11, `postcss-selector-parser` 6.1.3). Direkte Deps per
`pnpm update`, transitive per `pnpm.overrides` nur wenn nötig. CI: `govulncheck ./...` und
`pnpm audit --audit-level=high` als eigener Job, nicht im Gate (externe Datenbank, darf nicht
den Merge blockieren, soll aber sichtbar rot werden).

## Risks / Trade-offs

- Upload-Härtung könnte Bestandsclients treffen, die exotische Bildtypen senden → Whitelist
  bleibt JPEG/PNG/WEBP(/GIF für media), Fehlermeldung nennt den erkannten Typ.
- Nicht-exportierbarer Schlüssel: Safari-Private-Mode hat kein persistentes IndexedDB →
  Fallback: Schlüssel nur im Speicher, Tresor nach Reload gesperrt (kein Sicherheitsverlust).
- Media-Gate 404 statt 403: Frontend zeigt „Bild nicht verfügbar" (bestehender `AuthImage`-Pfad).
- `JWT_SECRET`-Prüfung ist ein Deploy-Risiko → Proposal nennt den Vorab-Check; `make deploy`
  bricht vor dem Restart ab, wenn der Wert auf dem Server zu kurz ist.
- Dependabot-Bumps (react-router 7.x) können Verhalten ändern → volle Test-Suite und E2E
  laufen vor dem Merge.

## Migration Plan

Deploy wie üblich; vorher `JWT_SECRET` auf dem VPS prüfen. Migration `060` läuft automatisch.
Tresor-Inhaber entsperren einmal neu. Rollback: vorheriges Binary; Migration `060` down ist ein
Tabellen-Rebuild ohne Datenverlust (Zeilen mit Kategorie `admin` werden gelöscht).
