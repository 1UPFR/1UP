package binutil

import (
	"os/exec"
	"syscall"
)

// HideWindow cache la fenetre CMD sur Windows
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
