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

package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenLogFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_log_dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := LogFileConfig{
		Prefix:    "test_",
		LogDir:    tempDir,
		DAGName:   "test_dag",
		RequestID: "12345678",
	}

	file, err := OpenLogFile(config)
	require.NoError(t, err)
	defer file.Close()

	assert.NotNil(t, file)
	assert.True(t, filepath.IsAbs(file.Name()))
	assert.Contains(t, file.Name(), "test_dag")
	assert.Contains(t, file.Name(), "test_")
	assert.Contains(t, file.Name(), "12345678")
}

func TestPrepareLogDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_log_dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		config   LogFileConfig
		expected string
	}{
		{
			name: "Default LogDir",
			config: LogFileConfig{
				LogDir:  tempDir,
				DAGName: "test_dag",
			},
			expected: filepath.Join(tempDir, "test_dag"),
		},
		{
			name: "Custom DAGLogDir",
			config: LogFileConfig{
				LogDir:    tempDir,
				DAGLogDir: filepath.Join(tempDir, "custom"),
				DAGName:   "test_dag",
			},
			expected: filepath.Join(tempDir, "custom", "test_dag"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareLogDirectory(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.DirExists(t, result)
		})
	}
}

func TestGenerateLogFilename(t *testing.T) {
	config := LogFileConfig{
		Prefix:    "test_",
		DAGName:   "test dag",
		RequestID: "12345678",
	}

	filename := generateLogFilename(config)

	assert.Contains(t, filename, "test_")
	assert.Contains(t, filename, "test_dag")
	assert.Contains(t, filename, time.Now().Format("20060102"))
	assert.Contains(t, filename, "12345678")
	assert.Contains(t, filename, ".log")
}

func TestOpenFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_log_dir")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test.log")

	file, err := openFile(filePath)
	require.NoError(t, err)
	defer file.Close()

	assert.NotNil(t, file)
	assert.Equal(t, filePath, file.Name())

	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
