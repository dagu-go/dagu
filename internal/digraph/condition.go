package digraph

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/dagu-org/dagu/internal/cmdutil"
)

var ErrConditionNotMet = fmt.Errorf("condition was not met")

// Condition contains a condition and the expected value.
// Conditions are evaluated and compared to the expected value.
// The condition can be a command substitution or an environment variable.
// The expected value must be a string without any substitutions.
type Condition struct {
	Command   string `json:"Command,omitempty"`   // Command to evaluate
	Condition string `json:"Condition,omitempty"` // Condition to evaluate
	Expected  string `json:"Expected,omitempty"`  // Expected value
}

// eval evaluates the condition and returns the actual value.
// It returns an error if the evaluation failed or the condition is invalid.
func (c Condition) eval(ctx context.Context) (bool, error) {
	switch {
	case c.Condition != "":
		return c.evalCondition(ctx)

	case c.Command != "":
		return c.evalCommand(ctx)

	default:
		return false, fmt.Errorf("invalid condition: Condition=%s", c.Condition)
	}
}

func (c Condition) evalCommand(ctx context.Context) (bool, error) {
	var commandToRun string
	if IsStepContext(ctx) {
		command, err := GetStepContext(ctx).EvalString(c.Command)
		if err != nil {
			return false, err
		}
		commandToRun = command
	} else if IsContext(ctx) {
		command, err := GetContext(ctx).EvalString(c.Command)
		if err != nil {
			return false, err
		}
		commandToRun = command
	} else {
		command, err := cmdutil.EvalString(c.Command)
		if err != nil {
			return false, err
		}
		commandToRun = command
	}

	shell := cmdutil.GetShellCommand("")
	if shell == "" {
		// Run the command directly
		cmd := exec.CommandContext(ctx, commandToRun)
		_, err := cmd.Output()
		if err != nil {
			return false, fmt.Errorf("%w: %s", ErrConditionNotMet, err)
		}
		return true, nil
	}

	// Run the command through a shell
	cmd := exec.CommandContext(ctx, shell, "-c", commandToRun)
	_, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrConditionNotMet, err)
	}
	return true, nil
}

func (c Condition) evalCondition(ctx context.Context) (bool, error) {
	if IsStepContext(ctx) {
		evaluatedVal, err := GetStepContext(ctx).EvalString(c.Condition)
		if err != nil {
			return false, err
		}
		return c.Expected == evaluatedVal, nil
	}

	evaluatedVal, err := GetContext(ctx).EvalString(c.Condition)
	if err != nil {
		return false, err
	}

	return c.Expected == evaluatedVal, nil
}

func (c Condition) String() string {
	return fmt.Sprintf("Condition=%s Expected=%s", c.Condition, c.Expected)
}

// evalCondition evaluates a single condition and checks the result.
// It returns an error if the condition was not met.
func evalCondition(ctx context.Context, c Condition) error {
	matched, err := c.eval(ctx)
	if err != nil {
		if errors.Is(err, ErrConditionNotMet) {
			return err
		}
		return fmt.Errorf("failed to evaluate condition: Condition=%s Error=%v", c.Condition, err)
	}

	if !matched {
		return fmt.Errorf("%w: Condition=%s Expected=%s", ErrConditionNotMet, c.Condition, c.Expected)
	}

	// Condition was met
	return nil
}

// EvalConditions evaluates a list of conditions and checks the results.
// It returns an error if any of the conditions were not met.
func EvalConditions(ctx context.Context, cond []Condition) error {
	for _, c := range cond {
		if err := evalCondition(ctx, c); err != nil {
			return err
		}
	}

	return nil
}
