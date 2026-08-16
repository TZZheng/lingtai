package tui

import (
	"errors"
	"fmt"

	"github.com/anthropics/lingtai-tui/internal/config"
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

// controlOwner performs one selected-Agent lifecycle operation on a contained,
// real Agent directory. It is request-aware so owners can route through the
// shared containment gate in control.go and resolve the canonical read-only
// runtime only after containment accepts the target. sleepAgent and suspendAgent
// are the existing shared owners in control.go.
type controlOwner func(req ControlRequest) error

// controlOwners is the command registry. Only `refresh` carries the one optional
// preset arg; every other registered command rejects any argument.
var controlOwners = map[string]controlOwner{
	"sleep": func(req ControlRequest) error {
		return controlAgentSignal(req.Project, req.Agent, "sleep", sleepAgent)
	},
	"suspend": func(req ControlRequest) error {
		return controlAgentSignal(req.Project, req.Agent, "suspend", suspendAgent)
	},
	"cpr": func(req ControlRequest) error {
		return requestCPRHeadlessWithDeps(req.Project, req.Agent, defaultClearContextDeps)
	},
	"clear": func(req ControlRequest) error {
		return controlAgentSignal(req.Project, req.Agent, "clear", func(agentDir string) error {
			globalDir, err := config.GlobalDirPath()
			if err != nil {
				return fmt.Errorf("resolve global TUI dir for clear: %w", err)
			}
			lingtaiCmd := config.LingtaiCmd(globalDir)
			_, err = requestClearContextHeadlessWithDeps(lingtaiCmd, agentDir, defaultClearWaitConfig, defaultClearContextDeps)
			return err
		})
	},
	"refresh": func(req ControlRequest) error {
		return controlAgentSignal(req.Project, req.Agent, "refresh", func(agentDir string) error {
			globalDir, err := config.GlobalDirPath()
			if err != nil {
				return fmt.Errorf("resolve global TUI dir for refresh: %w", err)
			}
			lingtaiCmd := config.LingtaiCmd(globalDir)
			return hardRefreshDirWithArgs(lingtaiCmd, agentDir, req.Arg)
		})
	},
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

// DispatchControl routes a parsed request through the command registry. Every
// owner reaches the shared containment-validation gate in control.go before any
// runtime resolution or lifecycle mutation, rejecting the reserved human target
// with ErrInvalidControlAgent. Only refresh may carry the one optional arg; any
// other registered command rejects it.
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
	if err := owner(req); err != nil {
		return ControlEnvelope{}, err
	}
	return ControlEnvelope{Command: req.Command, Agent: req.Agent, Status: "signaled"}, nil
}
