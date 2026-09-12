//go:build !windows

package encode

import "os/exec"

// hideWindow ist nur unter Windows nötig (siehe proc_windows.go).
func hideWindow(*exec.Cmd) {}
