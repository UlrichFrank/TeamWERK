// Package pick baut aus der Berechtigungs-Antwort des Servers
// (GET /api/videos/upload-eligible-games) die Mannschafts- und Spielauswahl
// des Tools. Reine Funktionen, damit die Auswahl-Regeln ohne Fyne testbar
// sind — die Oberfläche ruft sie nur auf und beschriftet.
//
// Warum nicht /api/teams und /api/games: beide beantworten die Frage nach der
// Kader-Zugehörigkeit (Trainer/Spieler/Eltern), nicht nach dem Upload-Recht.
// Wer einen Video-Dienst bei einer Mannschaft hat, mit der er sonst nicht
// verbunden ist, sieht sie dort nie — der Server erlaubt den Upload aber
// (video-upload-eligible-teams).
package pick

import (
	"sort"

	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/client"
)

// Teams liefert die wählbaren Mannschaften in Server-Reihenfolge (Rollen-Teams
// zuerst, dann reine Dienst-Teams), Duplikate nach ID entfernt — beim ersten
// Vorkommen gewinnt ein gesetztes UploadWithoutGame.
func Teams(e client.Eligible) []client.EligibleTeam {
	out := make([]client.EligibleTeam, 0, len(e.Teams))
	idx := map[int]int{}
	for _, t := range e.Teams {
		if i, dup := idx[t.ID]; dup {
			out[i].UploadWithoutGame = out[i].UploadWithoutGame || t.UploadWithoutGame
			continue
		}
		idx[t.ID] = len(out)
		out = append(out, t)
	}
	return out
}

// Games liefert die BEREITS GESPIELTEN Spiele (Datum ≤ today, "2006-01-02"),
// unter denen der Upload für teamID erlaubt ist — jüngstes zuerst. Ein
// zukünftiges Spiel in der Auswahl wäre nur Rauschen: ein Upload gehört immer
// zu einem schon gespielten Spiel. Der Server liefert nur die aktive Saison,
// ein Saisonfilter ist deshalb nicht nötig.
func Games(e client.Eligible, teamID int, today string) []client.Game {
	var out []client.Game
	for _, g := range e.Games {
		if client.DateOnly(g.Date) > today || !containsInt(g.TeamIDs, teamID) {
			continue
		}
		out = append(out, g)
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, dj := client.DateOnly(out[i].Date), client.DateOnly(out[j].Date)
		if di != dj {
			return di > dj
		}
		return out[i].ID > out[j].ID
	})
	return out
}

// AllowFreeTitle meldet, ob für teamID ein Upload ohne Spielbezug angeboten
// werden darf. Das ist genau der Rollen-Pfad (upload_without_game); für ein
// reines Dienst-Team lehnte POST /api/videos den Upload ohne game_id mit 403
// ab — die Option gar nicht anzubieten ist besser als eine Sackgasse im Klick.
func AllowFreeTitle(e client.Eligible, teamID int) bool {
	for _, t := range e.Teams {
		if t.ID == teamID && t.UploadWithoutGame {
			return true
		}
	}
	return false
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
