//go:build cgo && !wasm && !js && !android && !ios && !mobile && !test_web_driver

package widgets

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/require"
)

// fakeDesktopWindow has the field viewportOf reads, the way Fyne's desktop
// window does.
type fakeDesktopWindow struct {
	fyne.Window
	viewport *glfw.Window
}

// The GLFW window is read from the field by name and type, and anything
// without it -- the test driver's window -- is nil, not a crash.
func TestTheGLFWWindowIsReadFromTheViewportField(t *testing.T) {
	view := &glfw.Window{}
	require.Same(t, view, viewportOf(&fakeDesktopWindow{viewport: view}))

	test.NewApp()
	win := test.NewWindow(nil)
	defer win.Close()
	require.Nil(t, viewportOf(win), "a window with no viewport field is not one")
	watchLeave(win.Canvas()) // nothing to register, and nothing to break
}

/*
Fyne's desktop window keeps its GLFW window in an unexported field called
viewport, of type *glfw.Window, and registers no cursor-enter callback of its
own (quirk 42).

watchLeave depends on both. A Fyne that renamed the field would leave tips
appearing after the pointer left the window again, silently, because
viewportOf returns nil rather than failing; one that started registering the
callback would make this workaround redundant. Either shows up here.
*/
func TestFyneStillKeepsItsGLFWWindowWhereATipLooks(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", "fyne.io/fyne/v2").Output()
	require.NoError(t, err)
	dir := filepath.Join(strings.TrimSpace(string(out)), "internal", "driver", "glfw")

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(dir, "window_desktop.go"), nil, 0)
	require.NoError(t, err)
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "window" {
			return true
		}
		for _, field := range spec.Type.(*ast.StructType).Fields.List {
			star, ok := field.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Window" {
				continue
			}
			for _, name := range field.Names {
				if name.Name == "viewport" {
					found = true
				}
			}
		}
		return false
	})
	require.True(t, found, "Fyne's desktop window no longer has a viewport *glfw.Window field")

	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotContains(t, string(src), "SetCursorEnterCallback",
			"Fyne registers a cursor-enter callback now (%s): quirk 42 may be fixed upstream", filepath.Base(path))
	}
}
