//go:build darwin && cocoa

package native

func runUI() int {
	return RunCocoa()
}