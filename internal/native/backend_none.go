//go:build !linux || (!gtk && !qt) || (gtk && qt)

package native

import (
	"fmt"
	"os"
)

func runUI() int {
	fmt.Fprintln(os.Stderr, "GoDucky: no native UI backend enabled.")
	fmt.Fprintln(os.Stderr, "Build with -tags gtk (GTK4) or -tags qt (Qt 6) on Linux.")
	return 1
}
