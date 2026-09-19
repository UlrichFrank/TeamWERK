package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"

	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/client"
	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/encode"
	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/ffmpegbin"
	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/pick"
	"github.com/teamstuttgart/teamwerk/tools/video-encoder/internal/progress"
)

const noGame = "Freier Titel"

// optAppend/optReplace sind die Optionen der Hinzufügen-oder-Ersetzen-Auswahl
// (video-upload-anhaengen-oder-ersetzen), sichtbar nur wenn zum gewählten
// Spiel schon (ein) Video(s) existieren. "Hinzufügen" ist der sichere
// Default und entspricht dem bisherigen Verhalten unverändert.
const (
	optAppend  = "Hinzufügen (weiteres Video, z. B. 2. Halbzeit)"
	optReplace = "Ersetzen (vorhandenes Video wird gelöscht)"
)

var videoExtensions = []string{".mp4", ".mov", ".m4v", ".mkv", ".avi", ".mts", ".m2ts", ".mpg", ".mpeg", ".wmv", ".webm", ".3gp"}

// meta sind die Formularangaben eines Laufs. Vergleichbar, damit ein zweiter
// Versuch erkennt, ob die schon angelegte Video-Zeile noch passt.
type meta struct {
	title, description string
	teamID, gameID     int
}

// job hält, was ein fehlgeschlagener Lauf hinterlässt, damit ein zweiter
// Versuch nicht von vorn beginnt: das fertig encodierte Ergebnis (ein ganzes
// Spiel braucht leicht eine halbe Stunde) und — sobald angelegt — die
// Video-Zeile samt tus-Session. Während eines Laufs schreibt nur dessen
// Goroutine hinein; die UI liest erst wieder, wenn er beendet ist.
//
// replace/replaceIDs (video-upload-anhaengen-oder-ersetzen) stehen bewusst
// NICHT in meta: sie werden bei jedem start() frisch aus der aktuellen
// UI-Auswahl gesetzt, unabhängig davon, ob meta unverändert ist — ein
// Moduswechsel zwischen zwei Versuchen desselben Uploads soll keinen
// Job-Reset (neue Video-Zeile, neue tus-Session) auslösen.
type job struct {
	src        string
	encoded    string // "" solange nicht fertig encodiert
	account    string // Server + E-Mail der Anmeldung, unter der videoID angelegt wurde
	meta       meta
	videoID    int
	location   string
	replace    bool  // "Ersetzen" gewählt: nach Upload-Erfolg werden replaceIDs gelöscht
	replaceIDs []int // Video-IDs, die bei replace=true nach dem Upload gelöscht werden
}

type ui struct {
	app      fyne.App
	win      fyne.Window
	cacheDir string

	client       *client.Client
	email        string
	seasonID     int
	teamIDs      map[string]int
	gameIDs      map[string]int
	videosByGame map[int]client.ExistingVideo // gameID → bereits vorhandene(s) Video(s), für die Ersetzen-Auswahl
	eligible     client.Eligible              // Mannschaften + Spiele, für die der Nutzer hochladen darf (video-upload-eligible-teams); reine Vorauswahl, Server entscheidet verbindlich

	srcPath      string
	fileLabel    *widget.Label
	chooseBtn    *widget.Button
	teamSelect   *widget.Select
	gameSelect   *widget.Select
	titleEntry   *widget.Entry
	descEntry    *widget.Entry
	replaceHint  *widget.Label
	replaceGroup *widget.RadioGroup
	replaceBox   *fyne.Container // umschließt Hinweis+Auswahl, nur sichtbar bei vorhandenem Video zum gewählten Spiel
	startBtn     *widget.Button
	cancelBtn    *widget.Button
	bar          *widget.ProgressBar
	status       *widget.Label
	link         *widget.Hyperlink
	account      *widget.Label

	busy   bool
	cancel context.CancelFunc
	done   chan struct{}
	job    *job
}

func newUI(a fyne.App, w fyne.Window) *ui {
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	u := &ui{app: a, win: w, cacheDir: filepath.Join(cache, "teamwerk-video-encoder")}
	// Reste früherer Läufe (abgebrochene Encodes) aufräumen — über einen
	// Neustart hinweg wird nicht fortgesetzt.
	_ = os.RemoveAll(u.workDir())
	return u
}

func (u *ui) workDir() string   { return filepath.Join(u.cacheDir, "work") }
func (u *ui) ffmpegDir() string { return filepath.Join(u.cacheDir, "bin") }

// warmUp entpackt das eingebettete ffmpeg schon beim Start im Hintergrund,
// damit der erste Encode nicht darauf wartet. Fehler meldet später der Lauf.
func (u *ui) warmUp() {
	go func() { _, _ = ffmpegbin.Path(u.ffmpegDir()) }()
}

func (u *ui) build() fyne.CanvasObject {
	u.fileLabel = widget.NewLabel("Noch keine Datei gewählt — Datei auswählen oder ins Fenster ziehen.")
	u.fileLabel.Wrapping = fyne.TextWrapWord
	u.chooseBtn = widget.NewButtonWithIcon("Datei auswählen …", theme.FolderOpenIcon(), u.chooseFile)

	// Hinzufügen-oder-Ersetzen-Auswahl (video-upload-anhaengen-oder-ersetzen):
	// nur sichtbar, wenn zum gewählten Spiel schon (ein) Video(s) existieren
	// (onGameChanged steuert Show/Hide). Ein eigener Container statt eines
	// Form-Items, weil widget.Form in dieser Fyne-Version keine ausblendbaren
	// Zeilen kennt — ein normaler Container lässt sich dagegen komplett
	// verbergen, inklusive seiner Beschriftung. MUSS vor u.gameSelect gebaut
	// werden: dessen SetSelected(noGame) unten löst synchron onGameChanged
	// aus, das u.replaceGroup bereits braucht (sonst Nil-Pointer-Panic beim
	// Start).
	u.replaceHint = widget.NewLabel("")
	u.replaceHint.Wrapping = fyne.TextWrapWord
	u.replaceGroup = widget.NewRadioGroup([]string{optAppend, optReplace}, nil)
	u.replaceGroup.Horizontal = true
	u.replaceGroup.SetSelected(optAppend)
	u.replaceBox = container.NewVBox(u.replaceHint, u.replaceGroup)
	u.replaceBox.Hide()

	u.teamSelect = widget.NewSelect(nil, u.onTeamChanged)
	u.teamSelect.PlaceHolder = "Mannschaft auswählen …"
	u.gameSelect = widget.NewSelect([]string{noGame}, u.onGameChanged)
	u.gameSelect.SetSelected(noGame)
	u.titleEntry = widget.NewEntry()
	u.titleEntry.SetPlaceHolder("z. B. Heimspiel gegen TV Musterstadt")
	u.descEntry = widget.NewMultiLineEntry()
	u.descEntry.SetPlaceHolder("optional")
	u.descEntry.SetMinRowsVisible(3)
	hint := widget.NewLabel("Ein bereits gespieltes Spiel aus der Liste wählen (jüngstes zuerst) oder „Freier Titel“ lassen und unten einen Titel angeben. Bei einem Spiel ohne eigenen Titel bildet TeamWERK ihn aus Datum und Gegner.")
	hint.Wrapping = fyne.TextWrapWord
	hint.Importance = widget.LowImportance
	form := widget.NewForm(
		widget.NewFormItem("Mannschaft", u.teamSelect),
		widget.NewFormItem("Spiel", u.gameSelect),
		widget.NewFormItem("Titel", u.titleEntry),
		widget.NewFormItem("Beschreibung", u.descEntry),
	)

	u.startBtn = widget.NewButtonWithIcon("Encodieren und hochladen", theme.UploadIcon(), u.start)
	u.startBtn.Importance = widget.HighImportance
	u.cancelBtn = widget.NewButtonWithIcon("Abbrechen", theme.CancelIcon(), func() {
		if u.cancel != nil {
			u.cancel()
		}
	})
	u.cancelBtn.Disable()
	u.bar = widget.NewProgressBar()
	u.bar.Hide()
	u.status = widget.NewLabel("")
	u.status.Wrapping = fyne.TextWrapWord
	u.link = widget.NewHyperlink("", nil)
	u.link.Hide()
	u.account = widget.NewLabel("Nicht angemeldet")
	u.account.Importance = widget.LowImportance

	content := container.NewVBox(
		widget.NewCard("1. Videodatei", "", container.NewVBox(u.fileLabel, container.NewHBox(u.chooseBtn))),
		widget.NewCard("2. Angaben", "", container.NewVBox(form, hint, u.replaceBox)),
		widget.NewCard("3. Encodieren und hochladen", "",
			container.NewVBox(container.NewHBox(u.startBtn, u.cancelBtn), u.bar, u.status, u.link)),
	)
	return container.NewBorder(nil, container.NewPadded(u.account), nil, nil, container.NewVScroll(content))
}

// --- Anmeldung ---------------------------------------------------------------

func (u *ui) showLogin(message string) {
	prefs := u.app.Preferences()
	server := widget.NewEntry()
	server.SetText(prefs.StringWithFallback(prefServer, defaultServer))
	email := widget.NewEntry()
	email.SetText(prefs.String(prefEmail))
	pass := widget.NewPasswordEntry()

	var items []*widget.FormItem
	if message != "" {
		l := widget.NewLabel(message)
		l.Wrapping = fyne.TextWrapWord
		l.Importance = widget.DangerImportance
		items = append(items, widget.NewFormItem("", l))
	}
	items = append(items,
		widget.NewFormItem("Server", server),
		widget.NewFormItem("E-Mail", email),
		widget.NewFormItem("Passwort", pass),
	)
	dismiss := "Abbrechen"
	if u.client == nil {
		dismiss = "Beenden"
	}
	d := dialog.NewForm("Bei TeamWERK anmelden", "Anmelden", dismiss, items, func(ok bool) {
		if !ok {
			if u.client == nil {
				u.app.Quit()
			}
			return
		}
		u.login(server.Text, email.Text, pass.Text)
	}, u.win)
	d.Resize(fyne.NewSize(480, 320))
	d.Show()
	if email.Text != "" {
		u.win.Canvas().Focus(pass)
	} else {
		u.win.Canvas().Focus(email)
	}
}

func (u *ui) login(server, email, password string) {
	email = strings.TrimSpace(email)
	u.status.SetText("Anmeldung …")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var (
			seasonID int
			eligible client.Eligible
		)
		c, err := client.New(server, nil)
		if err == nil {
			err = c.Login(ctx, email, password)
		}
		if err == nil {
			seasonID, err = c.ActiveSeasonID(ctx)
		}
		if err == nil {
			eligible, err = c.Eligible(ctx)
		}
		fyne.Do(func() {
			u.status.SetText("")
			if err != nil {
				u.showLogin(userMessage(err))
				return
			}
			u.app.Preferences().SetString(prefServer, c.BaseURL())
			u.app.Preferences().SetString(prefEmail, email)
			u.client, u.email, u.seasonID = c, email, seasonID
			u.eligible = eligible
			u.account.SetText("Angemeldet als " + email + " bei " + c.BaseURL())
			u.setTeams(pick.Teams(eligible))
		})
	}()
}

func (u *ui) accountKey() string { return u.client.BaseURL() + "|" + u.email }

func (u *ui) setTeams(teams []client.EligibleTeam) {
	prev := u.teamSelect.Selected
	u.teamIDs = map[string]int{}
	var labels []string
	for _, t := range teams {
		label := t.Name
		if _, dup := u.teamIDs[label]; dup {
			label = fmt.Sprintf("%s (#%d)", t.Name, t.ID)
		}
		u.teamIDs[label] = t.ID
		labels = append(labels, label)
	}
	u.teamSelect.SetOptions(labels)
	if len(labels) == 0 {
		// Die Liste kommt aus der Upload-Berechtigung (Rolle ODER Video-Dienst),
		// nicht aus der Kader-Zugehörigkeit — ohne Hinweis stünde hier eine
		// leere Liste, obwohl der Nutzer z. B. als Spieler durchaus Teams hat.
		u.status.SetText("Für dieses Konto ist weder eine Trainer-/Vorstands-Berechtigung " +
			"noch ein Video-Dienst hinterlegt. Bitte den Vorstand bitten, dich als Trainer " +
			"im Kader oder für den Video-Dienst des Spiels einzutragen.")
	}
	if _, ok := u.teamIDs[prev]; ok {
		u.teamSelect.SetSelected(prev)
	} else {
		u.teamSelect.ClearSelected()
	}
}

func (u *ui) onTeamChanged(label string) {
	u.gameIDs = nil
	u.videosByGame = nil
	u.gameSelect.SetOptions([]string{noGame})
	u.gameSelect.SetSelected(noGame)
	u.replaceBox.Hide()
	teamID, ok := u.teamIDs[label]
	if !ok || u.client == nil {
		return
	}
	// Spiele kommen aus der Berechtigungs-Antwort (pick), nicht aus
	// /api/games — dessen Sichtbarkeit kennt nur Kader-Zugehörigkeit und
	// ließe ein Dienst-Spiel bei einer fremden Mannschaft verschwinden.
	games := pick.Games(u.eligible, teamID, time.Now().Format("2006-01-02"))
	freeTitle := pick.AllowFreeTitle(u.eligible, teamID)
	c := u.client
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Video-Info je Spiel ist eine Zugabe, kein Kernbestandteil der Auswahl:
		// schlägt der Abruf fehl (oder ist die Liste für ein reines Dienst-Team
		// leer, weil /api/videos der Video-Sichtbarkeit folgt), bleibt die
		// Liste einfach ohne Hinweistext nutzbar.
		videosByGame, _ := c.VideosByGame(ctx, teamID)
		fyne.Do(func() {
			if u.teamSelect.Selected != label {
				return // inzwischen andere Mannschaft gewählt
			}
			u.videosByGame = videosByGame
			u.gameIDs = map[string]int{}
			var opts []string
			// „Freier Titel" (Upload ohne Spielbezug) nur, wenn der Server ihn
			// für diese Mannschaft zulässt (Rollen-Pfad). Für ein reines
			// Dienst-Team ist die Option gar nicht da statt erst im Klick mit
			// 403 zu scheitern.
			if freeTitle {
				opts = append(opts, noGame)
			}
			for _, g := range games {
				l := g.Label()
				if v, ok := videosByGame[g.ID]; ok {
					l += " · " + v.Describe(formatSize)
				}
				if _, dup := u.gameIDs[l]; dup {
					l = fmt.Sprintf("%s (#%d)", l, g.ID)
				}
				u.gameIDs[l] = g.ID
				opts = append(opts, l)
			}
			u.gameSelect.SetOptions(opts)
			switch {
			case len(opts) == 0:
				u.gameSelect.ClearSelected()
				u.status.SetText("Dein Video-Dienst bei dieser Mannschaft betrifft ein Spiel, das noch nicht " +
					"stattgefunden hat — der Upload ist erst nach dem Spiel möglich.")
			case freeTitle:
				u.gameSelect.SetSelected(noGame)
				if len(opts) == 1 {
					u.status.SetText("Für diese Mannschaft gibt es in der aktiven Saison noch kein gespieltes Spiel — " +
						"„Freier Titel“ verwenden.")
				}
			default:
				u.gameSelect.SetSelected(opts[0])
			}
		})
	}()
}

// onGameChanged zeigt die Hinzufügen-oder-Ersetzen-Auswahl nur, wenn zum
// gewählten Spiel laut videosByGame schon (ein) Video(s) existieren
// (video-upload-anhaengen-oder-ersetzen). Bei "Freier Titel" oder einem Spiel
// ohne vorhandenes Video bleibt sie verborgen und die Auswahl auf "Hinzufügen"
// zurückgesetzt — das ist das unveränderte bisherige Verhalten.
func (u *ui) onGameChanged(label string) {
	gameID := u.gameIDs[label]
	existing, ok := u.videosByGame[gameID]
	if gameID == 0 || !ok || len(existing.IDs) == 0 {
		u.replaceGroup.SetSelected(optAppend)
		u.replaceBox.Hide()
		return
	}
	u.replaceHint.SetText(existing.Describe(formatSize) + " — wie soll mit dem neuen Video verfahren werden?")
	u.replaceBox.Show()
}

// --- Dateiauswahl ------------------------------------------------------------

// chooseFile öffnet den nativen Dateidialog des Systems (zenity: Win32 bzw.
// AppleScript unter macOS). Er blockiert, deshalb in einer Goroutine. Steht er
// nicht zur Verfügung, springt Fynes eigener Dialog ein.
func (u *ui) chooseFile() {
	patterns := make([]string, len(videoExtensions))
	for i, ext := range videoExtensions {
		patterns[i] = "*" + ext
	}
	go func() {
		path, err := zenity.SelectFile(
			zenity.Title("Spielvideo auswählen"),
			zenity.FileFilters{{Name: "Videodateien", Patterns: patterns, CaseFold: true}},
		)
		fyne.Do(func() {
			switch {
			case err == nil:
				u.setSource(path)
			case errors.Is(err, zenity.ErrCanceled):
			default:
				u.chooseFileFallback()
			}
		})
	}()
}

func (u *ui) chooseFileFallback() {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		_ = r.Close()
		u.setSource(r.URI().Path())
	}, u.win)
	d.SetFilter(storage.NewExtensionFileFilter(videoExtensions))
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}

func (u *ui) onDropped(_ fyne.Position, uris []fyne.URI) {
	for _, uri := range uris {
		if uri.Scheme() == "file" {
			u.setSource(uri.Path())
			return
		}
	}
}

func (u *ui) setSource(path string) {
	if u.busy {
		return
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		u.status.SetText("Das ist keine lesbare Datei: " + path)
		return
	}
	if u.job != nil && u.job.src != path {
		u.discardJob()
	}
	u.srcPath = path
	u.fileLabel.SetText(fmt.Sprintf("%s (%s)", filepath.Base(path), formatSize(fi.Size())))
	u.status.SetText("")
	u.link.Hide()
	u.bar.Hide()
}

func (u *ui) discardJob() {
	if u.job != nil && u.job.encoded != "" {
		_ = os.Remove(u.job.encoded)
	}
	u.job = nil
}

// --- Lauf --------------------------------------------------------------------

func (u *ui) start() {
	if u.busy {
		return
	}
	if u.client == nil {
		u.showLogin("")
		return
	}
	m := meta{
		title:       strings.TrimSpace(u.titleEntry.Text),
		description: strings.TrimSpace(u.descEntry.Text),
		teamID:      u.teamIDs[u.teamSelect.Selected],
		gameID:      u.gameIDs[u.gameSelect.Selected],
	}
	switch {
	case u.srcPath == "":
		u.status.SetText("Bitte zuerst eine Videodatei auswählen.")
		return
	case m.teamID == 0:
		u.status.SetText("Bitte eine Mannschaft auswählen.")
		return
	case m.gameID == 0 && !pick.AllowFreeTitle(u.eligible, m.teamID):
		// Reines Dienst-Team: der Server nimmt ohne game_id keinen Upload an.
		u.status.SetText("Für diese Mannschaft muss ein Spiel ausgewählt werden.")
		return
	case m.title == "" && m.gameID == 0:
		u.status.SetText("Bitte einen Titel angeben oder ein Spiel auswählen.")
		return
	}

	replace, replaceIDs := u.pendingReplace(m.gameID)
	if !replace {
		u.startJob(m, false, nil)
		return
	}
	// Löschen ist unwiderruflich — eine zusätzliche Bestätigung direkt vor dem
	// Start, weil die Radio-Auswahl selbst (bei der Spiel-Auswahl getroffen,
	// oft Minuten vor dem Klick auf "Encodieren und hochladen") dafür allein
	// nicht genug Gewicht hat (video-upload-anhaengen-oder-ersetzen).
	body := "Das vorhandene Video zu diesem Spiel wird nach erfolgreichem Hochladen unwiderruflich gelöscht."
	if len(replaceIDs) > 1 {
		body = fmt.Sprintf("Die %d vorhandenen Videos zu diesem Spiel werden nach erfolgreichem Hochladen unwiderruflich gelöscht.", len(replaceIDs))
	}
	dialog.ShowConfirm("Vorhandenes Video ersetzen?", body, func(ok bool) {
		if ok {
			u.startJob(m, true, replaceIDs)
		}
	}, u.win)
}

// pendingReplace liefert, ob "Ersetzen" gewählt ist und — falls ja — welche
// Video-IDs danach gelöscht werden sollen. gameID == 0 ("Freier Titel") oder
// kein vorhandenes Video für dieses Spiel bedeuten immer "Hinzufügen"
// (video-upload-anhaengen-oder-ersetzen: die Auswahl gilt bewusst nur für
// Spiel-gebundene Videos, siehe design.md).
func (u *ui) pendingReplace(gameID int) (bool, []int) {
	if gameID == 0 || u.replaceGroup.Selected != optReplace {
		return false, nil
	}
	existing, ok := u.videosByGame[gameID]
	if !ok || len(existing.IDs) == 0 {
		return false, nil
	}
	return true, append([]int(nil), existing.IDs...)
}

func (u *ui) startJob(m meta, replace bool, replaceIDs []int) {
	if u.job == nil {
		u.job = &job{src: u.srcPath}
	}
	j := u.job
	// Eine schon angelegte Video-Zeile gilt nur für dieselben Angaben und dieselbe
	// Anmeldung; sonst eine neue anlegen (das Encodat bleibt verwendbar).
	if j.videoID != 0 && (j.meta != m || j.account != u.accountKey()) {
		j.videoID, j.location = 0, ""
	}
	j.meta, j.account = m, u.accountKey()
	j.replace, j.replaceIDs = replace, replaceIDs

	ctx, cancel := context.WithCancel(context.Background())
	u.cancel, u.done = cancel, make(chan struct{})
	u.setBusy(true)
	c, seasonID, done := u.client, u.seasonID, u.done
	go func() {
		defer close(done)
		id, warn, err := u.run(ctx, c, seasonID, j)
		fyne.Do(func() { u.finish(c, id, warn, err) })
	}()
}

// run encodiert (sofern noch nicht geschehen), legt die Video-Zeile an und lädt
// hoch. Läuft in einer eigenen Goroutine; UI-Änderungen nur über fyne.Do. Der
// zweite Rückgabewert ist ein optionaler Warnhinweis (video-speicherplatz
// bereits gelöscht? Nein: video-upload-anhaengen-oder-ersetzen) — gesetzt,
// wenn der Upload zwar erfolgreich war, das Löschen eines Ersetzen-Kandidaten
// aber fehlschlug; das ist ausdrücklich KEIN Fehler (err bleibt nil).
func (u *ui) run(ctx context.Context, c *client.Client, seasonID int, j *job) (int, string, error) {
	if j.encoded == "" || !fileExists(j.encoded) {
		u.showStep("ffmpeg wird vorbereitet (beim ersten Start einige Sekunden) …")
		ffmpeg, err := ffmpegbin.Path(u.ffmpegDir())
		if err != nil {
			return 0, "", err
		}
		src, err := encode.Probe(ctx, ffmpeg, j.src)
		if err != nil {
			return 0, "", fmt.Errorf("die Datei lässt sich nicht als Video lesen: %w", err)
		}
		if err := os.MkdirAll(u.workDir(), 0o755); err != nil {
			return 0, "", err
		}
		stem := strings.TrimSuffix(filepath.Base(j.src), filepath.Ext(j.src))
		out := filepath.Join(u.workDir(), stem+"-720p.mp4")
		part := filepath.Join(u.workDir(), stem+"-720p.part.mp4")
		report := u.progressReporter("Schritt 1 von 2: Encodieren")
		u.showProgress("Schritt 1 von 2: Encodieren", 0, "")
		if err := encode.Run(ctx, ffmpeg, encode.BuildArgs(j.src, part, src), src.Duration, func(f float64) { report(f, f) }); err != nil {
			_ = os.Remove(part)
			return 0, "", err
		}
		if err := os.Rename(part, out); err != nil {
			return 0, "", err
		}
		j.encoded = out
	}

	fi, err := os.Stat(j.encoded)
	if err != nil {
		return 0, "", err
	}
	if j.videoID == 0 {
		u.showStep("Video wird in TeamWERK angelegt …")
		nv := client.NewVideo{Title: j.meta.title, TeamID: j.meta.teamID, SeasonID: seasonID, SizeBytes: fi.Size()}
		if j.meta.description != "" {
			nv.Description = &j.meta.description
		}
		if j.meta.gameID != 0 {
			nv.GameID = &j.meta.gameID
		}
		if j.videoID, err = c.CreateVideo(ctx, nv); err != nil {
			return 0, "", err
		}
	}

	report := u.progressReporter("Schritt 2 von 2: Hochladen")
	first := int64(-1)
	err = c.Upload(ctx, client.Upload{
		Path:      j.encoded,
		VideoID:   j.videoID,
		Location:  j.location,
		OnCreated: func(loc string) { j.location = loc },
		OnProgress: func(sent, total int64) {
			if total <= 0 {
				return
			}
			if first < 0 {
				first = sent // fortgesetzter Upload: Restzeit nur aus dem Neuen schätzen
			}
			eta := 0.0
			if total > first {
				eta = float64(sent-first) / float64(total-first)
			}
			report(float64(sent)/float64(total), eta)
		},
	})
	if err != nil {
		return 0, "", err
	}
	_ = os.Remove(j.encoded)

	// "Ersetzen": erst jetzt, nach gesichertem Upload-Erfolg, die zuvor
	// vorhandenen Videos löschen (nie davor — design.md "Löschen erst NACH
	// erfolgreichem Upload"). Ein Löschfehler wird als Warnhinweis
	// zurückgegeben, bricht den Erfolg des Uploads aber nicht.
	var warn string
	if j.replace {
		u.showStep("Vorheriges Video wird gelöscht …")
		for _, oldID := range j.replaceIDs {
			if oldID == j.videoID {
				continue // Sicherheitsnetz: niemals das gerade hochgeladene Video löschen
			}
			if delErr := c.DeleteVideo(ctx, oldID); delErr != nil {
				warn = "Das neue Video wurde hochgeladen, das vorherige konnte aber nicht automatisch " +
					"gelöscht werden (" + userMessage(delErr) + ") — bitte manuell in TeamWERK löschen."
				break
			}
		}
	}
	return j.videoID, warn, nil
}

// progressReporter liefert einen gedrosselten Fortschritts-Callback (max. 4×/s
// in die UI). f ist der angezeigte Anteil, etaFrac der für die Restzeit.
func (u *ui) progressReporter(phase string) func(f, etaFrac float64) {
	th := progress.NewThrottle(250 * time.Millisecond)
	started := time.Now()
	return func(f, etaFrac float64) {
		if !th.Allow(f >= 1) {
			return
		}
		rem := progress.FormatRemaining(progress.Remaining(time.Since(started), etaFrac))
		fyne.Do(func() { u.setProgress(phase, f, rem) })
	}
}

func (u *ui) showStep(text string) {
	fyne.Do(func() {
		u.bar.Hide()
		u.status.SetText(text)
	})
}

func (u *ui) showProgress(phase string, f float64, rem string) {
	fyne.Do(func() { u.setProgress(phase, f, rem) })
}

func (u *ui) setProgress(phase string, f float64, rem string) {
	u.bar.Show()
	u.bar.SetValue(f)
	text := fmt.Sprintf("%s – %d %%", phase, int(f*100))
	if rem != "" {
		text += " – " + rem
	}
	u.status.SetText(text)
}

func (u *ui) finish(c *client.Client, videoID int, deleteWarn string, err error) {
	u.setBusy(false)
	u.cancel, u.done = nil, nil
	switch {
	case err == nil:
		u.discardJob()
		u.srcPath = ""
		u.fileLabel.SetText("Noch keine Datei gewählt — Datei auswählen oder ins Fenster ziehen.")
		u.bar.SetValue(1)
		// Radio auf den sicheren Default zurücksetzen: die Auswahl gilt sonst
		// weiter für den nächsten Upload zum selben Spiel, obwohl "Ersetzen"
		// bereits ausgeführt wurde (video-upload-anhaengen-oder-ersetzen).
		u.replaceGroup.SetSelected(optAppend)
		msg := "Fertig! Das Video wird jetzt auf dem Server verarbeitet und erscheint danach in TeamWERK. " +
			"Alle Berechtigten bekommen eine Benachrichtigung."
		if deleteWarn != "" {
			msg += "\n\n" + deleteWarn
		}
		u.status.SetText(msg)
		if link, perr := url.Parse(c.VideoURL(videoID)); perr == nil {
			u.link.SetText("Video in TeamWERK öffnen")
			u.link.SetURL(link)
			u.link.Show()
		}
	case errors.Is(err, context.Canceled):
		u.bar.Hide()
		u.status.SetText("Abgebrochen.")
	case errors.Is(err, client.ErrSessionExpired):
		u.status.SetText("Die Anmeldung ist abgelaufen. Nach der erneuten Anmeldung wieder auf " +
			"„Encodieren und hochladen“ klicken — es geht dort weiter, wo es aufgehört hat.")
		u.showLogin(userMessage(err))
	default:
		u.status.SetText("Fehler: " + userMessage(err) +
			"\nEin erneuter Klick auf „Encodieren und hochladen“ setzt beim letzten fertigen Schritt fort.")
		dialog.ShowError(errors.New(userMessage(err)), u.win)
	}
}

func (u *ui) setBusy(b bool) {
	u.busy = b
	for _, w := range []fyne.Disableable{u.chooseBtn, u.teamSelect, u.gameSelect, u.titleEntry, u.descEntry, u.replaceGroup, u.startBtn} {
		if b {
			w.Disable()
		} else {
			w.Enable()
		}
	}
	if b {
		u.cancelBtn.Enable()
		u.link.Hide()
	} else {
		u.cancelBtn.Disable()
	}
}

// onClose fragt während eines Laufs nach und bricht ihn vor dem Schließen ab —
// sonst liefe ffmpeg als verwaister Prozess weiter.
func (u *ui) onClose() {
	if !u.busy {
		u.win.Close()
		return
	}
	dialog.ShowConfirm("Vorgang läuft", "Encodieren bzw. Hochladen abbrechen und das Programm beenden?", func(ok bool) {
		if !ok {
			return
		}
		go func() {
			u.stopRun()
			fyne.Do(u.win.Close)
		}()
	}, u.win)
}

// stopRun bricht einen laufenden Vorgang ab und wartet kurz, bis ffmpeg beendet
// ist. Auch beim Beenden über das App-Menü (Cmd+Q) aufgerufen.
func (u *ui) stopRun() {
	cancel, done := u.cancel, u.done
	if cancel == nil {
		return
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (u *ui) showAbout() {
	text := fmt.Sprintf("TeamWERK Video-Encoder %s\n\n"+
		"Encodiert Spielvideos auf diesem Rechner (H.264, 720p) und lädt sie in TeamWERK hoch.\n\n%s",
		appVersion(u.app), ffmpegbin.Notice())
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(l)
	scroll.SetMinSize(fyne.NewSize(540, 380))
	dialog.NewCustom("Über / Lizenzen", "Schließen", scroll, u.win).Show()
}

// userMessage macht aus Fehlern eine Meldung für Menschen.
func userMessage(err error) string {
	var uerr *url.Error
	if errors.As(err, &uerr) && !errors.Is(err, context.Canceled) {
		return "Der Server ist nicht erreichbar. Bitte die Internetverbindung und die Server-Adresse prüfen."
	}
	msg := err.Error()
	if msg == "" {
		return "Unbekannter Fehler"
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

func formatSize(n int64) string {
	const mb = 1 << 20
	if n >= 1<<30 {
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	}
	return fmt.Sprintf("%.1f MB", float64(n)/mb)
}
