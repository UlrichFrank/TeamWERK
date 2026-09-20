package gamestats

import (
	"context"
	"errors"
	"log/slog"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// PollResult fasst einen Lauf zusammen.
type PollResult struct {
	Staffeln       int
	GamesChanged   int
	ReportsFetched int
	ReportsParsed  int
	ReportsFailed  int
	PlayersLinked  int
}

// Changed meldet, ob der Lauf etwas verändert hat — nur dann lohnt ein
// SSE-Broadcast.
func (r PollResult) Changed() bool {
	return r.GamesChanged > 0 || r.ReportsParsed > 0 || r.ReportsFailed > 0 || r.PlayersLinked > 0
}

// Poller führt Abruf, Ablage und Auswertung zusammen.
type Poller struct {
	store   *Store
	client  *bwhv.Client
	reports *ReportStore
	orgID   int
}

// NewPoller liefert einen Poller.
func NewPoller(store *Store, client *bwhv.Client, reports *ReportStore, orgID int) *Poller {
	return &Poller{store: store, client: client, reports: reports, orgID: orgID}
}

// SyncStaffeln löst die Staffelcodes aller Kader der Saison gegen den Katalog
// auf und legt fehlende Staffeln an. Läuft unabhängig von Spieltagen, damit der
// erste Lauf einer Saison überhaupt einen Spielplan bekommt (design.md §4).
func (p *Poller) SyncStaffeln(ctx context.Context, seasonID int) (PollResult, error) {
	var res PollResult
	codes, err := p.store.StaffelCodes(ctx, seasonID)
	if err != nil {
		return res, err
	}
	for code := range codes {
		st, err := p.store.ResolveStaffel(ctx, p.client, seasonID, p.orgID, code)
		if err != nil {
			if errors.Is(err, ErrStaffelUnbekannt) {
				slog.Error("bwhv: Staffelcode unbekannt", "code", code, "season_id", seasonID)
				continue
			}
			return res, err
		}
		changed, err := p.syncSchedule(ctx, *st)
		if err != nil {
			return res, err
		}
		res.Staffeln++
		res.GamesChanged += changed
	}
	return res, nil
}

func (p *Poller) syncSchedule(ctx context.Context, st Staffel) (int, error) {
	classes, err := p.client.FetchCatalog(ctx, st.OrgID, st.SubOrgID, st.PeriodID)
	if err != nil {
		return 0, err
	}
	var classID string
	for _, c := range classes {
		if c.Sname == st.Code {
			classID = c.ID
			break
		}
	}
	if classID == "" {
		slog.Error("bwhv: Staffel im Katalog verschwunden", "code", st.Code)
		return 0, nil
	}
	sch, err := p.client.FetchSchedule(ctx, st.OrgID, st.SubOrgID, st.PeriodID, classID)
	if err != nil {
		return 0, err
	}
	return p.store.SaveSchedule(ctx, st.ID, sch)
}

// PollStaffel ruft Spielplan und offene Berichte einer Staffel ab.
func (p *Poller) PollStaffel(ctx context.Context, seasonID int, st Staffel) (PollResult, error) {
	var res PollResult
	changed, err := p.syncSchedule(ctx, st)
	if err != nil {
		return res, err
	}
	res.Staffeln = 1
	res.GamesChanged = changed

	pending, err := p.store.PendingReports(ctx, st.ID)
	if err != nil {
		return res, err
	}
	reportURL := p.client.ReportURLPrefix()
	for _, pr := range pending {
		fetched, parsed, failed, err := p.fetchOne(ctx, seasonID, pr, reportURL)
		if err != nil {
			return res, err
		}
		res.ReportsFetched += fetched
		res.ReportsParsed += parsed
		res.ReportsFailed += failed
	}

	kaderID, err := p.store.KaderIDForStaffel(ctx, st.ID)
	if err != nil {
		return res, err
	}
	if kaderID > 0 {
		linked, err := p.store.LinkOwnPlayers(ctx, st.ID, kaderID)
		if err != nil {
			return res, err
		}
		res.PlayersLinked = linked
	}
	return res, nil
}

// fetchOne holt, legt ab und wertet einen einzelnen Bericht aus.
//
// Ein Transportfehler lässt den Bericht im Zustand pending und bleibt damit
// wiederholbar — ob ein gesetztes sGID immer einen fertigen Bericht bedeutet,
// ist nicht verifiziert (design.md §11.2).
func (p *Poller) fetchOne(ctx context.Context, seasonID int, pr PendingReport, reportURL string) (fetched, parsed, failed int, err error) {
	raw, err := p.client.FetchReport(ctx, reportURL, pr.SGID)
	if err != nil {
		slog.Warn("bwhv: Berichtsabruf fehlgeschlagen", "game_no", pr.GameNo, "attempts", pr.Attempts+1, "error", err)
		return 0, 0, 0, p.store.MarkAttempt(ctx, pr.BwhvGameID, pr.SGID)
	}
	path, err := p.reports.Save(seasonID, pr.GameNo, raw)
	if err != nil {
		return 0, 0, 0, err
	}

	rep, perr := bwhv.ParseReport(raw)
	if perr != nil {
		slog.Warn("bwhv: Bericht nicht auswertbar", "game_no", pr.GameNo, "reason", perr.Error())
		return 1, 0, 1, p.store.MarkFailed(ctx, pr.BwhvGameID, pr.SGID, path, perr.Error())
	}
	if len(rep.Warnings) > 0 {
		slog.Info("bwhv: Bericht mit Abweichungen gespeichert",
			"game_no", pr.GameNo, "warnungen", bwhv.WarningsText(rep.Warnings))
	}
	if err := p.store.SaveReport(ctx, pr, rep, path); err != nil {
		return 1, 0, 0, err
	}
	return 1, 1, 0, nil
}

// RunDue führt einen fälligen Lauf über alle offenen Staffeln aus.
func (p *Poller) RunDue(ctx context.Context, seasonID int, due []DueStaffel) (PollResult, error) {
	var total PollResult
	for _, d := range due {
		res, err := p.PollStaffel(ctx, seasonID, d.Staffel)
		if err != nil {
			slog.Error("bwhv: Poll fehlgeschlagen", "staffel", d.Staffel.Code, "error", err)
			continue
		}
		total.Staffeln += res.Staffeln
		total.GamesChanged += res.GamesChanged
		total.ReportsFetched += res.ReportsFetched
		total.ReportsParsed += res.ReportsParsed
		total.ReportsFailed += res.ReportsFailed
		total.PlayersLinked += res.PlayersLinked
	}
	return total, nil
}
