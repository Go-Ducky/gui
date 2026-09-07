//go:build windows && win32

package native

func runUI() int {
	return RunWin32()
}