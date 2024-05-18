package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestExecuteV1(t *testing.T) {
	is := require.New(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	go run(ctx, os.Stdout, []string{"humanevalx-server", "-host", "localhost", "-port", "8080"})
	is.NoError(waitForReady(t, ctx, 3*time.Second, "http://localhost:8080/"))

	body := &ExecuteRequest{
		Programs: []Program{},
	}
	for i := 0; i < 16; i++ {
		body.Programs = append(body.Programs, Program{
			Runtime:     RuntimePython310,
			Code:        fmt.Sprintf("print('hello world %d')", i),
			TimeoutSecs: 1,
		})
	}
	bodyEnc := &bytes.Buffer{}

	is.NoError(json.NewEncoder(bodyEnc).Encode(body))
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://localhost:8080/v1/execute",
		bodyEnc,
	)
	is.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	is.NoError(err)
	is.Equal(http.StatusOK, resp.StatusCode)
	is.Equal("application/json", resp.Header.Get("Content-Type"))

	respBody, err := decode[ExecuteResponse](resp)
	is.NoError(err)
	is.Len(respBody.Results, len(body.Programs))

	for i, result := range respBody.Results {
		is.True(result.Success, "program %d failed", i)
	}
}

func decode[T any](r *http.Response) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// waitForReady calls the specified endpoint until it gets a 200
// response or until the context is cancelled or the timeout is
// reached.
func waitForReady(
	t *testing.T,
	ctx context.Context,
	timeout time.Duration,
	endpoint string,
) error {
	client := http.Client{}
	startTime := time.Now()
	for {
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			endpoint,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			t.Logf("Server ready in %s", time.Since(startTime))
			resp.Body.Close()
			return nil
		}
		resp.Body.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if time.Since(startTime) >= timeout {
				return fmt.Errorf("timeout reached while waiting for endpoint")
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
}
