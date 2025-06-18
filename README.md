# HumanEval-X Server

A REST API server for executing Python code snippets, designed for code evaluation tasks like HumanEval-X.

## Features

- Execute Python3 code snippets via REST API
- Batch execution of multiple programs in parallel
- Configurable timeouts per program
- Detailed execution results with compilation status, exit codes, and error messages
- Built-in concurrency management
- JSON request/response format

## Installation & Setup

### Prerequisites
- Go 1.21.1 or later
- Python 3.x installed and available in PATH

### Build
```bash
go build -o humanevalx-server
```

### Run
```bash
./humanevalx-server [flags]
```

## Configuration

Command line flags:
- `-host` - Host to listen on (default: "localhost")
- `-port` - Port to listen on (default: "8080")  
- `-max-concurrent-evaluations` - Maximum concurrent evaluations (default: 10)
- `-max-timeout-secs` - Maximum timeout in seconds (default: 60)

Example:
```bash
./humanevalx-server -host 0.0.0.0 -port 9000 -max-timeout-secs 30
```

## API Documentation

### Health Check
**GET /**
- Returns: `OK` (200 status)
- Use this endpoint to verify the server is running

### Execute Code
**POST /v1/execute**

Execute one or more Python code snippets.

#### Request Format
```json
{
  "programs": [
    {
      "runtime": "python3",
      "code": "print('Hello, World!')",
      "timeoutSecs": 10
    }
  ]
}
```

#### Request Fields
- `programs` (array): List of programs to execute
  - `runtime` (string): Must be "python3" (only supported runtime)
  - `code` (string): Python source code to execute
  - `timeoutSecs` (number): Maximum execution time in seconds (optional, defaults to server max)

#### Response Format
```json
{
  "results": [
    {
      "success": true,
      "compiled": true,
      "timeout": false,
      "exitCode": 0,
      "error": null
    }
  ]
}
```

#### Response Fields
- `results` (array): Execution results for each program
  - `success` (boolean): True if program executed successfully with exit code 0
  - `compiled` (boolean|null): True if compilation succeeded, null if not applicable
  - `timeout` (boolean): True if program execution timed out
  - `exitCode` (number|null): Program exit code, null if program didn't start
  - `error` (string|null): Error message if execution failed

#### Error Response
```json
{
  "error": "error description",
  "problems": {
    "field": "validation error"
  }
}
```

## Usage Examples

### Single Program Execution
```bash
curl -X POST http://localhost:8080/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "programs": [{
      "runtime": "python3",
      "code": "print(2 + 2)",
      "timeoutSecs": 5
    }]
  }'
```

### Batch Execution
```bash
curl -X POST http://localhost:8080/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "programs": [
      {
        "runtime": "python3", 
        "code": "print('\''Hello'\'')",
        "timeoutSecs": 5
      },
      {
        "runtime": "python3",
        "code": "import math; print(math.pi)",
        "timeoutSecs": 5
      }
    ]
  }'
```

### HumanEval Example
```bash
curl -X POST http://localhost:8080/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "programs": [{
      "runtime": "python3",
      "code": "def has_close_elements(numbers, threshold):\n    for i, a in enumerate(numbers):\n        for j, b in enumerate(numbers):\n            if i != j and abs(a - b) < threshold:\n                return True\n    return False\n\n# Test\nresult = has_close_elements([1.0, 2.0, 3.0], 0.5)\nprint(result)",
      "timeoutSecs": 10
    }]
  }'
```

## Docker Usage

### Build Docker Image
```bash
docker build -t humanevalx-server .
```

### Run Container
```bash
docker run -p 8080:8080 humanevalx-server
```

## Testing

Run the test suite:
```bash
go test ./...
```

The tests include:
- Runtime execution tests with various Python code snippets
- Integration tests with the full server
- Timeout and error handling tests

## Architecture

- `main.go` - Application entry point and configuration
- `server.go` - HTTP server and API handlers  
- `runtime.go` - Code execution runtime interface
- Python execution via `exec.CommandContext` with timeout support
- Structured logging with Zap
- Concurrent program execution with goroutines

## Known Limitations

- Only supports Python 3.x runtime
- No output capture from executed programs
- No support for multi-file programs or dependencies
- Security relies on system-level Python sandboxing

## License

This project is provided as-is for code evaluation purposes.