// Package native provides a UI-agnostic engine adapter that lets desktop
// frontends (GTK, Qt, Cocoa, Win32) drive the GoDucky agent directly.
package native

import (
	"github.com/go-ducky/gui/internal/guiservice"
)

// Run launches the compiled-in native UI backend and returns its exit code.
// The actual backend is selected by build tag (see backend_*.go).
func Run() int {
	return runUI()
}

// runUI is implemented per build tag by backend_gtk_linux.go,
// backend_qt_linux.go, and backend_none.go.

// Event is a single UI event pushed from the engine to the frontend.
type Event struct {
	Name string
	Data any
}

// Engine wraps the backend service and forwards its events to a channel that
// native UIs consume on their main loop.
type Engine struct {
	*guiservice.Service
	events chan Event
}

// New builds an engine wired to an unbuffered-ish event channel.
func New() *Engine {
	e := &Engine{
		Service: guiservice.NewService(),
		events:  make(chan Event, 256),
	}
	e.Service.SetEventSink(func(name string, data any) {
		select {
		case e.events <- Event{Name: name, Data: data}:
		default:
			// Drop the oldest event if the UI is saturated.
		}
	})
	return e
}

// Events returns the channel of UI events to consume on the main thread.
func (e *Engine) Events() <-chan Event {
	return e.events
}

// Event names forwarded from the engine (mirrors guiservice constants).
const (
	EvtStream   = guiservice.EvtStream
	EvtToolS    = guiservice.EvtToolStart
	EvtToolE    = guiservice.EvtToolEnd
	EvtStatus   = guiservice.EvtStatus
	EvtComplete = guiservice.EvtComplete
	EvtError    = guiservice.EvtError
	EvtApproval = guiservice.EvtApproval
	EvtOllama   = guiservice.EvtOllamaOp
)

// toString safely extracts a string from event payload maps.
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// indexOf finds needle in hay, falling back if not present.
func indexOf(needle string, hay []string, fallback int) int {
	for i, s := range hay {
		if s == needle {
			return i
		}
	}
	if fallback < len(hay) {
		return fallback
	}
	return 0
}
