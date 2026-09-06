//go:build linux && qt && !gtk

package native

// runUI launches the Qt 6 frontend.
func runUI() int {
	return RunQt()
}
