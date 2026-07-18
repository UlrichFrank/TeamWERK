## Why

Der Anwesenheit-Tab in `/profil` erscheint aktuell für **jedes** verlinkte
Mitglied — auch für Trainer/Vorstand/Kassierer, die selbst nie Spieler waren.
Konkreter Bug-Report: Thomas Eisele (Vereinsfunktion `trainer`, kein `spieler`)
sieht auf `/profil` den Tab „Anwesenheit" und kann sich dort sogar selbst
auswählen — obwohl es keine Anwesenheitsdaten zu ihm gibt und geben soll.

Ursache: `ProfilePage.tsx` gated den Tab auf `ownMember !== null` statt auf
die **Vereinsfunktion `spieler`**. Das ist eine Modell-Verletzung — Anwesenheit
ist im Fachmodell ein Spieler-Konzept (Trainer/Vorstand pflegen und sehen sie
im Team-Kontext unter `/team/{id}/anwesenheit`, nicht in ihrem Profil).

Die Spec (`attendance-statistics/spec.md`, Requirement „Trainer- und
Spieler-Sichten im Frontend") legt fest, dass es einen Tab gibt, sagt aber
**nicht**, wer ihn sehen darf. Diese Lücke schließt der Change.

## What Changes

- **Frontend-Regel:** Anwesenheit-Tab in `/profil` sichtbar nur wenn
  `own_member.club_functions` enthält `spieler` ODER mindestens eines der
  verlinkten `children` es enthält.
- **Selbst-Auswahl im Content** (`ProfilAnwesenheitContent`): Buttons für
  `own_member` nur wenn Spieler; Buttons für Kinder nur wenn Spieler.
- **Trainer-Drilldown bleibt intakt:** `/profil/anwesenheit?member=X` mit
  `forcedMemberId` überspringt die Options-Logik weiter — der Trainer muss
  nicht selbst Spieler sein, um über die Team-Sicht in die Detailstatistik
  eines Spielers zu drillen (bestehender Flow aus `TeamAnwesenheitPage.tsx`).

Backend unverändert: `GET /api/profile/me` liefert `club_functions` bereits an
sowohl `own_member` als auch jedem `children[i]` mit
(`internal/members/handler.go:1258-1260`, GROUP_CONCAT aus
`member_club_functions`).

## Impact

- **Betroffene Nutzer:** Alle System-Rolle `standard`-Nutzer, die verlinktes
  Mitglied sind, aber nicht die Vereinsfunktion `spieler` haben (typisch:
  reine Trainer, Vorstände, Kassierer ohne Spieler-Historie im Aktivenkader).
- **Kein Datenverlust:** Historische Anwesenheiten hängen an
  `kader × season × session/game`, nicht am Profil. Ex-Spieler, die jetzt
  Trainer sind, erreichen ihre alten Zahlen über `/team/{id}/anwesenheit`
  bzw. den Drilldown weiterhin.
- **Kein Backend-Change:** kein Migrations- oder API-Vertrag betroffen.

## Test-Anforderungen

Vitest-Tests in `web/src/pages/__tests__/ProfilePage.attendance-tab.test.tsx`
(neu) — decken folgende Konstellationen ab; jeweils Testname und garantierte
Invariante:

| Konstellation | Testname | Erwartung |
|---|---|---|
| `own=null`, `kids=[]` | `keinEigenesMitglied_KeinTab` | Tab-Liste enthält kein „Anwesenheit" |
| `own=[trainer]`, `kids=[]` | `nurTrainer_KeinTab` | Tab-Liste enthält kein „Anwesenheit" (Thomas-Fall) |
| `own=[spieler]`, `kids=[]` | `spieler_TabSichtbar_OwnAuswaehlbar` | Tab sichtbar; Content zeigt own als Option |
| `own=[trainer]`, `kids=[spieler]` | `trainerMitSpielerKind_TabNurKindOption` | Tab sichtbar; Content zeigt nur Kind, nicht own |
| `own=[spieler,trainer]`, `kids=[spieler]` | `spielerElternteil_BeideAuswaehlbar` | Tab sichtbar; beide Optionen |
| `forcedMemberId=42` bei `own=[trainer]` (kein spieler) | `forcedMemberId_UmgehtSpielerFilter` | `ProfilAnwesenheitContent` rendert `AttendanceStatsView` mit memberId=42 direkt (keine leere Auswahl, kein 403) |

## Nicht Teil dieses Changes

- Umbenennung/Trennung des Trainer-Drilldown-URLs (`/profil/anwesenheit?member=X`
  ist semantisch für Trainer irreführend, funktioniert aber). Falls
  überarbeitet, eigener Change.
- Ex-Spieler-Historie im Profil („Ich war 2019 Spieler und will meine Zahlen
  sehen"): bewusst nicht abgedeckt — Historie ist über Team-Sicht erreichbar.
