## REMOVED Requirements

### Requirement: Spielbericht-Duty-Slot nur für Presseteam sichtbar
**Reason**: Die System-Rolle `presseteam` entfällt. Der Spielbericht-Slot ist damit ein
Dienst wie jeder andere und folgt allein den allgemeinen Sichtbarkeitsregeln der
Dienstbörse (Team-Scope + Zielgruppe).
**Migration**: Keine — im `GET /api/duty-board`-Handler existierte nie ein Filter auf diesen
Diensttyp; die Anforderung beschrieb einen Zustand, den der Code nicht herstellte.

### Requirement: Spielbericht-Slot-Ziehen prüft Rolle
**Reason**: Die System-Rolle `presseteam` entfällt; der Guard hätte danach keinen
unterscheidenden Wert mehr. Wer den Bericht schreibt, bestimmt sich über den Besitz des
Dienstes — genau das, was das Ziehen herstellt.
**Migration**: `assertSlotTakePermitted` und der Fehlercode `role_required` auf
`POST /api/duty-slots/{id}/take` entfallen. Der Slot wird gezogen wie jeder andere Dienst.
