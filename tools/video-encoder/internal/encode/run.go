package encode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Run startet ffmpeg mit args und meldet den Fortschritt (0..1) über
// onProgress. total ist die Quelldauer; bei 0 bleibt der Fortschritt aus. Ein
// Exit-Code ≠ 0 wird zu einem Fehler mit den letzten ffmpeg-Fehlerzeilen.
// onProgress wird aus der aufrufenden Goroutine gerufen, ungedrosselt (~2/s,
// ffmpeg-Takt) — die UI drosselt selbst.
func Run(ctx context.Context, ffmpegPath string, args []string, total time.Duration, onProgress func(float64)) error {
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	hideWindow(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr := &tailBuffer{max: 8 << 10}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg konnte nicht gestartet werden: %w", err)
	}
	parseProgress(stdout, total, onProgress)
	err = cmd.Wait()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		msg := lastLines(stderr.String(), 3)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("ffmpeg ist mit Exit-Code %d abgebrochen: %s", exitErr.ExitCode(), msg)
		}
		return fmt.Errorf("ffmpeg: %w: %s", err, msg)
	}
	onProgress(1)
	return nil
}

// parseProgress liest die `-progress`-Blöcke von ffmpeg und meldet out_time
// relativ zu total. ffmpeg schreibt out_time_us und (historisch falsch
// benannt, ebenfalls in µs) out_time_ms; ältere Versionen nur letzteres.
// Gelesen wird bis EOF, auch ohne total — sonst blockiert ffmpeg an der vollen Pipe.
func parseProgress(r io.Reader, total time.Duration, onProgress func(float64)) {
	sc := bufio.NewScanner(r)
	seenUS := false
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if !ok || total <= 0 {
			continue
		}
		switch k {
		case "out_time_us":
			seenUS = true
		case "out_time_ms":
			if seenUS {
				continue
			}
		default:
			continue
		}
		us, err := strconv.ParseInt(v, 10, 64)
		if err != nil || us < 0 { // „N/A" vor dem ersten Frame
			continue
		}
		f := float64(us) / float64(total.Microseconds())
		if f > 1 {
			f = 1
		}
		onProgress(f)
	}
	_, _ = io.Copy(io.Discard, r)
}

// tailBuffer behält die letzten max Bytes von allem, was hineingeschrieben wird.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = t.buf[over:]
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}
