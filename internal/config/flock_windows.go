//go:build windows

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	lockFileExProc   = kernel32.NewProc("LockFileEx")
	unlockFileExProc = kernel32.NewProc("UnlockFileEx")
)

type fileLock struct {
	f *os.File
}

// acquireFileLock acquires an exclusive lock on path + ".lock" via LockFileEx.
// Blocks until the lock is available.
func acquireFileLock(path string) (*fileLock, error) {
	lockPath := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating lock directory: %w", err)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}

	ol := new(syscall.Overlapped)
	// LOCKFILE_EXCLUSIVE_LOCK without LOCKFILE_FAIL_IMMEDIATELY — blocks until
	// the lock is available, matching the Unix LOCK_EX (blocking) behavior.
	const lockfileExclusiveLock = 0x00000002
	r1, _, e1 := lockFileExProc.Call(
		f.Fd(),
		lockfileExclusiveLock,
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		f.Close()
		return nil, fmt.Errorf("acquiring file lock: %w", e1)
	}

	return &fileLock{f: f}, nil
}

// Release releases the file lock.
func (l *fileLock) Release() {
	ol := new(syscall.Overlapped)
	unlockFileExProc.Call(
		l.f.Fd(),
		0,
		1, 0,
		uintptr(unsafe.Pointer(ol)),
	)
	l.f.Close()
}
