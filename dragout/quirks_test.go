package dragout

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fyneDir is the Fyne module this build uses.
func fyneDir(t *testing.T) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", "fyne.io/fyne/v2").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

func fyneSources(t *testing.T, dir string) map[string]string {
	t.Helper()
	srcs := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		srcs[path] = string(b)
		return nil
	})
	require.NoError(t, err)
	return srcs
}

/*
Quirk 44: Fyne cannot start a drag out of its window. Neither it nor GLFW
calls any platform's drag-source API. The day one of them does, this fails,
and the package may be replaceable by Fyne's own.
*/
func TestFyneStillCannotStartADragOutOfTheWindow(t *testing.T) {
	for path, src := range fyneSources(t, fyneDir(t)) {
		for _, api := range []string{"start_drag", "DoDragDrop", "beginDraggingSession", "XdndPosition"} {
			require.NotContains(t, src, api,
				"Fyne calls %s now (%s): quirk 44 may be fixed upstream", api, filepath.Base(path))
		}
	}
}

/*
Quirk 45: Fyne ends a drag only when its own GLFW mouse-button callback
reports the release. The platform's drag takes the pointer, so that release
never comes and release() sends one through the same callback. A Fyne that
ended drags some other way would leave release() doing the wrong thing.
*/
func TestFyneStillEndsADragOnlyFromItsMouseButtonCallback(t *testing.T) {
	dir := filepath.Join(fyneDir(t), "internal", "driver", "glfw")
	desktop, err := os.ReadFile(filepath.Join(dir, "window_desktop.go"))
	require.NoError(t, err)
	require.Contains(t, string(desktop), "SetMouseButtonCallback(w.mouseClicked)")

	ends := 0
	for _, src := range fyneSources(t, dir) {
		ends += strings.Count(src, ".DragEnd()")
	}
	require.Equal(t, 1, ends, "Fyne ends a drag from somewhere other than processMouseClicked now")
}
