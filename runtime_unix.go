//go:build darwin

package main

import "syscall"

// setMemoryLimitOnPid sets the memory limit on an already-running process.
// On macOS/Darwin, we don't have prlimit, so this is a best-effort no-op.
// The process would need to set its own limits, which isn't ideal.
func setMemoryLimitOnPid(pid int, maxBytes uint64) error {
	// macOS doesn't support setting rlimits on other processes
	// The child process would need to set its own limits
	// For now, this is unsupported on macOS
	return syscall.ENOSYS
}
