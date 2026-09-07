//go:build (!linux && !(windows && win32) && !(darwin && cocoa)) || (linux && !gtk && !qt) || (gtk && qt)

package native

import (
	"fmt"
	"os"
)

func runUI() int {
	fmt.Fprintln(os.Stderr, "GoDucky: no native UI backend enabled.")
	fmt.Fprintln(os.Stderr, "Linux: build with -tags gtk (GTK4) or -tags qt (Qt 6).")
	fmt.Fprintln(os.Stderr, "Windows: build with -tags win32.")
	fmt.Fprintln(os.Stderr, "macOS: build with -tags cocoa.")
	return 1
}