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

func humanEvalExampleProblem() ProgramProto {
	prompt := "from typing import List\n\n\ndef has_close_elements(numbers: List[float], threshold: float) -> bool:\n    \"\"\" Check if in given list of numbers, are any two numbers closer to each other than\n    given threshold.\n    >>> has_close_elements([1.0, 2.0, 3.0], 0.5)\n    False\n    >>> has_close_elements([1.0, 2.8, 3.0, 4.0, 5.0, 2.0], 0.3)\n    True\n    \"\"\"\n"
	canonicalSolution := "    for idx, elem in enumerate(numbers):\n        for idx2, elem2 in enumerate(numbers):\n            if idx != idx2:\n                distance = abs(elem - elem2)\n                if distance < threshold:\n                    return True\n\n    return False\n"
	test := "\n\nMETADATA = {\n    'author': 'jt',\n    'dataset': 'test'\n}\n\n\ndef check(candidate):\n    assert candidate([1.0, 2.0, 3.9, 4.0, 5.0, 2.2], 0.3) == True\n    assert candidate([1.0, 2.0, 3.9, 4.0, 5.0, 2.2], 0.05) == False\n    assert candidate([1.0, 2.0, 5.9, 4.0, 5.0], 0.95) == True\n    assert candidate([1.0, 2.0, 5.9, 4.0, 5.0], 0.8) == False\n    assert candidate([1.0, 2.0, 3.0, 4.0, 5.0, 2.0], 0.1) == True\n    assert candidate([1.1, 2.2, 3.1, 4.1, 5.1], 1.0) == True\n    assert candidate([1.1, 2.2, 3.1, 4.1, 5.1], 0.5) == False\n\n"
	main := "if __name__ == '__main__':\n    check(has_close_elements)\n"
	code := fmt.Sprintf("%s%s%s%s", prompt, canonicalSolution, test, main)
	return ProgramProto{
		Runtime:     "python3",
		Code:        code,
		TimeoutSecs: 10,
	}
}

func TestExecuteV1(t *testing.T) {
	is := require.New(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	go run(ctx, os.Stdout, []string{"humanevalx-server", "-host", "localhost", "-port", "8080"})
	is.NoError(waitForReady(t, ctx, 3*time.Second, "http://localhost:8080/"))

	body := &ExecuteRequest{
		Programs: []ProgramProto{},
	}
	body.Programs = append(body.Programs, ProgramProto{
		Runtime:     "python3",
		Code:        "print('hello world')",
		TimeoutSecs: 1,
	})
	body.Programs = append(body.Programs, humanEvalExampleProblem())
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
