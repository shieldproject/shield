package task

import (
	"encoding/json"
	"fmt"
	"strings"

	boshui "github.com/cloudfoundry/bosh-cli/ui"
)

type ReporterImpl struct {
	ui          boshui.UI
	isForEvents bool

	hasOutput bool

	events    []*Event
	lastEvent *Event

	outputRest string
}

func NewReporter(ui boshui.UI, isForEvents bool) *ReporterImpl {
	return &ReporterImpl{ui: ui, isForEvents: isForEvents}
}

func (r ReporterImpl) TaskStarted(id int) {
	r.ui.BeginLinef("Task %d", id)
}

func (r ReporterImpl) TaskFinished(id int, state string) {
	if len(r.events) > 0 {
		start := r.events[0].TimeAsStr()
		end := r.lastEvent.TimeAsStr()
		duration := r.events[0].DurationAsStr(*r.lastEvent)
		r.ui.PrintLinef("\nStarted  %s\nFinished %s\nDuration %s", start, end, duration)
	}

	if r.hasOutput {
		r.ui.PrintLinef("Task %d %s", id, state)
	} else {
		r.ui.EndLinef(". %s", strings.Title(state))
	}
}

func (r *ReporterImpl) TaskOutputChunk(id int, chunk []byte) {
	if !r.hasOutput {
		r.hasOutput = true
		r.ui.BeginLinef("\n")
		if !r.isForEvents {
			r.ui.BeginLinef("\n")
		}
	}

	if r.isForEvents {
		r.outputRest += string(chunk)

		for {
			idx := strings.Index(r.outputRest, "\n")
			if idx == -1 {
				return
			}
			if len(r.outputRest[0:idx]) > 0 {
				r.showEvent(r.outputRest[0:idx])
			}
			r.outputRest = r.outputRest[idx+1:]
		}
	} else {
		r.showChunk(chunk)
	}
}

func (r *ReporterImpl) showEvent(str string) {
	var event Event

	err := json.Unmarshal([]byte(str), &event)
	if err != nil {
		panic(fmt.Sprintf("unmarshal chunk '%s'", str))
	}

	for _, ev := range r.events {
		if ev.IsSame(event) {
			event.StartEvent = ev
			break
		}
	}

	if r.lastEvent != nil && r.lastEvent.IsSame(event) {
		switch {
		case event.State == EventStateStarted:
			// does not make sense

		case event.State == EventStateFinished:
			r.ui.PrintBlock(fmt.Sprintf(" (%s)", event.DurationSinceStartAsStr()))

		case event.State == EventStateFailed:
			r.ui.PrintBlock(fmt.Sprintf(" (%s)", event.DurationSinceStartAsStr()))
			r.ui.PrintErrorBlock(fmt.Sprintf(
				"\n            L Error: %s", event.Data.Error))
		}
	} else {
		if r.lastEvent != nil && event.IsWorthKeeping() {
			if event.Type == EventTypeDeprecation || event.Error != nil {
				// Some spacing around deprecations and errors
				r.ui.PrintBlock("\n")
			}
		}

		prefix := fmt.Sprintf("\n%s | ", event.TimeAsHoursStr())
		desc := event.Stage

		if len(event.Tags) > 0 {
			desc += " " + strings.Join(event.Tags, ", ")
		}

		switch {
		case event.Type == EventTypeDeprecation:
			r.ui.PrintBlock(prefix)
			r.ui.PrintErrorBlock(fmt.Sprintf("Deprecation: %s", event.Message))

		case event.Type == EventTypeWarning:
			r.ui.PrintBlock(prefix)
			r.ui.PrintErrorBlock(fmt.Sprintf("Warning: %s", event.Message))

		case event.State == EventStateStarted:
			r.ui.PrintBlock(prefix)
			r.ui.PrintBlock(fmt.Sprintf("%s: %s", desc, event.Task))

		case event.State == EventStateFinished:
			r.ui.PrintBlock(prefix)
			r.ui.PrintBlock(fmt.Sprintf("%s: %s (%s)",
				desc, event.Task, event.DurationSinceStartAsStr()))

		case event.State == EventStateFailed:
			r.ui.PrintBlock(prefix)
			r.ui.PrintBlock(fmt.Sprintf("%s: %s (%s)",
				desc, event.Task, event.DurationSinceStartAsStr()))
			r.ui.PrintErrorBlock(fmt.Sprintf(
				"\n            L Error: %s", event.Data.Error))

		case event.Error != nil:
			r.ui.PrintBlock(prefix)
			r.ui.PrintErrorBlock(fmt.Sprintf("Error: %s", event.Error.Message))

		default:
			// Skip event
		}
	}

	if event.IsWorthKeeping() {
		r.events = append(r.events, &event)
		r.lastEvent = &event
	}
}

func (r *ReporterImpl) showChunk(bytes []byte) {
	r.ui.PrintBlock(string(bytes))
}
