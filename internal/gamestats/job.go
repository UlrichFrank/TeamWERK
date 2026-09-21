package gamestats

import (
	"context"
	"log/slog"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"

	"database/sql"

	"github.com/teamstuttgart/teamwerk/internal/background"
	"github.com/teamstuttgart/teamwerk/internal/bwhv"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// catalogHour ist die Stunde des täglichen Katalog- und Spielplan-Laufs.
//
// Er läuft unabhängig von Spieltagen, weil der erste Lauf einer Saison sonst
// nie ein Fenster bekäme: das Fenster wird aus dem gespeicherten Spielplan
// abgeleitet, und den gibt es vorher nicht (design.md §4). Er ist außerdem der
// Weg, auf dem Verlegungen in den Spielplan kommen.
const catalogHour = 6

// pollRunTimeout deckt einen vollständigen Lauf über alle Staffeln ab.
const pollRunTimeout = 30 * time.Minute

// SchedulerJob liefert den Minutentakt-Einstieg des BWHV-Polls.
//
// Er wird von der Komposition (main.go) in den Scheduler eingehängt, statt dass
// der Scheduler dieses Paket importiert: scheduler ist Foundation und darf
// keine Domäne importieren. Die Alternative wäre, den ganzen Lauf im Scheduler
// zu duplizieren (design.md §8).
//
// Der eigentliche Lauf läuft über background.Go (Goroutine-Gate) mit eigenem
// Context: er dauert Minuten und darf den Scheduler-Tick nicht blockieren.
func SchedulerJob(db *sql.DB, cfg *appconfig.Config) func() {
	return func() { runPollTick(db, cfg) }
}

func runPollTick(db *sql.DB, cfg *appconfig.Config) {
	if cfg == nil || cfg.BwhvReportDir == "" {
		return
	}
	store := NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seasonID, err := store.ActiveSeason(ctx)
	if err != nil {
		return // ohne aktive Saison gibt es nichts abzurufen
	}

	now := time.Now().In(timez.Berlin())
	poller := NewPoller(store, bwhv.NewClient(),
		NewReportStore(cfg.BwhvReportDir), cfg.BwhvOrgID)

	if now.Hour() == catalogHour && now.Minute() == 0 {
		background.Go("bwhv-catalog", func() {
			runCtx, runCancel := context.WithTimeout(context.Background(), pollRunTimeout)
			defer runCancel()
			res, err := poller.SyncStaffeln(runCtx, seasonID)
			if err != nil {
				slog.Error("bwhv: Katalog-Lauf fehlgeschlagen", "error", err)
				return
			}
			slog.Info("bwhv: Katalog-Lauf", "staffeln", res.Staffeln, "spiele_geaendert", res.GamesChanged)
		})
		return
	}

	due, err := store.DueStaffeln(ctx, seasonID, now)
	if err != nil {
		slog.Error("bwhv: Fälligkeit nicht ermittelbar", "error", err)
		return
	}
	if len(due) == 0 {
		return
	}
	background.Go("bwhv-poll", func() {
		runCtx, runCancel := context.WithTimeout(context.Background(), pollRunTimeout)
		defer runCancel()
		res, err := poller.RunDue(runCtx, seasonID, due)
		if err != nil {
			slog.Error("bwhv: Poll fehlgeschlagen", "error", err)
			return
		}
		if res.Changed() {
			slog.Info("bwhv: Poll",
				"staffeln", res.Staffeln, "spiele_geaendert", res.GamesChanged,
				"berichte_geparst", res.ReportsParsed, "berichte_fehlgeschlagen", res.ReportsFailed,
				"spieler_zugeordnet", res.PlayersLinked)
		}
	})
}
