//go:build windows

package shell

import (
	"os"
	"os/exec"

	fd "github.com/ushineko/fynedesygn"
)

// Restart starts a fresh copy of this program on the same arguments and quits
// this one, so a setting that Fyne fixes at window creation (the interface
// scale) takes effect. Windows has no exec(2), so the new process gets a new
// pid. OnStop runs first so a pending save is written.
func (s *Shell) Restart() {
	s.Stop()
	exe, err := os.Executable()
	if err != nil {
		s.App.Quit()
		return
	}
	cmd := exec.Command(exe, os.Args[1:]...) //nolint:gosec // our own binary, our own argv
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		s.Flash("Could not restart: "+err.Error()+". Close and reopen the window.", fd.StatusWarn)
		return
	}
	s.App.Quit()
}
