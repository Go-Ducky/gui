//go:build linux && gtk && !qt

package native

func runUI() int {
	return RunGTK()
}
