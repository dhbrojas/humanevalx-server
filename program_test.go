package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgram(t *testing.T) {
	is := require.New(t)

	p := []*Program{
		{
			Runtime:     RuntimePython310,
			Code:        "import time\ntime.sleep(10)",
			TimeoutSecs: 1,
		},
		{
			Runtime:     RuntimePython310,
			Code:        "print('hello, world')",
			TimeoutSecs: 10,
		},
	}
	statuses := []ProgramStatus{
		ProgramStatusTimeout,
		ProgramStatusDone,
	}

	for i, pp := range p {
		rp, err := pp.Run(context.Background())
		is.NoError(err)
		is.Equal(ProgramStatusRunning, rp.Status())

		rp.Wait()

		is.Equal(statuses[i], rp.Status())

	}
}
