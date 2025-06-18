package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPythonRuntime(t *testing.T) {
	ctx := context.Background()
	is := require.New(t)
	logger, err := zap.NewProductionConfig().Build(zap.WithCaller(false))
	is.NoError(err)
	runtime := NewPythonRuntime(logger)

	snippets := []string{
		"time.sleep(10)",
		"import time\ntime.sleep(10)",
		"print('hello, world')",
	}
	timeouts := []int{
		1,
		1,
		10,
	}
	expected := []bool{
		false,
		false,
		true,
	}

	for i := range snippets {
		code := snippets[i]
		timeout := timeouts[i]
		shouldSucceed := expected[i]

		testCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(timeout))
		defer cancel()

		compilerExitCode, programExitCode, _, _, err := runtime.CompileAndRun(testCtx, logger, code)
		if shouldSucceed {
			is.NoError(err)
			is.Equal(0, compilerExitCode)
			is.Equal(0, programExitCode)
		} else {
			is.True(err != nil || compilerExitCode != 0 || programExitCode != 0)
		}
	}
}
