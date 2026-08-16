package tui

import (
	"errors"
	"fmt"
)

// ControlRequest is one parsed selected-Agent headless control invocation,
// shared between the CLI parser in tui/main.go and the dispatch layer.
type ControlRequest struct {
	Project string
	Agent   string
	Command string
	Arg     string
}

// ControlEnvelope is the stable structured stdout emitted for a dispatched
// command. It never carries private agent identity.
type ControlEnvelope struct {
	Command string `json:"command"`
	Agent   string `json:"agent"`
	Status  string `json:"status"`
}

// controlOwner performs one selected-Agent lifecycle signal on a contained,
// real Agent directory. sleepAgent and suspendAgent are the existing shared
// owners in control.go; future cpr/clear/refresh owners register here when
// implemented.
type controlOwner func(agentDir string) error

// controlOwners is the command registry. A registered command with a nil owner
// is recognized by the parser and dispatch layer but not yet implemented. Only
// `refresh` carries the one optional preset arg; every other registered command
// rejects any argument.
var controlOwners = map[string]controlOwner{
	"sleep":   sleepAgent,
	"suspend": suspendAgent,
	"cpr":     nil,
	"clear":   nil,
	"refresh": nil,
}

var (
	// ErrInvalidControlCommand means the requested command is not in the
	// registered control set.
	ErrInvalidControlCommand = errors.New("invalid control command")
	// ErrInvalidControlArgs means a recognized command carried an argument the
	// registry does not accept. Only refresh accepts the one optional preset
	// arg.
	ErrInvalidControlArgs = errors.New("invalid control command arguments")
	// ErrControlNotImplemented means a recognized control command has no
	// registered owner yet.
	ErrControlNotImplemented = errors.New("control command not implemented")
)

// DispatchControl routes a parsed request through the command registry.
// Implemented owners (sleep/suspend) delegate to the shared containment-
// validated signal facade in control.go; recognized-but-unimplemented commands
// fail with ErrControlNotImplemented and produce no side effects. Only refresh
// may carry the one optional arg; any other registered command rejects it.
func DispatchControl(req ControlRequest) (ControlEnvelope, error) {
	owner, ok := controlOwners[req.Command]
	if !ok {
		return ControlEnvelope{}, fmt.Errorf("%w: %q", ErrInvalidControlCommand, req.Command)
	}
	if req.Arg != "" && req.Command != "refresh" {
		return ControlEnvelope{}, fmt.Errorf("%w: command %q does not accept an argument", ErrInvalidControlArgs, req.Command)
	}
	if owner == nil {
		return ControlEnvelope{}, fmt.Errorf("%w: %q", ErrControlNotImplemented, req.Command)
	}
	if err := controlAgentSignal(req.Project, req.Agent, req.Command, owner); err != nil {
		return ControlEnvelope{}, err
	}
	return ControlEnvelope{Command: req.Command, Agent: req.Agent, Status: "signaled"}, nil
}
