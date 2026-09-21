package gamestats

import (
	"context"
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
	if len(codes) == 0 {
		return res, nil
	}
	cat, err := loadCatalog(ctx, p.client, p.orgID)
	if err != nil {
		return res, err
	}
	for code := range codes {
		entry, ok := cat.lookup(code)
		if !ok {
			slog.Error("bwhv: Staffelcode unbekannt", "code", code, "season_id", seasonID)
			continue
		}
		st, err := p.store.upsertStaffel(ctx, Staffel{
			SeasonID: seasonID, Code: code, Name: entry.Name,
			OrgID: p.orgID, SubOrgID: entry.SubOrgID, PeriodID: cat.Period,
		})
		if err != nil {
			return res, err
		}
		changed, err := p.syncSchedule(ctx, *st, entry.ClassID)
		if err != nil {
			slog.Error("bwhv: Spielplan-Abruf fehlgeschlagen", "code", code, "error", err)
			continue
		}
		res.Staffeln++
		res.GamesChanged += changed
	}
	return res, nil
}

// syncSchedule holt den Spielplan einer Staffel. classID kommt aus dem
// Katalog des Laufs — ein eigener Katalog-Abruf je Staffel wäre der teuerste
// Teil des Ganzen (siehe catalog.go).
func (p *Poller) syncSchedule(ctx context.Context, st Staffel, classID string) (int, error) {
	sch, err := p.client.FetchSchedule(ctx, st.OrgID, st.SubOrgID, st.PeriodID, classID)
	if err != nil {
		return 0, err
	}
	return p.store.SaveSchedule(ctx, st.ID, sch)
}

// PollStaffel ruft Spielplan und offene Berichte einer Staffel ab.
//
// cat darf nil sein; dann wird der Katalog für diesen einen Lauf geladen. Ein
// Aufrufer, der mehrere Staffeln abklappert, reicht denselben Katalog durch.
func (p *Poller) PollStaffel(ctx context.Context, seasonID int, st Staffel, cat *catalog) (PollResult, error) {
	var res PollResult
	if cat == nil {
		var err error
		if cat, err = loadCatalog(ctx, p.client, p.orgID); err != nil {
			return res, err
		}
	}
	entry, ok := cat.lookup(st.Code)
	if !ok {
		slog.Error("bwhv: Staffel im Katalog verschwunden", "code", st.Code)
		return res, nil
	}
	changed, err := p.syncSchedule(ctx, st, entry.ClassID)
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
	if len(due) == 0 {
		return total, nil
	}
	cat, err := loadCatalog(ctx, p.client, p.orgID)
	if err != nil {
		return total, err
	}
	for _, d := range due {
		res, err := p.PollStaffel(ctx, seasonID, d.Staffel, cat)
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

// SyncAndPollAll richtet ein und holt nach: Katalog, Spielpläne UND alle
// offenen Berichte jeder zugeordneten Staffel.
//
// Eigener Weg neben SyncStaffeln (Katalog + Spielplan), weil der tägliche
// 06:00-Lauf bewusst keine PDFs zieht — die kommen über das Spieltags-Fenster.
// Genau das greift bei einer Einrichtung mitten in der Saison aber nicht: das
// Nachholfenster reicht CatchUpDays zurück, alles davor bliebe für immer
// ungeholt. Der manuelle Anstoß meint "hol alles", und das tut er hier.
func (p *Poller) SyncAndPollAll(ctx context.Context, seasonID int) (PollResult, error) {
	total, err := p.SyncStaffeln(ctx, seasonID)
	if err != nil {
		return total, err
	}
	staffeln, err := p.store.ListStaffeln(ctx, seasonID)
	if err != nil {
		return total, err
	}
	if len(staffeln) == 0 {
		return total, nil
	}
	cat, err := loadCatalog(ctx, p.client, p.orgID)
	if err != nil {
		return total, err
	}
	for _, st := range staffeln {
		res, err := p.PollStaffel(ctx, seasonID, st, cat)
		if err != nil {
			slog.Error("bwhv: Nachhol-Abruf fehlgeschlagen", "staffel", st.Code, "error", err)
			continue
		}
		// Staffeln/GamesChanged hat SyncStaffeln schon gezählt; hier zählen
		// nur die Berichte dazu, sonst stünde alles doppelt in der Bilanz.
		total.ReportsFetched += res.ReportsFetched
		total.ReportsParsed += res.ReportsParsed
		total.ReportsFailed += res.ReportsFailed
		total.PlayersLinked += res.PlayersLinked
	}
	return total, nil
}
