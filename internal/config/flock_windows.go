//go:build windows

package config

import (
	"golang.org/x/sys/windows"
)

// flockExclusive acquires an exclusive lock on the file descriptor.
func flockExclusive(fd uintptr) error {
	// Use LockFileEx with LOCKFILE_EXCLUSIVE_LOCK for exclusive access
	// Lock the entire file (offset 0, length 0xFFFFFFFF)
	var overlapped windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(fd),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		0xFFFFFFFF,
		0,
		&overlapped,
	)
}

// flockUnlock releases the lock on the file descriptor.
func flockUnlock(fd uintptr) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(
		windows.Handle(fd),
		0,
		0xFFFFFFFF,
		0,
		&overlapped,
	)
}
