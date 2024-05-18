package main

import (
	"encoding/json"
	"fmt"
)

type Runtime string

const (
	RuntimePython310 Runtime = "python3.10"
)

func NewRuntime(s string) (Runtime, error) {
	switch s {
	case "python3.10":
		return RuntimePython310, nil
	default:
		return "", fmt.Errorf("unknown runtime: %s", s)
	}
}

func (r Runtime) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

func (r *Runtime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	val, err := NewRuntime(s)
	if err != nil {
		return err
	}
	*r = val
	return nil
}
