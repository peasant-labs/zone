// lock.go provides flock-based file locking for concurrent access protection.
package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// ErrLockContention is returned when another zone process holds the lock.
var ErrLockContention = errors.New("another zone process is operating on this repo")

// Lock provides an exclusive advisory lock on the .zone/ directory using flock(2).
// It writes a PID file on acquire so that contention errors include the holder's PID,
// and auto-recovers stale locks left by dead processes.
type Lock struct {
	dir  string
	file *os.File
}

// NewLock returns a Lock targeting cacheDir (i.e. .zone/).
func NewLock(cacheDir string) *Lock {
	return &Lock{dir: cacheDir}
}

// Acquire obtains an exclusive non-blocking flock on .zone/.lock.
// On success it writes the current PID to .zone/.lock.pid.
// If the lock is held by a live process, returns ErrLockContention.
// If the lock is held by a dead process (stale), it auto-recovers with a warning.
func (l *Lock) Acquire() error {
	lockPath := filepath.Join(l.dir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		// Lock acquired — write our PID
		l.file = f
		pidPath := filepath.Join(l.dir, ".lock.pid")
		_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
		return nil
	}

	f.Close()

	// Check both EWOULDBLOCK and EAGAIN (cross-kernel compatibility)
	if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
		return fmt.Errorf("flock: %w", err)
	}

	// Lock is held — check if the holder is still alive
	pid := readPIDFile(filepath.Join(l.dir, ".lock.pid"))
	if pid > 0 && isDeadProcess(pid) {
		// Stale lock: clean up and retry once
		_ = os.Remove(filepath.Join(l.dir, ".lock"))
		_ = os.Remove(filepath.Join(l.dir, ".lock.pid"))
		fmt.Fprintf(os.Stderr, "Warning: Recovered stale lock from dead process (PID %d).\n", pid)
		return l.Acquire() // retry once after cleanup
	}

	// Treat pid <= 0 (parse failure, empty file) as stale — auto-recover
	if pid <= 0 {
		_ = os.Remove(filepath.Join(l.dir, ".lock"))
		_ = os.Remove(filepath.Join(l.dir, ".lock.pid"))
		fmt.Fprintf(os.Stderr, "Warning: Recovered stale lock (no valid PID found).\n")
		return l.Acquire()
	}

	return fmt.Errorf("%w (PID %d)", ErrLockContention, pid)
}

// Release unlocks the flock and removes the PID file.
// Safe to call on an unheld Lock (no-op).
func (l *Lock) Release() {
	if l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	l.file.Close()
	l.file = nil
	_ = os.Remove(filepath.Join(l.dir, ".lock.pid"))
}

// IsHeld reports whether this Lock instance currently holds the flock.
func (l *Lock) IsHeld() bool {
	return l.file != nil
}

// ReadLockPID reads the PID from .lock.pid in the given cache directory.
// Returns 0 if the file doesn't exist or can't be parsed.
func ReadLockPID(cacheDir string) int {
	return readPIDFile(filepath.Join(cacheDir, ".lock.pid"))
}

// readPIDFile reads an integer PID from the file at path.
// Returns 0 on any error or if the content is not a valid integer.
func readPIDFile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// isDeadProcess reports whether the process with the given PID no longer exists.
// Uses /proc on Linux and kill(pid, 0) on macOS/other Unix.
func isDeadProcess(pid int) bool {
	if runtime.GOOS == "linux" {
		_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
		return os.IsNotExist(err)
	}
	// macOS + other Unix: kill -0 checks existence without signaling
	err := syscall.Kill(pid, 0)
	return err == syscall.ESRCH
}
