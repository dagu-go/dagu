// Copyright (C) 2024 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package scheduler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/dagu-org/dagu/internal/cmdutil"
	"github.com/dagu-org/dagu/internal/digraph"
	"github.com/dagu-org/dagu/internal/digraph/scheduler"
	"github.com/dagu-org/dagu/internal/fileutil"
	"github.com/dagu-org/dagu/internal/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestScheduler(t *testing.T) {
	t.Parallel()

	t.Run("SequentialStepsSuccess", func(t *testing.T) {
		sc := setup(t, withMaxActiveRuns(1))

		// 1 -> 2 -> 3
		graph := sc.newGraph(t,
			successStep("1"),
			successStep("2", "1"),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 3)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
	})
	t.Run("SequentialStepsWithFailure", func(t *testing.T) {
		sc := setup(t, withMaxActiveRuns(1))

		// 1 -> 2 -> 3 -> 4
		graph := sc.newGraph(t,
			successStep("1"),
			successStep("2", "1"),
			failStep("3", "2"),
			successStep("4", "3"),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		// 1, 2, 3 should be executed and 4 should be canceled because 3 failed
		result.AssertDoneCount(t, 3)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusError)
		result.AssertNodeStatus(t, "4", scheduler.NodeStatusCancel)
	})
	t.Run("ParallelSteps", func(t *testing.T) {
		sc := setup(t, withMaxActiveRuns(3))

		// 1,2,3
		graph := sc.newGraph(t,
			successStep("1"),
			successStep("2"),
			successStep("3"),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 3)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
	})
	t.Run("ParallelStepsWithFailure", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 3 -> 4, 2 (fail)
		graph := sc.newGraph(t,
			successStep("1"),
			failStep("2"),
			successStep("3", "1"),
			successStep("4", "3"),
		)

		result := graph.Schedule(t, scheduler.StatusError)
		result.AssertDoneCount(t, 4)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusError)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "4", scheduler.NodeStatusSuccess)
	})
	t.Run("ContinueOnFailure", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 2 (fail) -> 3
		graph := sc.newGraph(t,
			successStep("1"),
			newStep("2",
				withDepends("1"),
				withCommand("false"),
				withContinueOnFailure(),
			),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		// 1, 2, 3 should be executed even though 2 failed
		result.AssertDoneCount(t, 3)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusError)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
	})
	t.Run("ContinueOnSkip", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 2 (skip) -> 3
		graph := sc.newGraph(t,
			successStep("1"),
			newStep("2",
				withDepends("1"),
				withCommand("false"),
				withPrecondition("`echo 1`", "0"),
				withContinueOnSkipped(),
			),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		// 1, 2,
		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSkipped)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
	})
	t.Run("CancelSchedule", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 2 (cancel when running) -> 3 (should not be executed)
		graph := sc.newGraph(t,
			successStep("1"),
			newStep("2", withDepends("1"), withCommand("sleep 100")),
			failStep("3", "2"),
		)

		go func() {
			time.Sleep(time.Millisecond * 300) // wait for step 2 to start
			graph.Cancel(t)
		}()

		result := graph.Schedule(t, scheduler.StatusCancel)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusCancel)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusNone)
	})
	t.Run("Timeout", func(t *testing.T) {
		sc := setup(t, withTimeout(time.Second*2))

		// 1 -> 2 (timeout) -> 3 (should not be executed)
		graph := sc.newGraph(t,
			newStep("1", withCommand("sleep 1")),
			newStep("2", withCommand("sleep 10"), withDepends("1")),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		// 1 should be executed and 2 should be canceled because of timeout
		// 3 should not be executed and should be canceled
		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusCancel)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusCancel)
	})
	t.Run("RetryPolicyFail", func(t *testing.T) {
		const file = "flag_test_retry_fail"

		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1",
				withCommand(fmt.Sprintf("%s %s", testScript, file)),
				withRetryPolicy(2, 0),
			),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		result.AssertDoneCount(t, 3) // 1, 2(retry)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusError)

		node := result.Node(t, "1")
		require.Equal(t, 2, node.State().RetryCount) // 2 retry
	})
	t.Run("RetryPolicySuccess", func(t *testing.T) {
		file := filepath.Join(
			os.TempDir(), fmt.Sprintf("flag_test_retry_success_%s", uuid.Must(uuid.NewRandom()).String()),
		)

		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1",
				withCommand(fmt.Sprintf("%s %s", testScript, file)),
				withRetryPolicy(1, time.Millisecond*500),
			),
		)

		go func() {
			// Create file for successful retry
			time.Sleep(time.Millisecond * 300) // wait for step 1 to start

			// Create file during the retry interval
			f, err := os.Create(file)
			require.NoError(t, err)
			defer f.Close()

			t.Cleanup(func() {
				_ = os.Remove(file)
			})
		}()

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 2) // 1, 2(retry and success)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
	})
	t.Run("PreconditionMatch", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 2 (precondition match) -> 3
		graph := sc.newGraph(t,
			successStep("1"),
			newStep("2", withCommand("echo 2"), withPrecondition("`echo 1`", "1")),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 3)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSuccess)
	})
	t.Run("PreconditionNotMatch", func(t *testing.T) {
		sc := setup(t)

		// 1 -> 2 (precondition not match) -> 3
		graph := sc.newGraph(t,
			successStep("1"),
			newStep("2", withCommand("echo 2"), withPrecondition("`echo 1`", "0")),
			successStep("3", "2"),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 1) // only 1 should

		// 1 should be executed and 2, 3 should be skipped
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSkipped)
		result.AssertNodeStatus(t, "3", scheduler.NodeStatusSkipped)
	})
	t.Run("OnExitHandler", func(t *testing.T) {
		sc := setup(t, withOnExit(successStep("onExit")))

		graph := sc.newGraph(t, successStep("1"))

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "onExit", scheduler.NodeStatusSuccess)
	})
	t.Run("OnExitHandlerFail", func(t *testing.T) {
		sc := setup(t, withOnExit(failStep("onExit")))

		graph := sc.newGraph(t, successStep("1"))

		// Overall status should be error because onExit failed
		result := graph.Schedule(t, scheduler.StatusError)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "onExit", scheduler.NodeStatusError)
	})
	t.Run("OnCancelHandler", func(t *testing.T) {
		sc := setup(t, withOnCancel(successStep("onCancel")))

		graph := sc.newGraph(t,
			newStep("1", withCommand("sleep 10")),
		)

		go func() {
			time.Sleep(time.Millisecond * 100) // wait for step 1 to start
			graph.Signal(syscall.SIGTERM)
		}()

		result := graph.Schedule(t, scheduler.StatusCancel)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusCancel)
		result.AssertNodeStatus(t, "onCancel", scheduler.NodeStatusSuccess)
	})
	t.Run("OnSuccessHandler", func(t *testing.T) {
		sc := setup(t, withOnSuccess(successStep("onSuccess")))

		graph := sc.newGraph(t, successStep("1"))

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "onSuccess", scheduler.NodeStatusSuccess)
	})
	t.Run("OnFailureHandler", func(t *testing.T) {
		sc := setup(t, withOnFailure(successStep("onFailure")))

		graph := sc.newGraph(t, failStep("1"))

		result := graph.Schedule(t, scheduler.StatusError)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusError)
		result.AssertNodeStatus(t, "onFailure", scheduler.NodeStatusSuccess)
	})
	t.Run("CancelOnSignal", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1", withCommand("sleep 10")),
		)

		go func() {
			time.Sleep(time.Millisecond * 100) // wait for step 1 to start
			graph.Signal(syscall.SIGTERM)
		}()

		result := graph.Schedule(t, scheduler.StatusCancel)

		result.AssertDoneCount(t, 1)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusCancel)
	})
	t.Run("Repeat", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1",
				withCommand("sleep 1"),
				withRepeatPolicy(true, time.Millisecond*500),
			),
		)

		go func() {
			time.Sleep(time.Millisecond * 1750)
			graph.Cancel(t)
		}()

		result := graph.Schedule(t, scheduler.StatusCancel)

		// 1 should be repeated 2 times
		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusCancel)

		node := result.Node(t, "1")
		// done count should be 1 because 2nd execution is canceled
		require.Equal(t, 1, node.State().DoneCount)
	})
	t.Run("RepeatFail", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1",
				withCommand("false"),
				withRepeatPolicy(true, time.Millisecond*300),
			),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		// Done count should be 1 because it failed and not repeated
		result.AssertDoneCount(t, 1)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusError)

		node := result.Node(t, "1")
		require.Equal(t, 1, node.State().DoneCount)
	})
	t.Run("StopRepetitiveTaskGracefully", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1",
				withCommand("sleep 1"),
				withRepeatPolicy(true, time.Millisecond*300),
			),
		)

		done := make(chan struct{})
		go func() {
			time.Sleep(time.Millisecond * 100)
			graph.Signal(syscall.SIGTERM)
			close(done)
		}()

		result := graph.Schedule(t, scheduler.StatusSuccess)
		<-done

		result.AssertDoneCount(t, 1)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
	})
	t.Run("NodeSetupFailure", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1", withWorkingDir("/nonexistent"),
				withScript("echo 1"),
			),
		)

		result := graph.Schedule(t, scheduler.StatusError)

		result.AssertDoneCount(t, 1)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusError)

		require.Contains(t, result.Error.Error(), "failed to setup script")
	})
	t.Run("NodeTeardownFailure", func(t *testing.T) {
		sc := setup(t)

		graph := sc.newGraph(t,
			newStep("1", withCommand("sleep 1")),
		)

		nodes := graph.Nodes()
		go func() {
			time.Sleep(time.Millisecond * 300)
			_ = nodes[0].CloseLog()
		}()

		result := graph.Schedule(t, scheduler.StatusError)

		// file already closed
		require.Error(t, result.Error)

		result.AssertDoneCount(t, 1)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusError)
		require.Contains(t, result.Error.Error(), "file already closed")
	})
	t.Run("OutputVariables", func(t *testing.T) {
		sc := setup(t)

		// 1: echo hello > OUT
		// 2: echo $OUT > RESULT
		graph := sc.newGraph(t,
			newStep("1", withCommand("echo hello"), withOutput("OUT")),
			newStep("2", withCommand("echo $OUT"), withDepends("1"), withOutput("RESULT")),
		)

		result := graph.Schedule(t, scheduler.StatusSuccess)

		result.AssertDoneCount(t, 2)
		result.AssertNodeStatus(t, "1", scheduler.NodeStatusSuccess)
		result.AssertNodeStatus(t, "2", scheduler.NodeStatusSuccess)

		node := result.Node(t, "2")

		// check if RESULT variable is set to "hello"
		output, ok := node.Data().Step.OutputVariables.Load("RESULT")
		require.True(t, ok, "output variable not found")
		require.Equal(t, "RESULT=hello", output, "expected output %q, got %q", "hello", output)
	})
}

func successStep(name string, depends ...string) digraph.Step {
	return newStep(name, withDepends(depends...), withCommand("true"))
}

func failStep(name string, depends ...string) digraph.Step {
	return newStep(name, withDepends(depends...), withCommand("false"))
}

type stepOption func(*digraph.Step)

func withDepends(depends ...string) stepOption {
	return func(step *digraph.Step) {
		step.Depends = depends
	}
}

func withContinueOnFailure() stepOption {
	return func(step *digraph.Step) {
		step.ContinueOn.Failure = true
	}
}

func withContinueOnSkipped() stepOption {
	return func(step *digraph.Step) {
		step.ContinueOn.Skipped = true
	}
}

func withRetryPolicy(limit int, interval time.Duration) stepOption {
	return func(step *digraph.Step) {
		step.RetryPolicy.Limit = limit
		step.RetryPolicy.Interval = interval
	}
}

func withRepeatPolicy(repeat bool, interval time.Duration) stepOption {
	return func(step *digraph.Step) {
		step.RepeatPolicy.Repeat = repeat
		step.RepeatPolicy.Interval = interval
	}
}

func withPrecondition(condition, expected string) stepOption {
	return func(step *digraph.Step) {
		step.Preconditions = []digraph.Condition{
			{
				Condition: condition,
				Expected:  expected,
			},
		}
	}
}

func withScript(script string) stepOption {
	return func(step *digraph.Step) {
		step.Script = script
	}
}

func withWorkingDir(dir string) stepOption {
	return func(step *digraph.Step) {
		step.Dir = dir
	}
}

func withOutput(output string) stepOption {
	return func(step *digraph.Step) {
		step.Output = output
	}
}

func withCommand(command string) stepOption {
	return func(step *digraph.Step) {
		cmd, args, err := cmdutil.SplitCommand(command)
		if err != nil {
			panic(fmt.Errorf("unexpected: %w", err))
		}
		step.CmdWithArgs = command
		step.Command = cmd
		step.Args = args
	}
}

func newStep(name string, opts ...stepOption) digraph.Step {
	step := digraph.Step{Name: name}
	for _, opt := range opts {
		opt(&step)
	}

	return step
}

type testHelper struct {
	test.Helper

	Scheduler *scheduler.Scheduler
	Config    *scheduler.Config
}

type schedulerOption func(*scheduler.Config)

func withTimeout(d time.Duration) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.Timeout = d
	}
}

func withMaxActiveRuns(n int) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.MaxActiveRuns = n
	}
}

func withOnExit(step digraph.Step) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.OnExit = &step
	}
}

func withOnCancel(step digraph.Step) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.OnCancel = &step
	}
}

func withOnSuccess(step digraph.Step) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.OnSuccess = &step
	}
}

func withOnFailure(step digraph.Step) schedulerOption {
	return func(cfg *scheduler.Config) {
		cfg.OnFailure = &step
	}
}

func setup(t *testing.T, opts ...schedulerOption) testHelper {
	t.Helper()

	th := test.Setup(t)

	cfg := &scheduler.Config{
		LogDir: th.Config.Paths.LogDir,
		ReqID:  uuid.Must(uuid.NewRandom()).String(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	sc := scheduler.New(cfg)

	return testHelper{
		Helper:    test.Setup(t),
		Scheduler: sc,
		Config:    cfg,
	}
}

func (th testHelper) newGraph(t *testing.T, steps ...digraph.Step) graphHelper {
	t.Helper()

	graph, err := scheduler.NewExecutionGraph(steps...)
	require.NoError(t, err)

	return graphHelper{
		testHelper:     th,
		ExecutionGraph: graph,
	}
}

type graphHelper struct {
	testHelper
	*scheduler.ExecutionGraph
}

func (gh graphHelper) Schedule(t *testing.T, expectedStatus scheduler.Status) scheduleResult {
	t.Helper()

	ctx := digraph.NewContext(gh.Context, &digraph.DAG{}, nil, nil, gh.Config.ReqID, "logFile")

	var doneNodes []*scheduler.Node
	nodeCompletedChan := make(chan *scheduler.Node)

	done := make(chan struct{})
	go func() {
		for node := range nodeCompletedChan {
			doneNodes = append(doneNodes, node)
		}
		done <- struct{}{}
	}()

	err := gh.Scheduler.Schedule(ctx, gh.ExecutionGraph, nodeCompletedChan)

	close(nodeCompletedChan)

	switch expectedStatus {
	case scheduler.StatusSuccess, scheduler.StatusCancel:
		require.NoError(t, err)

	case scheduler.StatusError:
		require.Error(t, err)

	case scheduler.StatusRunning, scheduler.StatusNone:
		t.Errorf("unexpected status %s", expectedStatus)

	}

	require.Equal(t, expectedStatus.String(), gh.Scheduler.Status(gh.ExecutionGraph).String(),
		"expected status %s, got %s", expectedStatus, gh.Scheduler.Status(gh.ExecutionGraph))

	// wait for items of nodeCompletedChan to be processed
	<-done
	close(done)

	return scheduleResult{
		graphHelper: gh,
		Done:        doneNodes,
		Error:       err,
	}
}

func (gh graphHelper) Signal(sig syscall.Signal) {
	gh.Scheduler.Signal(gh.Context, gh.ExecutionGraph, sig, nil, false)
}

func (gh graphHelper) Cancel(t *testing.T) {
	t.Helper()

	gh.Scheduler.Cancel(gh.Context, gh.ExecutionGraph)
}

type scheduleResult struct {
	graphHelper
	Done  []*scheduler.Node
	Error error
}

func (sr scheduleResult) AssertDoneCount(t *testing.T, expected int) {
	t.Helper()

	require.Len(t, sr.Done, expected, "expected %d done nodes, got %d", expected, len(sr.Done))
}

func (sr scheduleResult) AssertNodeStatus(t *testing.T, stepName string, expected scheduler.NodeStatus) {
	t.Helper()

	var target *scheduler.Node

	nodes := sr.ExecutionGraph.Nodes()
	for _, node := range nodes {
		if node.Data().Step.Name == stepName {
			target = node
		}
	}

	if sr.Config.OnExit != nil && sr.Config.OnExit.Name == stepName {
		target = sr.Scheduler.HandlerNode(digraph.HandlerOnExit)
	}
	if sr.Config.OnSuccess != nil && sr.Config.OnSuccess.Name == stepName {
		target = sr.Scheduler.HandlerNode(digraph.HandlerOnSuccess)
	}
	if sr.Config.OnFailure != nil && sr.Config.OnFailure.Name == stepName {
		target = sr.Scheduler.HandlerNode(digraph.HandlerOnFailure)
	}
	if sr.Config.OnCancel != nil && sr.Config.OnCancel.Name == stepName {
		target = sr.Scheduler.HandlerNode(digraph.HandlerOnCancel)
	}

	if target == nil {
		t.Fatalf("step %s not found", stepName)
	}

	require.Equal(t, expected.String(), target.State().Status.String(), "expected status %q, got %q", expected.String(), target.State().Status.String())
}

func (sr scheduleResult) Node(t *testing.T, stepName string) *scheduler.Node {
	t.Helper()

	nodes := sr.ExecutionGraph.Nodes()
	for _, node := range nodes {
		if node.Data().Step.Name == stepName {
			return node
		}
	}

	if sr.Config.OnExit != nil && sr.Config.OnExit.Name == stepName {
		return sr.Scheduler.HandlerNode(digraph.HandlerOnExit)
	}
	if sr.Config.OnSuccess != nil && sr.Config.OnSuccess.Name == stepName {
		return sr.Scheduler.HandlerNode(digraph.HandlerOnSuccess)
	}
	if sr.Config.OnFailure != nil && sr.Config.OnFailure.Name == stepName {
		return sr.Scheduler.HandlerNode(digraph.HandlerOnFailure)
	}
	if sr.Config.OnCancel != nil && sr.Config.OnCancel.Name == stepName {
		return sr.Scheduler.HandlerNode(digraph.HandlerOnCancel)
	}

	t.Fatalf("step %s not found", stepName)
	return nil
}

// testScript is a shell script that fails if the file with the name of
// the first argument does not exist
var testScript = filepath.Join(fileutil.MustGetwd(), "testdata/testfile.sh")
