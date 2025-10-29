//go:build linux

package main

import "golang.org/x/sys/unix"

// setMemoryLimitOnPid sets the memory limit on an already-running process using prlimit.
// On Linux, this uses the prlimit syscall to set RLIMIT_AS.
func setMemoryLimitOnPid(pid int, maxBytes uint64) error {
	rlimit := unix.Rlimit{
		Cur: maxBytes,
		Max: maxBytes,
	}

	// Use prlimit to set the limit on the running process
	// unix.RLIMIT_AS is the address space (virtual memory) limit
	return unix.Prlimit(pid, unix.RLIMIT_AS, &rlimit, nil)
}
