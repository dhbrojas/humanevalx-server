package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

func NewServer(
	logger *zap.Logger,
	config *Config,
) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/v1/execute", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, problems, err := decodeValid[ExecuteRequest](r)
		if err != nil {
			writeError(w, err, problems)
			return
		}

		// Execute programs in parallel, up to the maximum number of concurrent
		// programs.
		wg := sync.WaitGroup{}
		results := make([]ExecutionResult, len(req.Programs))
		for i := range req.Programs {
			copiedIndex := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				program := req.Programs[copiedIndex]
				runningProgram, err := program.Run(r.Context())
				if err != nil {
					results[copiedIndex] = ExecutionResult{
						Success: false,
					}
					logger.Error("program failed to run", zap.Error(err))
					return
				}

				// Wait for the program to finish.
				runningProgram.Wait()
				results[copiedIndex] = ExecutionResult{
					Success: runningProgram.Status() == ProgramStatusDone,
				}
				logger.Info("program finished", zap.String("status", string(runningProgram.Status())), zap.Int("index", copiedIndex))
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

// ErrorResponse is an error returned by the server.
type ErrorResponse struct {
	Error    string            `json:"error"`
	Problems map[string]string `json:"problems,omitempty"`
}

// ExecuteRequest is a request to execute a batch of code.
type ExecuteRequest struct {
	Programs []Program `json:"programs"`
}

type ExecutionResult struct {
	Success bool `json:"Success"`
}

// ExecuteResponse is a response to executing a batch of code.
type ExecuteResponse struct {
	Results []ExecutionResult `json:"results"`
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
	encode(w, http.StatusInternalServerError, ErrorResponse{
		Error:    err.Error(),
		Problems: problems,
	})
}
