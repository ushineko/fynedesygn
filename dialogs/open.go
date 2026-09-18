package dialogs

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// ErrNoPath is returned by OpenPath for an empty path.
var ErrNoPath = errors.New("there is nothing to open yet")

/*
OpenPath hands a file or directory to the desktop's opener and lets go.

It is deliberately the smallest possible subprocess: the desktop decides what
a file is worth opening in, because it already knows and this library has no
business having an opinion. The path is a single argument, so no shell parses
it. The child is started under context.Background on purpose: tying it to a
context of the window's would mean closing the window took the user's text
editor with it. A machine without an opener gets an error naming it, which is
still enough to open the path by hand.
*/
func OpenPath(path string) error {
	if path == "" {
		return ErrNoPath
	}
	name, args := opener(path)
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // one argument, no shell; the opener is the platform's
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ask the desktop to open %s: %w", path, err)
	}
	// Release the child; its exit is not this program's business.
	go func() { _ = cmd.Wait() }()
	return nil
}

// Opener names the platform's desktop opener.
func Opener() string {
	name, _ := opener("")
	return name
}
