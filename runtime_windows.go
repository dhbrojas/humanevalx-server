//go:build windows

package main

import "syscall"

// setMemoryLimitOnPid is a no-op on Windows.
// Windows uses different mechanisms for resource limiting (Job Objects).
func setMemoryLimitOnPid(pid int, maxBytes uint64) error {
	// Not implemented for Windows
	// Would require using Windows Job Objects API
	return syscall.ENOSYS
}
