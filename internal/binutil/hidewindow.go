//go:build !windows

package binutil

import "os/exec"

// HideWindow est un no-op sur les plateformes non-Windows
func HideWindow(cmd *exec.Cmd) {}
