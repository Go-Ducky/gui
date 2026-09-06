//go:build !linux || (!gtk && !qt) || (gtk && qt)

package native

import (
	"fmt"
	"os"
)

// runUI explains that no native backend was compiled in.
func runUI() int {
	fmt.Fprintln(os.Stderr, "GoDucky: no native UI backend enabled.")
	fmt.Fprintln(os.Stderr, "Build with -tags gtk (GTK4) or -tags qt (Qt 6) on Linux.")
	return 1
}
