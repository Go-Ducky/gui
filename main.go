// Command goducky is the native desktop app for the GoDucky agent.
// The UI backend (GTK4 or Qt 6) is selected at build time:
//
//	go build -tags gtk .
//	go build -tags qt .
package main

import (
	"os"

	"github.com/go-ducky/gui/internal/native"
)

func main() {
	os.Exit(native.Run())
}
