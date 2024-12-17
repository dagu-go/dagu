// Copyright (C) 2024 Yota Hamada
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"github.com/dagu-org/dagu/internal/agent"
	"github.com/dagu-org/dagu/internal/client"
	"github.com/dagu-org/dagu/internal/config"
	"github.com/dagu-org/dagu/internal/digraph"
	"github.com/dagu-org/dagu/internal/digraph/scheduler"
	"github.com/dagu-org/dagu/internal/logger"
	"github.com/spf13/cobra"
)

func restartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart /path/to/spec.yaml",
		Short: "Stop the running DAG and restart it",
		Long:  `dagu restart /path/to/spec.yaml`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				log.Fatalf("Configuration load failed: %v", err)
			}

			quiet, err := cmd.Flags().GetBool("quiet")
			if err != nil {
				log.Fatalf("Flag retrieval failed (quiet): %v", err)
			}

			initLogger := logger.NewLogger(logger.NewLoggerArgs{
				Debug:  cfg.Debug,
				Format: cfg.LogFormat,
				Quiet:  quiet,
			})

			// Load the DAG file and stop the DAG if it is running.
			specFilePath := args[0]
			dag, err := digraph.Load(cmd.Context(), cfg.BaseConfig, specFilePath, "")
			if err != nil {
				initLogger.Fatal("DAG load failed", "error", err, "file", args[0])
			}

			dataStore := newDataStores(cfg)
			cli := newClient(cfg, dataStore, initLogger)
			ctx := cmd.Context()

			if err := stopDAGIfRunning(ctx, cli, dag, initLogger); err != nil {
				initLogger.Fatal("DAG stop operation failed",
					"error", err,
					"DAG", dag.Name)
			}

			// Wait for the specified amount of time before restarting.
			waitForRestart(dag.RestartWait, initLogger)

			// Retrieve the parameter of the previous execution.
			params, err := getPreviousExecutionParams(ctx, cli, dag)
			if err != nil {
				initLogger.Fatal("Previous execution parameter retrieval failed",
					"error", err,
					"DAG", dag.Name)
			}

			// Start the DAG with the same parameter.
			// Need to reload the DAG file with the parameter.
			dag, err = digraph.Load(cmd.Context(), cfg.BaseConfig, specFilePath, params)
			if err != nil {
				initLogger.Fatal("DAG reload failed",
					"error", err,
					"file", specFilePath,
					"params", params)
			}

			requestID, err := generateRequestID()
			if err != nil {
				initLogger.Fatal("Request ID generation failed", "error", err)
			}

			logFile, err := logger.OpenLogFile(logger.LogFileConfig{
				Prefix:    "restart_",
				LogDir:    cfg.LogDir,
				DAGLogDir: dag.LogDir,
				DAGName:   dag.Name,
				RequestID: requestID,
			})
			if err != nil {
				initLogger.Fatal("Log file creation failed",
					"error", err,
					"DAG", dag.Name)
			}
			defer logFile.Close()

			agentLogger := logger.NewLogger(logger.NewLoggerArgs{
				Debug:   cfg.Debug,
				Format:  cfg.LogFormat,
				LogFile: logFile,
				Quiet:   quiet,
			})

			agentLogger.Info("DAG restart initiated",
				"DAG", dag.Name,
				"requestID", requestID,
				"logFile", logFile.Name())

			agt := agent.New(
				requestID,
				dag,
				agentLogger,
				filepath.Dir(logFile.Name()),
				logFile.Name(),
				newClient(cfg, dataStore, agentLogger),
				dataStore,
				&agent.Options{Dry: false})

			listenSignals(cmd.Context(), agt)
			if err := agt.Run(cmd.Context()); err != nil {
				agentLogger.Fatal("DAG restart failed",
					"error", err,
					"DAG", dag.Name,
					"requestID", requestID)
			}
		},
	}
	cmd.Flags().BoolP("quiet", "q", false, "suppress output")
	return cmd
}

// stopDAGIfRunning stops the DAG if it is running.
// Otherwise, it does nothing.
func stopDAGIfRunning(ctx context.Context, e client.Client, dag *digraph.DAG, lg logger.Logger) error {
	curStatus, err := e.GetCurrentStatus(ctx, dag)
	if err != nil {
		return err
	}

	if curStatus.Status == scheduler.StatusRunning {
		lg.Infof("Stopping: %s", dag.Name)
		cobra.CheckErr(stopRunningDAG(ctx, e, dag))
	}
	return nil
}

// stopRunningDAG attempts to stop the running DAG
// by sending a stop signal to the agent.
func stopRunningDAG(ctx context.Context, e client.Client, dag *digraph.DAG) error {
	for {
		curStatus, err := e.GetCurrentStatus(ctx, dag)
		if err != nil {
			return err
		}

		// If the DAG is not running, do nothing.
		if curStatus.Status != scheduler.StatusRunning {
			return nil
		}

		if err := e.Stop(ctx, dag); err != nil {
			return err
		}

		time.Sleep(time.Millisecond * 100)
	}
}

// waitForRestart waits for the specified amount of time before restarting
// the DAG.
func waitForRestart(restartWait time.Duration, lg logger.Logger) {
	if restartWait > 0 {
		lg.Info("Waiting for restart", "duration", restartWait)
		time.Sleep(restartWait)
	}
}

func getPreviousExecutionParams(ctx context.Context, e client.Client, dag *digraph.DAG) (string, error) {
	latestStatus, err := e.GetLatestStatus(ctx, dag)
	if err != nil {
		return "", err
	}

	return latestStatus.Params, nil
}
