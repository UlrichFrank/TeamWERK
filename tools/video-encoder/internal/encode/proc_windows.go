//go:build windows

package encode

import (
	"os/exec"
	"syscall"
)

// createNoWindow (CREATE_NO_WINDOW) verhindert, dass ffmpeg als
// Konsolenprogramm aus der GUI-App heraus ein schwarzes Konsolenfenster öffnet.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
