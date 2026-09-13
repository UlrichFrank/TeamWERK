package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Der Server nutzt tusd (internal/videos/upload.go). Implementiert ist nur,
// was dieser Upload braucht: Creation, PATCH in Chunks, HEAD zum Fortsetzen.
const (
	tusVersion     = "1.0.0"
	uploadEndpoint = "/api/videos/upload/"
)

// Upload beschreibt einen Datei-Upload an eine angelegte Video-Zeile.
type Upload struct {
	Path    string
	VideoID int
	// Location einer bereits angelegten tus-Session (Fortsetzen nach erneuter
	// Anmeldung); leer = neue Session anlegen.
	Location string
	// OnCreated bekommt die Location einer neu angelegten Session, damit der
	// Aufrufer sie für ein späteres Fortsetzen aufheben kann.
	OnCreated func(location string)
	// OnProgress meldet gesendete/gesamte Bytes — ungedrosselt, pro gelesenem Block.
	OnProgress func(sent, total int64)
}

// Upload überträgt die Datei per tus. Bricht die Verbindung ab oder antwortet
// der Server mit 5xx/409, fragt der Client per HEAD den bestätigten Stand ab
// und setzt dort fort (mit wachsenden Pausen, insgesamt einige Minuten). Ein
// abgelaufener Access-Token wird erneuert und derselbe Chunk wiederholt; ist
// auch der Refresh-Token abgelaufen, endet der Upload mit ErrSessionExpired.
func (c *Client) Upload(ctx context.Context, u Upload) error {
	f, err := os.Open(u.Path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	size := fi.Size()
	progress := u.OnProgress
	if progress == nil {
		progress = func(int64, int64) {}
	}

	loc := u.Location
	var offset int64
	if loc == "" {
		if loc, err = c.createUpload(ctx, size, u.VideoID, filepath.Base(u.Path)); err != nil {
			return err
		}
		if u.OnCreated != nil {
			u.OnCreated(loc)
		}
	} else if offset, err = c.uploadOffset(ctx, loc); err != nil {
		return err
	}
	progress(offset, size)

	failures := 0
	for offset < size {
		n := min(c.chunkSize, size-offset)
		next, err := c.patchChunk(ctx, loc, f, offset, n, size, progress)
		if err == nil {
			offset, failures = next, 0
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !retryable(err) {
			return err
		}
		if failures >= len(c.retryDelays) {
			return fmt.Errorf("Upload nach %d Versuchen abgebrochen: %w", failures, err)
		}
		if err := c.sleep(ctx, c.retryDelays[failures]); err != nil {
			return err
		}
		failures++
		// Der abgebrochene PATCH kann teilweise angekommen sein — der Server
		// weiß, bis wohin. Scheitert auch das, geht es mit dem alten Stand
		// weiter; ein falscher Offset kommt dann als 409 zurück (retrybar).
		if o, herr := c.uploadOffset(ctx, loc); herr == nil {
			offset = o
		} else if !retryable(herr) {
			return herr
		}
	}
	progress(size, size)
	return nil
}

// retryable trennt vorübergehende Fehler (Netzwerk, 5xx, 409 Offset-Konflikt,
// 429) von endgültigen (abgelaufene Sitzung, 4xx, Abbruch).
func retryable(err error) bool {
	if errors.Is(err, ErrSessionExpired) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, ErrMaintenance) {
		return true
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.Status >= 500 || se.Status == http.StatusConflict || se.Status == http.StatusTooManyRequests
	}
	var perr *pathError
	return !errors.As(err, &perr)
}

// pathError markiert Fehler, die kein Wiederholen heilt (fremde Location).
type pathError struct{ err error }

func (e *pathError) Error() string { return e.err.Error() }
func (e *pathError) Unwrap() error { return e.err }

// createUpload legt die tus-Session an. video_id bindet sie serverseitig an die
// vorab angelegte Zeile (preUploadCreate prüft Eigentümer und Status).
func (c *Client) createUpload(ctx context.Context, size int64, videoID int, filename string) (string, error) {
	meta := strings.Join([]string{
		"video_id " + b64(strconv.Itoa(videoID)),
		"filename " + b64(filename),
		"filetype " + b64("video/mp4"),
	}, ",")
	resp, err := c.do(ctx, func() (*http.Request, error) {
		req, err := c.newRequest(ctx, http.MethodPost, uploadEndpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Tus-Resumable", tusVersion)
		req.Header.Set("Upload-Length", strconv.FormatInt(size, 10))
		req.Header.Set("Upload-Metadata", meta)
		return req, nil
	})
	if err != nil {
		return "", err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusCreated {
		return "", statusError("Upload anlegen", resp)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", errors.New("Upload anlegen: Server hat keine Upload-Adresse geliefert")
	}
	u, err := c.resolve(loc)
	if err != nil {
		return "", &pathError{err}
	}
	return u.String(), nil
}

// uploadOffset fragt per HEAD ab, wie viele Bytes der Server bestätigt hat.
func (c *Client) uploadOffset(ctx context.Context, loc string) (int64, error) {
	resp, err := c.do(ctx, func() (*http.Request, error) {
		req, err := c.newRequest(ctx, http.MethodHead, loc, nil)
		if err != nil {
			return nil, &pathError{err}
		}
		req.Header.Set("Tus-Resumable", tusVersion)
		return req, nil
	})
	if err != nil {
		return 0, err
	}
	defer drain(resp)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return parseOffset(resp)
	case http.StatusNotFound, http.StatusGone:
		return 0, errors.New("die Upload-Sitzung existiert auf dem Server nicht mehr — bitte neu hochladen")
	default:
		return 0, statusError("Upload-Stand abfragen", resp)
	}
}

// patchChunk sendet n Bytes ab offset und liefert den neuen Server-Offset.
func (c *Client) patchChunk(ctx context.Context, loc string, f *os.File, offset, n, total int64, progress func(int64, int64)) (int64, error) {
	resp, err := c.do(ctx, func() (*http.Request, error) {
		// Pro Versuch ein frischer Reader: nach einem 401 wird derselbe Chunk
		// von vorn gesendet.
		body := &countingReader{r: io.NewSectionReader(f, offset, n), sent: offset, total: total, fn: progress}
		req, err := c.newRequest(ctx, http.MethodPatch, loc, body)
		if err != nil {
			return nil, &pathError{err}
		}
		req.ContentLength = n
		req.Header.Set("Tus-Resumable", tusVersion)
		req.Header.Set("Upload-Offset", strconv.FormatInt(offset, 10))
		req.Header.Set("Content-Type", "application/offset+octet-stream")
		return req, nil
	})
	if err != nil {
		return 0, err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusNoContent {
		return 0, statusError("Upload", resp)
	}
	return parseOffset(resp)
}

func parseOffset(resp *http.Response) (int64, error) {
	o, err := strconv.ParseInt(resp.Header.Get("Upload-Offset"), 10, 64)
	if err != nil || o < 0 {
		return 0, errors.New("Upload: Server lieferte keinen gültigen Upload-Offset")
	}
	return o, nil
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

type countingReader struct {
	r     io.Reader
	sent  int64
	total int64
	fn    func(sent, total int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.sent += int64(n)
		c.fn(c.sent, c.total)
	}
	return n, err
}
