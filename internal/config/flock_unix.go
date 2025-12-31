//go:build !windows

package config

import (
	"syscall"
)

// flockExclusive acquires an exclusive lock on the file descriptor.
func flockExclusive(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_EX)
}

// flockUnlock releases the lock on the file descriptor.
func flockUnlock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
