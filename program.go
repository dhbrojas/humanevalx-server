package main

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

type Program struct {
	// Runtime is the version of the runtime to use.
	Runtime Runtime `json:"runtime"`
	// Code is the source code to execute.
	Code string `json:"code"`
	// TimeoutSecs is the maximum number of seconds the program is allowed to
	// run.
	TimeoutSecs float64 `json:"timeout_secs"`
}

type ProgramStatus string

const (
	ProgramStatusRunning ProgramStatus = "running"
	ProgramStatusDone    ProgramStatus = "done"
	ProgramStatusError   ProgramStatus = "error"
	ProgramStatusTimeout ProgramStatus = "timeout"
)

type RunningProgram struct {
	Program
	ctx    context.Context
	cancel context.CancelFunc
	status ProgramStatus
	mu     sync.Mutex
	wg     sync.WaitGroup
	cmd    *exec.Cmd
}

func (p *Program) Run(ctx context.Context) (*RunningProgram, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(float64(time.Second)*p.TimeoutSecs))

	rp := &RunningProgram{
		Program: *p,
		ctx:     ctx,
		cancel:  cancel,
		status:  ProgramStatusRunning,
		// TODO: Change once we get more runtimes.
		cmd: exec.CommandContext(ctx, string(p.Runtime), "-c", p.Code),
	}

	rp.cmd.WaitDelay = time.Duration(float64(time.Second) * p.TimeoutSecs)

	if err := rp.cmd.Start(); err != nil {
		cancel()
		rp.status = ProgramStatusError
		return nil, err
	}

	rp.wg.Add(1)
	go func() {
		defer rp.wg.Done()
		defer rp.cancel()

		if err := rp.cmd.Wait(); err != nil {
			rp.mu.Lock()
			defer rp.mu.Unlock()

			if ctx.Err() == context.DeadlineExceeded {
				rp.status = ProgramStatusTimeout
			} else {
				rp.status = ProgramStatusError
			}

			return
		}

		rp.mu.Lock()
		defer rp.mu.Unlock()
		rp.status = ProgramStatusDone
	}()

	return rp, nil
}

func (p *RunningProgram) Status() ProgramStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

func (p *RunningProgram) Wait() ProgramStatus {
	p.wg.Wait()
	// We don't need to lock here because we know the program is done.
	return p.status
}
