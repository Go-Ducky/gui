//go:build linux && qt && !gtk

package native

func runUI() int {
	return RunQt()
}
