package main

import (
	"context"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type Runtime interface {
	// Returns the the compiler's exit code, program's exit code, stdout, stderr, and an error
	// if one occurred.
	CompileAndRun(ctx context.Context, logger *zap.Logger, code string) (int, int, string, string, error)
}

type PythonRuntime struct {
	logger         *zap.Logger
	maxMemoryBytes uint64
}

var _ Runtime = &PythonRuntime{}

func NewPythonRuntime(logger *zap.Logger, maxMemoryBytes uint64) *PythonRuntime {
	return &PythonRuntime{
		logger:         logger,
		maxMemoryBytes: maxMemoryBytes,
	}
}

func (r *PythonRuntime) CompileAndRun(ctx context.Context, logger *zap.Logger, code string) (int, int, string, string, error) {
	cmd := exec.CommandContext(ctx, "python3", "-c", code)

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return -1, -1, "", "", err
	}

	// Set memory limit immediately after starting (platform-specific implementation)
	if r.maxMemoryBytes > 0 {
		if err := setMemoryLimitOnPid(cmd.Process.Pid, r.maxMemoryBytes); err != nil {
			logger.Warn("failed to set memory limit", zap.Error(err))
		}
	}

	logger.Info("running program")

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Exited() {
				logger.Info("program exited with non-zero exit code", zap.Int("code", exitErr.ExitCode()))
				return 0, exitErr.ExitCode(), stdout.String(), stderr.String(), nil
			}
			if ctx.Err() == context.DeadlineExceeded {
				logger.Info("program timed out", zap.Error(ctx.Err()))
				return -1, -1, stdout.String(), stderr.String(), ctx.Err()
			}
			logger.Error("program was terminated", zap.Error(err))
			return -1, -1, stdout.String(), stderr.String(), err
		}
		logger.Error("program failed to run", zap.Error(err))
		return 0, -1, stdout.String(), stderr.String(), err
	}

	logger.Info("program ran successfully")

	return 0, 0, stdout.String(), stderr.String(), nil
}
