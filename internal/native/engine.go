package native

import (
	"github.com/go-ducky/gui/internal/guiservice"
)

func Run() int {
	return runUI()
}

type Event struct {
	Name string
	Data any
}

type Engine struct {
	*guiservice.Service
	events chan Event
}

func New() *Engine {
	e := &Engine{
		Service: guiservice.NewService(),
		events:  make(chan Event, 256),
	}
	e.Service.SetEventSink(func(name string, data any) {
		select {
		case e.events <- Event{Name: name, Data: data}:
		default:

		}
	})
	return e
}

func (e *Engine) Events() <-chan Event {
	return e.events
}

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

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

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

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
