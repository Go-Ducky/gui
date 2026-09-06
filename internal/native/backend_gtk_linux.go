//go:build linux && gtk && !qt

package native

// runUI launches the GTK4 frontend.
func runUI() int {
	return RunGTK()
}
