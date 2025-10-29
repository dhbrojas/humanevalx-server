//go:build linux

package main

import "syscall"

// setMemoryLimitOnPid sets the memory limit on an already-running process using prlimit.
// On Linux, this uses the prlimit syscall to set RLIMIT_AS.
func setMemoryLimitOnPid(pid int, maxBytes uint64) error {
	rlimit := syscall.Rlimit{
		Cur: maxBytes,
		Max: maxBytes,
	}

	// Use prlimit to set the limit on the running process
	// syscall.RLIMIT_AS is the address space (virtual memory) limit
	return syscall.Prlimit(pid, syscall.RLIMIT_AS, &rlimit, nil)
}
