// Copyright (C) 2024 The Dagu Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dagu-org/dagu/internal/config"
	"github.com/dagu-org/dagu/internal/dag"
	"github.com/dagu-org/dagu/internal/persistence"

	"github.com/dagu-org/dagu/internal/client"
	"github.com/dagu-org/dagu/internal/dag/scheduler"
	"github.com/dagu-org/dagu/internal/util"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// cmdTest is a helper struct to test commands.
// It contains the arguments to the command and the expected output.
type cmdTest struct {
	args        []string
	expectedOut []string
}

// testRunCommand is a helper function to test a command.
func testRunCommand(t *testing.T, cmd *cobra.Command, test cmdTest) {
	t.Helper()

	root := &cobra.Command{Use: "root"}
	root.AddCommand(cmd)

	// Set arguments.
	root.SetArgs(test.args)

	// Run the command

	// TODO: Fix thet test after update the logging code so that it can be
	err := root.Execute()
	require.NoError(t, err)

	// configured to write to a buffer.
	// _ = withSpool(t, func() {
	// 	err := root.Execute()
	// 	require.NoError(t, err)
	// })
	//
	// Check if the expected output is present in the standard output.
	// for _, s := range test.expectedOut {
	// 	require.Contains(t, out, s)
	// }
}

// withSpool temporarily buffers the standard output and returns it as a string.
/*
func withSpool(t *testing.T, testFunction func()) string {
	t.Helper()

	origStdout := os.Stdout

	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	log.SetOutput(w)

	defer func() {
		os.Stdout = origStdout
		log.SetOutput(origStdout)
		_ = w.Close()
	}()

	testFunction()

	os.Stdout = origStdout
	_ = w.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	out := buf.String()

	t.Cleanup(func() {
		t.Log(out)
	})

	return out
}
*/

func testDAGFile(name string) string {
	return filepath.Join(
		filepath.Join(util.MustGetwd(), "testdata"),
		name,
	)
}

const (
	waitForStatusTimeout = time.Millisecond * 5000
	tick                 = time.Millisecond * 50
)

// testStatusEventual tests the status of a DAG to be the expected status.
func testStatusEventual(t *testing.T, e client.Client, dagFile string, expected scheduler.Status) {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err)

	workflow, err := dag.Load(cfg.BaseConfig, dagFile, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		status, err := e.GetCurrentStatus(workflow)
		require.NoError(t, err)
		return expected == status.Status
	}, waitForStatusTimeout, tick)
}

// testLastStatusEventual tests the last status of a DAG to be the expected status.
func testLastStatusEventual(
	t *testing.T,
	hs persistence.HistoryStore,
	dg string,
	expected scheduler.Status,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		status := hs.ReadStatusRecent(dg, 1)
		if len(status) < 1 {
			return false
		}
		return expected == status[0].Status.Status
	}, waitForStatusTimeout, tick)
}
