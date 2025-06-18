package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

func NewServer(
	logger *zap.Logger,
	config *Config,
) http.Handler {
	mux := http.NewServeMux()

	pythonRuntime := NewPythonRuntime(logger)
	requestIdx := atomic.Uint64{}

	mux.Handle("/v1/execute", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, problems, err := decodeValid[ExecuteRequest](r)
		if err != nil {
			writeError(w, err, problems)
			return
		}

		// Execute programs in parallel, up to the maximum number of concurrent
		// programs.
		wg := sync.WaitGroup{}
		results := make([]ProgramResult, len(req.Programs))
		concurrencyLimit := make(chan struct{}, config.MaxConcurrentEvaluations)
		for i := range req.Programs {
			copiedIndex := i
			wg.Add(1)
			go func() {
				concurrencyLimit <- struct{}{}
				defer func() { <-concurrencyLimit }()
				defer wg.Done()
				program := req.Programs[copiedIndex]

				if program.Runtime != "python3" {
					results[copiedIndex] = ProgramResult{
						Success: false,
						Error:   newString("unsupported runtime"),
					}
					return
				}

				if program.TimeoutSecs <= 0 || program.TimeoutSecs > config.MaxTimeoutSecs {
					program.TimeoutSecs = config.MaxTimeoutSecs
				}

				ctx, cancel := context.WithTimeout(r.Context(), time.Duration(program.TimeoutSecs)*time.Second)
				defer cancel()

				idx := requestIdx.Add(1)

				logger := logger.With(
					zap.Uint64("trace_id", idx),
					zap.Int("index", copiedIndex),
					zap.String("runtime", program.Runtime),
				)

				compilerExitCode, programExitCode, stdout, stderr, err := pythonRuntime.CompileAndRun(ctx, logger, program.Code)

				success := programExitCode == 0 && compilerExitCode == 0 && err == nil

				var compiledPtr *bool
				if compilerExitCode == 0 {
					compiledPtr = newBool(true)
				} else if compilerExitCode != -1 {
					compiledPtr = newBool(false)
				}

				timeoutPtr := newBool(err == context.DeadlineExceeded)

				var exitCodePtr *int
				if programExitCode == 0 {
					exitCodePtr = newInt(0)
				} else if programExitCode != -1 {
					exitCodePtr = newInt(programExitCode)
				}

				var errString *string
				if err != nil {
					errString = newString(err.Error())
				}

				var stdoutPtr *string
				if stdout != "" {
					stdoutPtr = newString(stdout)
				}

				var stderrPtr *string
				if stderr != "" {
					stderrPtr = newString(stderr)
				}

				results[copiedIndex] = ProgramResult{
					Success:  success,
					Compiled: compiledPtr,
					Timeout:  timeoutPtr,
					ExitCode: exitCodePtr,
					Error:    errString,
					Stdout:   stdoutPtr,
					Stderr:   stderrPtr,
				}
			}()
		}

		wg.Wait()

		if err := encode(w, http.StatusOK, ExecuteResponse{
			Results: results,
		}); err != nil {
			logger.Error("encode response", zap.Error(err))
		}
	}))

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))

	return mux
}

func newBool(b bool) *bool {
	return &b
}

func newInt(i int) *int {
	return &i
}

func newString(s string) *string {
	return &s
}

// ErrorResponse is an error returned by the server.
type ErrorResponse struct {
	Error    string            `json:"error"`
	Problems map[string]string `json:"problems,omitempty"`
}

// ExecuteRequest is a request to execute a batch of code.
type ExecuteRequest struct {
	Programs []ProgramProto `json:"programs"`
}

type ProgramProto struct {
	// Runtime is the name of the runtime to use.
	Runtime string `json:"runtime"`
	// Code is the source code to execute.
	Code string `json:"code"`
	// TimeoutSecs is the maximum number of seconds the program is allowed to
	// run.
	TimeoutSecs float64 `json:"timeoutSecs"`
}

type ProgramResult struct {
	// Success is true if the program ran successfully and exited with a zero
	// status code.
	Success bool `json:"success"`
	// Compiled is true if the program was compiled successfully.
	Compiled *bool `json:"compiled"`
	// Timeout is true if the program timed out.
	Timeout *bool `json:"timeout"`
	// ExitCode is the exit code of the program.
	ExitCode *int `json:"exitCode"`
	// Error is the error message if the program failed to run.
	Error *string `json:"error"`
	// Stdout is the standard output from the program.
	Stdout *string `json:"stdout"`
	// Stderr is the standard error output from the program.
	Stderr *string `json:"stderr"`
}

// ExecuteResponse is a response to executing a batch of code.
type ExecuteResponse struct {
	Results []ProgramResult `json:"results"`
}

func (r ExecuteRequest) Valid(ctx context.Context) (problems map[string]string) {
	return nil
}

func encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

type Validator interface {
	// Valid checks the object and returns any problems. If len(problems) == 0
	// then the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

func decodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}

func writeError(w http.ResponseWriter, err error, problems map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:    err.Error(),
		Problems: problems,
	})
}
