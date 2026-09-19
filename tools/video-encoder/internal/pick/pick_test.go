package pick

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/client"
)

const today = "2026-09-19"

// fixture: Trainer der mB2 (7) mit Kamera-Dienst bei der mA2 (9); die mA2
// hat ein gespieltes Dienst-Spiel und die mB2 zwei gespielte + ein
// zukünftiges Spiel.
func fixture() client.Eligible {
	return client.Eligible{
		GameIDs: []int{4, 5, 6, 9},
		Teams: []client.EligibleTeam{
			{ID: 7, Name: "mB2", UploadWithoutGame: true},
			{ID: 9, Name: "mA2", UploadWithoutGame: false},
		},
		Games: []client.Game{
			{ID: 4, Date: "2026-01-11T00:00:00Z", Opponent: "TSV Altstadt", SeasonID: 3, TeamIDs: []int{7}},
			{ID: 5, Date: "2026-03-01T00:00:00Z", Opponent: "TV Musterstadt", SeasonID: 3, TeamIDs: []int{7}},
			{ID: 6, Date: "2026-03-08T00:00:00Z", Opponent: "SV Beispiel", SeasonID: 3, TeamIDs: []int{9}},
			{ID: 9, Date: "2026-12-01T00:00:00Z", Opponent: "Zukunftsgegner", SeasonID: 3, TeamIDs: []int{7}},
		},
	}
}

// Szenario „Mannschaft nur über Video-Dienst erreichbar": die mA2 ist
// wählbar, listet genau das Dienst-Spiel und bietet keinen Freien Titel.
func TestDutyOnlyTeamSelectableWithoutFreeTitle(t *testing.T) {
	e := fixture()
	teams := Teams(e)
	if len(teams) != 2 || teams[1].ID != 9 {
		t.Fatalf("Teams = %+v, want mB2 then mA2", teams)
	}
	games := Games(e, 9, today)
	if len(games) != 1 || games[0].ID != 6 {
		t.Fatalf("Games(mA2) = %+v, want only the duty game", games)
	}
	if AllowFreeTitle(e, 9) {
		t.Error("duty-only team must not offer a free title")
	}
}

// Szenario „Trainer der eigenen Mannschaft": alle gespielten Spiele, jüngstes
// zuerst, das Zukunftsspiel fehlt, Freier Titel erlaubt.
func TestOwnTeamListsPastGamesNewestFirstWithFreeTitle(t *testing.T) {
	e := fixture()
	games := Games(e, 7, today)
	if len(games) != 2 || games[0].ID != 5 || games[1].ID != 4 {
		t.Fatalf("Games(mB2) = %+v, want [5 4]", games)
	}
	if !AllowFreeTitle(e, 7) {
		t.Error("role team must offer a free title")
	}
}

// Szenario „Keine Berechtigung": leere Antwort → keine Mannschaft.
func TestNoPermissionYieldsNoTeams(t *testing.T) {
	e := client.Eligible{}
	if len(Teams(e)) != 0 || len(Games(e, 7, today)) != 0 || AllowFreeTitle(e, 7) {
		t.Error("empty response must yield nothing selectable")
	}
}

// Szenario „Mannschaft mit Dienst, aber ohne vergangenes Spiel": Team
// wählbar, Spielliste leer.
func TestDutyForFutureGameOnlyKeepsTeamButNoGames(t *testing.T) {
	e := client.Eligible{
		Teams: []client.EligibleTeam{{ID: 9, Name: "mA2"}},
		Games: []client.Game{{ID: 9, Date: "2026-12-01T00:00:00Z", TeamIDs: []int{9}}},
	}
	if teams := Teams(e); len(teams) != 1 || teams[0].ID != 9 {
		t.Fatalf("Teams = %+v", teams)
	}
	if games := Games(e, 9, today); len(games) != 0 {
		t.Fatalf("Games = %+v, want none before the game day", games)
	}
	// Am Spieltag selbst ist das Spiel wählbar (Datum ≤ heute).
	if games := Games(e, 9, "2026-12-01"); len(games) != 1 {
		t.Fatalf("Games on match day = %+v, want the game", games)
	}
}

// Ein Team, das Rollen- und Dienst-Team zugleich ist, erscheint einmal — mit
// dem Rollen-Recht.
func TestTeamsDedupKeepsRoleRight(t *testing.T) {
	e := client.Eligible{Teams: []client.EligibleTeam{
		{ID: 7, Name: "mB2", UploadWithoutGame: false},
		{ID: 7, Name: "mB2", UploadWithoutGame: true},
	}}
	teams := Teams(e)
	if len(teams) != 1 || !teams[0].UploadWithoutGame {
		t.Fatalf("Teams = %+v, want one entry with role right", teams)
	}
}

// Spiel mit zwei erlaubten Mannschaften erscheint unter beiden.
func TestGameWithTwoTeamsListedUnderBoth(t *testing.T) {
	e := client.Eligible{Games: []client.Game{{ID: 1, Date: "2026-01-01", TeamIDs: []int{7, 9}}}}
	if len(Games(e, 7, today)) != 1 || len(Games(e, 9, today)) != 1 || len(Games(e, 8, today)) != 0 {
		t.Error("game must be listed under each allowed team and nowhere else")
	}
}
