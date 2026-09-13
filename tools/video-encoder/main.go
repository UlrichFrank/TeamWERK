// Command teamwerk-video-encoder ist das Offline-Encoding-Tool für TeamWERK:
// Spielvideo auswählen, auf diesem Rechner in das Zielformat encodieren
// (H.264/yuv420p, 720p, 4-s-Keyframes, AAC) und über den bestehenden
// tus-Upload hochladen. Der Server packt die Datei danach nur noch um.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// version setzt der macOS-Release-Build per -ldflags "-X main.version=…"; der
// Windows-Build bekommt sie über die Fyne-Metadaten (fyne package --app-version).
var version string

const (
	appID         = "org.team-stuttgart.teamwerk.videoencoder"
	defaultServer = "https://teamwerk.team-stuttgart.org"
	prefServer    = "server"
	prefEmail     = "email"
)

func main() {
	a := app.NewWithID(appID)
	w := a.NewWindow("TeamWERK Video-Encoder")
	u := newUI(a, w)
	w.SetContent(u.build())
	w.SetOnDropped(u.onDropped)
	w.SetCloseIntercept(u.onClose)
	a.Lifecycle().SetOnStopped(u.stopRun)
	w.SetMainMenu(fyne.NewMainMenu(fyne.NewMenu("Hilfe",
		fyne.NewMenuItem("Über TeamWERK Video-Encoder …", u.showAbout))))
	w.Resize(fyne.NewSize(640, 640))
	w.Show()
	u.warmUp()
	u.showLogin("")
	a.Run()
}

func appVersion(a fyne.App) string {
	if version != "" {
		return version
	}
	if v := a.Metadata().Version; v != "" && v != "0.0.0" {
		return v
	}
	return "Entwicklerversion"
}
