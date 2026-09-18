//go:build unix

package shell

import (
	"os"
	"syscall"

	fd "github.com/ushineko/fynedesygn"
)

/*
Restart replaces this process with a fresh one on the same arguments, so a
setting that Fyne fixes at window creation (the interface scale) takes effect
without the user finding the launcher again. OnStop runs first so a pending
save is written and any lock released; exec(2) keeps the pid, so a tray or
launcher watching it sees one program.
*/
func (s *Shell) Restart() {
	if s.opts.OnStop != nil {
		s.opts.OnStop(s)
	}
	exe, err := os.Executable()
	if err != nil {
		s.App.Quit()
		return
	}
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil { //nolint:gosec // our own binary, our own argv
		s.Flash("Could not restart: "+err.Error()+". Close and reopen the window.", fd.StatusWarn)
	}
}
